package main

import (
	"github.com/tobyjsullivan/shifty/qryptos"
	"log"
	"os"
	"time"
	"github.com/tobyjsullivan/shifty/tyche/plan"
	"fmt"
)

const (
	envApiKey    = "QRYPTOS_API_TOKEN_ID"
	envApiSecret = "QRYPTOS_API_SECRET_KEY"
	loopDelay    = 10 * time.Second
)

var (
	qryptosApiKey    = os.Getenv(envApiKey)
	qryptosApiSecret = os.Getenv(envApiSecret)
	publicClient     = qryptos.DefaultClient()
	privateClient    = qryptos.NewPrivateClient(qryptosApiKey, qryptosApiSecret)
	productIdLookup  = make(map[int]string)
)

type currencyStatus struct {
	currency       string
	balance        qryptos.Amount
	productDetails *qryptos.ProductDetails
	openOrders     []*qryptos.OrderDetails
}

func main() {
	if qryptosApiKey == "" {
		log.Fatalln("Must set", envApiKey)
	}
	if qryptosApiSecret == "" {
		log.Fatalln("Must set", envApiSecret)
	}

	log.Println("[main] Initializing...")
	products, err := publicClient.FetchProducts()
	if err != nil {
		log.Fatalln("error: failed to fetch products:", err)
	}

	for _, product := range products {
		productIdLookup[product.ProductID] = product.CurrencyPairCode
	}

	ticker := time.NewTicker(loopDelay)
	for range ticker.C {
		log.Println("[main] Triggering loop...")
		go loop()
	}
}

func loop() {
	var p plan.Plan

	log.Println("[loop] Fetching products...")
	products, err := publicClient.FetchProducts()
	if err != nil {
		log.Println("error: failed to fetch products:", err)
		return
	}

	productMap := make(map[string]*qryptos.ProductDetails)
	productIdLookup := make(map[int]string)
	for _, product := range products {
		productMap[product.CurrencyPairCode] = product
		productIdLookup[product.ProductID] = product.CurrencyPairCode
	}

	log.Println("[loop] Fetching balances...")
	acctBalances, err := privateClient.FetchAccountBalances()
	if err != nil {
		log.Println("error: failed to fetch balances:", err)
		return
	}

	balanceMap := make(map[string]qryptos.Amount)
	for _, acctInfo := range acctBalances {
		if acctInfo.Balance == qryptos.AmountZero {
			continue
		}

		balanceMap[acctInfo.Currency] = acctInfo.Balance
	}

	btcBalance := balanceMap["BTC"]

	log.Println("[loop] Fetching orders...")
	orderDetails, err := privateClient.FetchOrders()
	if err != nil {
		log.Println("error: failed to fetch orders:", err)
		return
	}

	// TODO Identify currencies we should be buying (disragarding current orders)
	// Hack: Just hardcoded for now
	buyCurrencies := []string{
		"ETHBTC",
		"LTCBTC",
		"XMRBTC",
		"UBTCBTC",
	}

	// Divy up buy budget
	portion := qryptos.Amount(int(btcBalance) / len(buyCurrencies))
	buyAmounts := make(map[string]qryptos.Amount)
	for _, pairCode := range buyCurrencies {
		buyAmounts[pairCode] = portion
	}

	// For each balance that isn't BTC
	for currency, bal := range balanceMap {
		if currency == "BTC" {
			continue
		}

		pairCode := fmt.Sprintf("%s%s", currency, "BTC")
		product := productMap[pairCode]
		if product.Disabled {
			continue
		}

		// Check the current market ask
		mktAsk := product.MarketAsk

		// Find any open orders for that product
		pendingSells := qryptos.Amount(0)
		for _, order := range orderDetails {
			if order.Status != qryptos.OrderStatusLive {
				continue
			}
			if order.CurrencyPairCode != pairCode {
				continue
			}

			if order.Side != qryptos.OrderSideSell {
				continue
			}

			// Cancel any sell orders with price greater than current marketAsk
			if order.Price > mktAsk {
				p.QueueStep(&CancelOrderStep{order.ID})
				continue
			}

			// Otherwise, count the unfilled amount of the sell order
			pendingSells += order.Quantity
			if _, ok := buyAmounts[product.CurrencyPairCode]; ok {
				buyAmounts[product.CurrencyPairCode] -= order.Quantity.Multiply(mktAsk)
			}
		}

		// Create a new order for any remaining balance
		remBalance := bal - pendingSells
		if min := qryptos.MinimumOrderQuantity(product.BaseCurrency); remBalance < min {
			log.Println("[loop] Quantity too small for sell order. Book:", pairCode, "; Quantity:", remBalance, "; Min:", min)
		} else {
			p.QueueStep(&CreateLimitOrderStep{
				product.ProductID,
				qryptos.OrderSideSell,
				remBalance,
				mktAsk - qryptos.MinimalUnit,
			})
		}
		if _, ok := buyAmounts[product.CurrencyPairCode]; ok {
			buyAmounts[product.CurrencyPairCode] -= remBalance.Multiply(mktAsk)
		}
	}

	// Cancel any current buy orders which are not in our buyList
	for _, order := range orderDetails {
		if order.Status != qryptos.OrderStatusLive || order.Side != qryptos.OrderSideBuy {
			continue
		}

		if buyAmt, wantToBuy := buyAmounts[order.CurrencyPairCode]; wantToBuy {
			// Check if this is already at market bid
			product := productMap[order.CurrencyPairCode]
			desiredQty := buyAmt.Multiply(product.MarketBid)
			cancelThreshold := desiredQty / 2
			if order.Price == product.MarketBid && order.Quantity > cancelThreshold {
				log.Println("[loop] Current buy order for", order.CurrencyPairCode, "is good.")
				buyAmounts[order.CurrencyPairCode] -= order.Quantity.Multiply(order.Price)
				continue
			}
		}

		p.QueueStep(&CancelOrderStep{order.ID})
	}

	var i int
	for pairCode, amt := range buyAmounts {
		i++
		log.Println("[loop] Want to buy", amt, "worth of", pairCode, ". counter:", i)
	}

	// create buy orders
	for pairCode, amount := range buyAmounts {
		product := productMap[pairCode]
		if product.Disabled {
			continue
		}

		bidPrice := product.MarketBid + qryptos.MinimalUnit
		if bidPrice >= product.MarketAsk {
			continue
		}

		quantity := amount.Divide(bidPrice)

		if quantity <= qryptos.MinimumOrderQuantity(product.BaseCurrency) {
			log.Println("[loop] Order too small for", pairCode, ". Quantity:", quantity)
			continue
		}

		p.QueueStep(&CreateLimitOrderStep{
			product.ProductID,
			qryptos.OrderSideBuy,
			quantity,
			bidPrice,
		})
	}

	log.Println("[loop] Finished planning")
	for _, step := range p.Steps {
		fmt.Println("[loop] Planned Step:", step.String())
	}

	p.Apply()
}

type CancelOrderStep struct {
	orderId int
}

func (s *CancelOrderStep) Apply() error {
	return privateClient.CancelOrder(s.orderId)
}

func (s *CancelOrderStep) String() string {
	return fmt.Sprintf("Cancel order %d", s.orderId)
}

type EditOrderStep struct {
	orderId  int
	quantity qryptos.Amount
	price    qryptos.Amount
}

func (s *EditOrderStep) Apply() error {
	return privateClient.EditOrder(s.orderId, s.quantity, s.price)
}

func (s *EditOrderStep) String() string {
	return fmt.Sprintf("Edit order %d. Quantity: %.08f; Price: %.08f",
		s.orderId, s.quantity.ToDecimal(), s.price.ToDecimal())
}

type CreateLimitOrderStep struct {
	productId int
	side      string
	quantity  qryptos.Amount
	price     qryptos.Amount
}

func (s *CreateLimitOrderStep) Apply() error {
	orderId, err := privateClient.CreateLimitOrder(s.productId, s.side, s.quantity, s.price)
	if err != nil {
		log.Println("[CreateLimitOrderStep::Apply] Error creating order:", err)
		return err
	}
	log.Println("[CreateLimitOrderStep::Apply] Order created:", orderId)
	return nil
}

func (s *CreateLimitOrderStep) String() string {
	return fmt.Sprintf("Create limit order. ProductID: %d (%s); Side: %s; Quantity: %.08f; Price: %.08f",
		s.productId, productIdLookup[s.productId], s.side, s.quantity.ToDecimal(), s.price.ToDecimal())
}
