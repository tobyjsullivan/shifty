package main

import (
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
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

type position struct {
	openingExecutionId int
	openingPrice       qryptos.Amount
	quantity           qryptos.Amount
	closingOrderId     int
	closed             bool
}

func runBudget(productUpdates chan *qryptos.ProductDetails) {
	fmt.Println("INFO [runBudget] Starting run...")
	var buyOrderIds []int
	var openedPositions []*position

	ticker := time.NewTicker(loopDelay)
	for range ticker.C {
		fmt.Println("DEBUG [runBudget] Tick.")
		// Load up the current context
		ctx, err := fetchContext()
		if err != nil {
			fmt.Println("ERROR [runBudget]", "error in fetchContext:", err.Error())
			continue
		}

		select {
		case productUpdates <- ctx.productDetails:
			// no-op
		default:
			fmt.Println("DEBUG [runBudget] productUpdates buffer is full.")
		}

		markPositionsClosed(ctx, openedPositions)

		// Check for and record any new open position
		checkForNewPositions(ctx, &openedPositions, buyOrderIds)

		// Compute remaining budget
		remainingBudget := capitalAmount
		for _, position := range openedPositions {
			if position.closed {
				continue
			}

			remainingBudget -= position.quantity.Multiply(position.openingPrice)

			if position.closingOrderId == 0 {
				continue
			}

			// Any filled quantity can be available in budget
			closingOrder := ctx.findOrder(position.closingOrderId)
			if closingOrder != nil {
				remainingBudget += closingOrder.FilledQuantity.Multiply(position.openingPrice)
			}
		}
		fmt.Println("DEBUG [runBudget] Computed remaining budget:", remainingBudget)

		// Update bid with remaining budget by editing order if possible or cancelling and creating a new order
		go updateBuyOrder(ctx, remainingBudget, buyOrderIds)

		// Update any sell orders that are priced above current market ask (cannot go below market ask or we'll compete with our self)
		go updateSellOrders(ctx, openedPositions)
	}
}

func markPositionsClosed(ctx *context, openedPositions []*position) {
	for _, position := range openedPositions {
		closingOrder := ctx.findOrder(position.closingOrderId)
		if closingOrder != nil && closingOrder.Status != qryptos.OrderStatusLive {
			position.closed = true
			fmt.Println("INFO [runBudget] Closed position. Order #", closingOrder.ID)
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

func closePosition(ctx *context, minPrice, quantity qryptos.Amount) (int, error) {
	fmt.Println("[closePosition]", "Creating sell order...")

	productId := ctx.productDetails.ProductID
	price := ctx.productDetails.MarketAsk
	if price < minPrice {
		price = minPrice
	}

	orderId, err := client.CreateLimitOrder(productId, qryptos.OrderSideSell, quantity, price)
	if err != nil {
		return 0, err
	}

	fmt.Println("[closePosition]", "Sell order created.", orderId)

	return orderId, nil
}

func updateBuyOrder(ctx *context, remainingBudget qryptos.Amount, buyOrderIds []int) {
	fmt.Println("DEBUG [runBudget] Managing buy order(s)")
	maxBid := ctx.productDetails.MarketAsk - qryptos.MinimalUnit
	buyPrice := ctx.productDetails.MarketBid
	if buyPrice > maxBid {
		buyPrice = maxBid
	}
	var buyQuantity qryptos.Amount
	buyQuantity.FromDecimal(float64(remainingBudget) / float64(buyPrice))
	var editableBuyOrderFound bool
	shouldUpdateOrder := true
	for _, buyOrderId := range buyOrderIds {
		buyOrder := ctx.findOrder(buyOrderId)
		if buyOrder == nil || buyOrder.Status != qryptos.OrderStatusLive {
			return
		}

		// No need to update if we're already the best price
		// TODO Determine if we're ahead of the pack and, if so, drop back
		if buyOrder.Price >= ctx.productDetails.MarketBid {
			fmt.Println("DEBUG [runBudget] Current buy order is at market bid.", buyOrderId)
			shouldUpdateOrder = false
			return
		}

		if buyOrder.CanEdit() {
			fmt.Println("INFO [runBudget] Editing buy order.", buyOrderId, "Current market bid:", ctx.productDetails.MarketBid)
			editableBuyOrderFound = true
			err := client.EditOrder(buyOrder.ID, buyQuantity, buyPrice)
			if err != nil {
				fmt.Println("ERROR [runBudget] Error while editing order:", err.Error())
				return
			}
		} else {
			fmt.Println("DEBUG [runBudget] Cancelling current buy order.")
			err := client.CancelOrder(buyOrder.ID)
			if err != nil {
				fmt.Println("ERROR [runBudget] Error while cancelling order:", err.Error())
				return
			}
		}
	}
	// Create a new buy order if none was found to edit (and there's budget)
	if shouldUpdateOrder && !editableBuyOrderFound && remainingBudget > 0.0 {
		fmt.Println("INFO [runBudget] Creating new order")
		orderId, err := client.CreateLimitOrder(ctx.productDetails.ProductID, qryptos.OrderSideBuy, buyQuantity, buyPrice)
		if err != nil {
			fmt.Println("ERROR [runBudget] Error while creating buy order:", err.Error())
			return
		}
		buyOrderIds = append(buyOrderIds, orderId)
		fmt.Println("INFO [runBudget] New order created.", orderId)
	}
}

func updateSellOrders(ctx *context, openedPositions []*position) {
	fmt.Println("DEBUG [runBudget] Managing sell orders")
	for i, pos := range openedPositions {
		if pos.closed {
			continue
		}

		sellOrderId := pos.closingOrderId
		if sellOrderId == 0 {
			fmt.Println("INFO [runBudget] Closing position.")
			// Try to merge new position with another so that we don't get stuck with positions that are too small to close
			var mergeCandidate *position
			for j := 0; j < len(openedPositions); j++ {
				if i == j {
					continue
				}

				current := openedPositions[j]
				if current.closed {
					continue
				}
				if current.closingOrderId != 0 {
					if sellOrder := ctx.findOrder(current.closingOrderId); sellOrder == nil || !sellOrder.CanEdit() {
						continue
					}
				}
				if current.openingPrice == pos.openingPrice {
					mergeCandidate = current
					break
				}
			}
			if mergeCandidate != nil {
				// Add this positions quantity to the mergeCandidate
				mergeCandidate.quantity += pos.quantity
				pos.quantity = 0

				// Edit the quantity on the mergeCandidates order
				if mergeCandidate.closingOrderId != 0 {
					sellOrder := ctx.findOrder(mergeCandidate.closingOrderId)
					err := client.EditOrder(mergeCandidate.closingOrderId, mergeCandidate.quantity, sellOrder.Price)
					if err != nil {
						fmt.Println("ERROR [runBudget] Error editing order after position merge:", err.Error())
						continue
					}
				}
				pos.closed = true
				fmt.Println("INFO [runBudget] Merged positions")
				continue
			}

			minPrice := qryptos.Amount(float64(pos.openingPrice) * minimumSplit)
			closingId, err := closePosition(ctx, minPrice, pos.quantity)
			if err != nil {
				fmt.Println("ERROR [runBudget] Error closing position:", err.Error())
				continue
			}
			pos.closingOrderId = closingId
			continue
		}


		mktAsk := ctx.productDetails.MarketAsk
		minAsk := qryptos.Amount(float64(pos.openingPrice) * minimumSplit)
		if mktAsk < minAsk {
			fmt.Println("DEBUG [runBudget] Current market ask is below minimum ask for sell order.", sellOrderId)
		} else {
			continue
		}

		sellOrder := ctx.findOrder(sellOrderId)
		if sellOrder == nil {
			fmt.Println("INFO [runBudget] Cannot find sell order.", sellOrderId)
			continue
		}
		if !sellOrder.CanEdit() {
			fmt.Println("INFO [runBudget] Cannot edit sell order.", sellOrderId)
			continue
		}

		if sellOrder.Price > mktAsk && sellOrder.Price > minAsk {
			price := mktAsk
			if price < minAsk {
				price = minAsk
			}
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

func checkForNewPositions(ctx *context, openedPositions *[]*position, buyOrderIds []int) {
	priorExecutionIds := make(map[int]bool)
	for _, position := range *openedPositions {
		if position.closed {
			continue
		}

		fmt.Println("DEBUG [runBudget] Found execution in opened positions.", position.openingExecutionId)
		priorExecutionIds[position.openingExecutionId] = true
	}
	for _, buyOrderId := range buyOrderIds {
		buyOrder := ctx.findOrder(buyOrderId)
		if buyOrder == nil {
			fmt.Println("DEBUG [runBudget] Could not find buyOrder.", buyOrderId)
			continue
		}

		for _, execution := range buyOrder.Executions {
			if !priorExecutionIds[execution.ID] {
				fmt.Println("INFO [runBudget] Detected new opened position from execution.", execution.ID)
				*openedPositions = append(*openedPositions, &position{
					openingExecutionId: execution.ID,
					openingPrice: execution.Price,
					quantity: execution.Quantity,
				})
			}
		}
	}
}
