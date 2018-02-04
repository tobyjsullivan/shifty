package exchange

type ID interface {}

type Product interface {
	ProductID() ID
	BaseCurrency() string
	QuotedCurrency() string
	MarketAsk() float64
	MarketBid() float64
	Volume24Hour() float64
}

type Order interface {
	OrderID() ID
	Side() string
	Status() string
	Product() Product
}

type Client interface {
	FetchProducts() ([]Product, error)
	FetchOrders() ([]Order, error)
	FetchOrder() (Order, error)
	CreateLimitOrder(Product, side string, quantity, price float64) (Order, error)
}
