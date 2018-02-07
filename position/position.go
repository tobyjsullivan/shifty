package main

import (
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
	"math"
	"time"
)

var (
	buyOrderIds []int
	openedPositions []*position
	closedPositions []*position
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

type position struct {
	openingExecutionId int
	openingPrice       float64
	quantity           float64
	closingOrderId     int
}

func runBudget() {
	fmt.Println("INFO [runBudget] Starting run...")

	ticker := time.NewTicker(loopDelay)
	for range ticker.C {
		fmt.Println("DEBUG [runBudget] Tick.")
		// Load up the current context
		ctx, err := fetchContext()
		if err != nil {
			fmt.Println("ERROR [runBudget]", "error in fetchContext:", err.Error())
			continue
		}

		// Check for any closed positions and remove
		for i, position := range openedPositions {
			closingOrder := ctx.findOrder(position.closingOrderId)
			if closingOrder != nil && closingOrder.Status != "live" {
				closedPositions = append(closedPositions, position)

				// Nifty delete hack (doesn't preserve order)
				openedPositions[i] = openedPositions[len(openedPositions) - 1]
				openedPositions = openedPositions[:len(openedPositions) - 1]

				fmt.Println("INFO [runBudget] Removed closed position. Order #", closingOrder.ID)
			}
		}

		// Check for and record any new open position
		priorExecutionIds := make(map[int]bool)
		for _, position := range openedPositions {
			fmt.Println("DEBUG [runBudget] Found execution in opened positions.", position.openingExecutionId)
			priorExecutionIds[position.openingExecutionId] = true
		}
		for _, position := range closedPositions {
			fmt.Println("DEBUG [runBudget] Found execution in closed positions.", position.openingExecutionId)
			priorExecutionIds[position.openingExecutionId] = true
		}
		for _, buyOrderId := range buyOrderIds {
			buyOrder := ctx.findOrder(buyOrderId)
			if buyOrder == nil {
				fmt.Println("INFO [runBudget] Could not find buyOrder.", buyOrderId)
				continue
			}

			for _, execution := range buyOrder.Executions {
				if !priorExecutionIds[execution.ID] {
					fmt.Println("INFO [runBudget] Detected new opened position from execution.", execution.ID)
					openedPositions = append(openedPositions, &position{
						openingExecutionId: execution.ID,
						openingPrice: execution.Price,
						quantity: execution.Quantity,
					})
				}
			}
		}

		// Compute remaining budget
		remainingBudget := capitalAmount
		for _, position := range openedPositions {
			remainingBudget -= position.quantity * position.openingPrice

			if position.closingOrderId == 0 {
				continue
			}

			// Any filled quantity can be available in budget
			closingOrder := ctx.findOrder(position.closingOrderId)
			if closingOrder != nil {
				remainingBudget += closingOrder.FilledQuantity * position.openingPrice
			}
		}
		fmt.Println("DEBUG [runBudget] Computed remaining budget:", remainingBudget)

		// Update bid with remaining budget by editing order if possible or cancelling and creating a new order
		buyPrice := math.Min(ctx.productDetails.MarketBid + minimalUnit, ctx.productDetails.MarketAsk - minimalUnit)
		buyQuantity := remainingBudget / buyPrice
		var editableBuyOrderFound bool
		shouldUpdateOrder := true
		for _, buyOrderId := range buyOrderIds {
			buyOrder := ctx.findOrder(buyOrderId)
			if buyOrder == nil || buyOrder.Status != "live" {
				continue
			}

			// No need to update if we're already the best price
			if buyOrder.Price >= ctx.productDetails.MarketBid {
				fmt.Println("DEBUG [runBudget] Current buy order is at market bid.", buyOrderId)
				shouldUpdateOrder = false
				continue
			}

			if buyOrder.CanEdit() {
				fmt.Println("INFO [runBudget] Editing buy order.", buyOrderId, "Current market bid:", ctx.productDetails.MarketBid)
				editableBuyOrderFound = true
				err := client.EditOrder(buyOrder.ID, buyQuantity, buyPrice)
				if err != nil {
					fmt.Println("ERROR [runBudget] Error while editing order:", err.Error())
					continue
				}
			} else {
				err := client.CancelOrder(buyOrder.ID)
				if err != nil {
					fmt.Println("ERROR [runBudget] Error while cancelling order:", err.Error())
					continue
				}
			}
		}
		// Create a new buy order if none was found to edit (and there's budget)
		if shouldUpdateOrder && !editableBuyOrderFound && remainingBudget > 0.0 {
			fmt.Println("INFO [runBudget] Creating new order")
			orderId, err := client.CreateLimitOrder(ctx.productDetails.ProductID, "buy", buyQuantity, buyPrice)
			if err != nil {
				fmt.Println("ERROR [runBudget] Error while creating order:", err.Error())
				continue
			}
			buyOrderIds = append(buyOrderIds, orderId)
			fmt.Println("INFO [runBudget] New order created.", orderId)
		}

		// Update any sell orders that are priced above current market ask (cannot go below market ask or we'll compete with our self)
		for _, position := range openedPositions {
			sellOrderId := position.closingOrderId
			if sellOrderId == 0 {
				fmt.Println("INFO [runBudget] Closing position.")
				closingId, err := closePosition(ctx, position.openingPrice * minimumSplit, position.quantity)
				if err != nil {
					fmt.Println("ERROR [runBudget] Error closing position:", err.Error())
					continue
				}
				position.closingOrderId = closingId
				continue
			}
			sellOrder := ctx.findOrder(sellOrderId)
			if sellOrder == nil || !sellOrder.CanEdit() {
				fmt.Println("INFO [runBudget] Either cannot find or cannot edit order.", sellOrderId)
				continue
			}

			mktAsk := ctx.productDetails.MarketAsk
			minAsk := position.openingPrice * minimumSplit

			if mktAsk < minAsk {
				fmt.Println("DEBUG [runBudget] Current market ask is below minimum ask for sell order.", sellOrderId)
			}

			if sellOrder.Price > mktAsk && sellOrder.Price > minAsk {
				price := math.Max(minAsk, mktAsk)
				fmt.Println(fmt.Sprintf(
					"INFO [runBudget] Sell price is %.08f but current market ask is %.08f so editing order",
					sellOrder.Price,
					mktAsk,
				))
				if err := client.EditOrder(sellOrderId, sellOrder.Quantity, price); err != nil {
					fmt.Println("ERROR [runBudget] Error editing sell order:", err.Error())
					continue
				}
			}
		}
	}
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

func closePosition(ctx *context, minPrice, quantity float64) (int, error) {
	fmt.Println("[closePosition]", "Creating sell order...")
	sideValue := "sell"

	productId := ctx.productDetails.ProductID
	price := math.Max(ctx.productDetails.MarketAsk, minPrice)

	orderId, err := client.CreateLimitOrder(productId, sideValue, quantity, price)
	if err != nil {
		return 0, err
	}

	fmt.Println("[closePosition]", "Sell order created.", orderId)

	return orderId, nil
}
