package main

import (
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
	"math"
	"time"
)

type context struct {
	productDetails *qryptos.ProductDetails
	orders         []*qryptos.OrderDetails
}

func (ctx *context) findOrder(orderId int) *qryptos.OrderDetails {
	for _, order := range ctx.orders {
		if order.ID == orderId {
			return order
		}
	}

	return nil
}

type desiredPosition struct {
	buyOrderId int
	positions  []*openedPosition
}

type openedPosition struct {
	openingExecutionId int
	openingPrice       float64
	closingOrderId     int
}

func runBudget() error {
	fmt.Println("[runBudget]", "Starting run...")

	details, err := getProductDetails()
	if err != nil {
		return err
	}

	desired, err := beginPosition(details, capitalAmount)
	if err != nil {
		return err
	}

	for {
		time.Sleep(loopDelay)
		fmt.Println("[runBudget]", "Starting next loop...")

		ctx, err := fetchContext()
		if err != nil {
			fmt.Println("[runBudget]", "error in fetchContext:", err.Error())
			continue
		}

		if err = desired.closeOpenPositions(ctx); err != nil {
			fmt.Println("[runBudget]", "error in closeOpenPositions:", err.Error())
			continue
		}

		// Update buy order if out bid is below current market
		if err = desired.matchMarketBid(ctx); err != nil {
			fmt.Println("[runBudget]", "error in matchMarketBid:", err.Error())
			continue
		}

		// Update sell orders if our ask is above current market
		if err := desired.matchMarketAsk(ctx); err != nil {
			fmt.Println("[runBudget]", "error in matchMarketAsk:", err.Error())
			continue
		}

		if desired.closed(ctx) {
			break
		}
	}

	fmt.Println("[runBudget]", "All positions closed!")

	return nil
}

func fetchContext() (*context, error) {
	details, err := getProductDetails()
	if err != nil {
		return nil, err
	}

	orders, err := client.FetchOrders()
	if err != nil {
		return nil, err
	}

	return &context{
		productDetails: details,
		orders:         orders,
	}, nil
}

func beginPosition(productDetails *qryptos.ProductDetails, budget float64) (*desiredPosition, error) {
	fmt.Println("[beginPosition]", "Creating buy order...")
	sideValue := "buy"

	productId := productDetails.ProductID
	price := productDetails.MarketBid + 0.00000001
	amount := budget / price

	orderId, err := client.CreateLimitOrder(productId, sideValue, amount, price)
	if err != nil {
		return nil, err
	}
	fmt.Println("[beginPosition]", "Buy order created.", orderId)

	return &desiredPosition{
		buyOrderId: orderId,
	}, nil
}

func (p *desiredPosition) closeOpenPositions(ctx *context) error {
	buyOrder := ctx.findOrder(p.buyOrderId)
	executions := buyOrder.Executions

	for _, execution := range executions {
		found := false
		for _, position := range p.positions {
			if position.openingExecutionId == execution.ID {
				found = true
				break
			}
		}

		if found {
			continue
		}
		fmt.Println("[closeOpenPositions]", "New open position found.", execution.ID)

		orderId, err := closePosition(ctx, execution.Price, execution.Quantity)
		if err != nil {
			return err
		}

		p.positions = append(p.positions, &openedPosition{
			openingExecutionId: execution.ID,
			openingPrice:       execution.Price,
			closingOrderId:     orderId,
		})
	}

	return nil
}

func (p *desiredPosition) matchMarketBid(ctx *context) error {
	buyOrder := ctx.findOrder(p.buyOrderId)
	if buyOrder == nil {
		return nil
	}

	mktBid := ctx.productDetails.MarketBid

	if buyOrder.CanEdit() && buyOrder.Price < mktBid {
		fmt.Println(fmt.Sprintf(
			"[matchMarketBid] Buy price is %.08f but current market bid is %.08f so editing order",
			buyOrder.Price,
			mktBid,
		))
		qty := capitalAmount / mktBid
		if err := client.EditOrder(p.buyOrderId, qty, mktBid); err != nil {
			return err
		}
	}

	return nil
}

func (p *desiredPosition) matchMarketAsk(ctx *context) error {
	for _, position := range p.positions {
		sellOrderId := position.closingOrderId
		sellOrder := ctx.findOrder(sellOrderId)
		if sellOrder == nil || !sellOrder.CanEdit() {
			continue
		}

		mktAsk := ctx.productDetails.MarketAsk
		minAsk := position.openingPrice

		if sellOrder.Price > mktAsk && sellOrder.Price > minAsk {
			price := math.Max(minAsk, mktAsk)
			fmt.Println(fmt.Sprintf(
				"[matchMarketAsk] Sell price is %.08f but current market ask is %.08f so editing order",
				sellOrder.Price,
				mktAsk,
			))
			if err := client.EditOrder(sellOrderId, sellOrder.Quantity, price); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *desiredPosition) closed(ctx *context) bool {
	openOrder := ctx.findOrder(p.buyOrderId)
	if openOrder == nil {
		// Assume brand new and data not loaded into context
		return false
	}
	if openOrder.Status == "live" {
		return false
	}

	for _, pos := range p.positions {
		fmt.Println("[desiredPosition.closed]", "Checking sell order status.", pos.closingOrderId)
		closeOrder := ctx.findOrder(pos.closingOrderId)
		if closeOrder == nil {
			// Assume brand new and data not loaded into context
			return false
		}
		if closeOrder.Status == "live" {
			return false
		}
	}

	return true
}

func closePosition(ctx *context, buyPrice, quantity float64) (int, error) {
	fmt.Println("[closePosition]", "Creating sell order...")
	sideValue := "sell"

	productId := ctx.productDetails.ProductID
	price := math.Max(ctx.productDetails.MarketAsk-0.00000001, buyPrice+0.00000001)

	orderId, err := client.CreateLimitOrder(productId, sideValue, quantity, price)
	if err != nil {
		return 0, err
	}

	fmt.Println("[closePosition]", "Sell order created.", orderId)

	return orderId, nil
}
