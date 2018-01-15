package main

import (
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
	"math"
	"os"
	"time"
)

const (
	baseCurrency  = "VZT"
	quoteCurrency = "BTC"
	capitalAmount = 0.01
	loopDelay     = 30 * time.Second
)

const (
	sideBuy  = "BUY"
	sideSell = "SELL"
)

var (
	apiTokenId   = os.Getenv("QRYPTOS_API_TOKEN_ID")
	apiSecretKey = os.Getenv("QRYPTOS_API_SECRET_KEY")

	positionQuantity float64
	client           = &qryptos.PrivateClient{
		ApiTokenID:   apiTokenId,
		ApiSecretKey: apiSecretKey,
	}
	buyOrderId    int
	sellOrderId   int
	unitCost      float64
	lastSellPrice float64
)

func main() {
	for {
		go loop()

		time.Sleep(loopDelay)
	}
}

func loop() {
	details := getProductDetails()

	productId := details.ProductID

	if buyOrderId == 0 {
		fmt.Println("[loop] Creating buy order...")
		amount := capitalAmount / details.MarketBid
		positionQuantity = amount
		bid := details.MarketBid + 0.00000001
		if bid >= details.MarketAsk {
			bid = details.MarketBid
		}
		unitCost = bid

		buyOrderId = createOrder(productId, sideBuy, amount, bid)
		fmt.Printf("[loop] Order created: %d\n", buyOrderId)
		return
	} else if !orderFilled(buyOrderId) {
		fmt.Println("[loop] Waiting for buy order to fill.")
		// TODO update bid to current market bid (and adjust quantity)
		return
	}

	fmt.Println("[loop] Buy order filled.")
	return

	if sellOrderId == 0 {
		fmt.Println("[loop] Creating sell order...")
		amount := positionQuantity
		ask := details.MarketAsk - 0.00000001
		if ask <= details.MarketBid {
			ask = details.MarketAsk
		}
		ask = math.Max(ask, unitCost*1.02)
		lastSellPrice = ask

		sellOrderId = createOrder(productId, sideSell, amount, ask)
		fmt.Printf("[loop] Order created: %d\n", sellOrderId)
		return
	} else if !orderFilled(sellOrderId) {
		fmt.Println("[loop] Waiting for sell order to fill.")
		// TODO update ask price based on current markets
		return
	}

	estProfit := (lastSellPrice - unitCost) * positionQuantity

	fmt.Println("[loop] Position is closed! Resetting.")
	resetState()

	fmt.Printf("[loop] Estimated profit was %.08f BTC (+/- fees)\n", estProfit)
}

func getProductDetails() *qryptos.ProductDetails {
	// Get currency details
	allProducts, err := qryptos.DefaultClient().FetchProducts()
	if err != nil {
		panic(err)
	}

	for _, product := range allProducts {
		if product.BaseCurrency == baseCurrency && product.QuotedCurrency == quoteCurrency {
			return product
		}
	}

	panic("Product details not found")
}

func orderFilled(orderId int) bool {
	details, err := client.FetchOrder(orderId)
	if err != nil {
		panic(err)
	}

	return details.Status == "filled"
}

func createOrder(productId int, side string, amount, price float64) int {
	sideValue := "buy"
	if side == sideSell {
		sideValue = "sell"
	}

	orderId, err := client.CreateLimitOrder(productId, sideValue, amount, price)
	if err != nil {
		panic(err)
	}

	return orderId
}

func resetState() {
	buyOrderId = 0
	sellOrderId = 0
	positionQuantity = 0
	unitCost = 0
	lastSellPrice = 0
}
