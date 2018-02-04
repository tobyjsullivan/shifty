package main

import (
	"github.com/tobyjsullivan/shifty/qryptos"
	"math"
	"time"
	"fmt"
)

type context struct {
	productDetails *qryptos.ProductDetails
	orders []*qryptos.OrderDetails
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
	positions []*openedPosition
}

type openedPosition struct {
	openingExecutionId int
	closingOrderId int
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

		if desired.Closed(ctx) {
			break
		}

		err = desired.closeOpenPositions(ctx)
		if err != nil {
			fmt.Println("[runBudget]", "error in closeOpenPositions:", err.Error())
			continue
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
		orders: orders,
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
	fmt.Println("[closeOpenPositions]", "Checking for open positions...")

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
			openingExecutionId:execution.ID,
			closingOrderId: orderId,
		})
	}

	return nil
}

func (p *desiredPosition) Closed(ctx *context) bool {
	fmt.Println("[desiredPosition.Closed]", "Checking buy order status.", p.buyOrderId)
	openOrder := ctx.findOrder(p.buyOrderId)
	if openOrder.Status == "live" {
		return false
	}

	for _, pos := range p.positions {
		fmt.Println("[desiredPosition.Closed]", "Checking sell order status.", pos.closingOrderId)
		closeOrder := ctx.findOrder(pos.closingOrderId)
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
	price := math.Max(ctx.productDetails.MarketAsk - 0.00000001, buyPrice * 1.03)

	orderId, err := client.CreateLimitOrder(productId, sideValue, quantity, price)
	if err != nil {
		return 0, err
	}

	fmt.Println("[closePosition]", "Sell order created.", orderId)

	return orderId, nil
}
