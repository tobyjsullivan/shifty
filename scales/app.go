package main

import (
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
	"sort"
)

const qryptosApiUrl = "https://api.qryptos.com"
const availableCapital = 0.03
const topN = 5

func main() {
	productDetails, err := qryptos.DefaultClient().FetchProducts()
	if err != nil {
		panic(err)
	}

	var reports []*report

	for _, prodData := range productDetails {
		if prodData.Disabled || prodData.Currency != "BTC" || prodData.BaseCurrency == "XRP" {
			continue
		}

		r := buildReport(prodData)
		printReport(r)

		reports = append(reports, r)
	}

	sort.Sort(Reports(reports))

	fmt.Println(fmt.Sprintf("TOP %d:", topN))
	topReports := make([]*report, topN)
	totalWeight := 0.0
	for i := 0; i < topN; i++ {
		r := reports[len(reports)-(i+1)]
		printReport(r)
		topReports[i] = r

		totalWeight += r.weight
	}

	fmt.Println("Recommended Orders:")

	for _, r := range topReports {
		offset := 0.0
		bidAmount := r.bid + offset
		askAmount := r.ask - offset

		capitalProportion := availableCapital * (r.weight / totalWeight)

		quantity := capitalProportion / bidAmount

		fmt.Printf("BUY  %s; Bid: %.08f; Quantity: %.04f; Risk: %.08f\n", r.currencyPair, bidAmount, quantity, quantity*bidAmount)
		fmt.Printf("SELL %s; Ask: %.08f; Quantity: %.04f\n", r.currencyPair, askAmount, quantity)
	}
}

type productDetails struct {
	ID               string  `json:"id"`
	Currency         string  `json:"currency"`
	CurrencyPairCode string  `json:"currency_pair_code"`
	MarketAsk        float32 `json:"market_ask"`
	MarketBid        float32 `json:"market_bid"`
	Volume24Hr       string  `json:"volume_24h"`
}

type report struct {
	currencyPair  string
	bid           float64
	ask           float64
	spread        float64
	volume24Hr    float64
	volume24HrBtc float64
	weight        float64
}

func buildReport(details *qryptos.ProductDetails) *report {

	spread := details.MarketAsk - details.MarketBid
	currentRate := (details.MarketBid + details.MarketAsk) / 2.0
	volume24HrBtc := details.Volume24Hour * currentRate

	return &report{
		currencyPair:  details.CurrencyPairCode,
		bid:           details.MarketBid.ToDecimal(),
		ask:           details.MarketAsk.ToDecimal(),
		spread:        spread.ToDecimal(),
		volume24Hr:    details.Volume24Hour.ToDecimal(),
		volume24HrBtc: volume24HrBtc.ToDecimal(),
		weight:        spread.ToDecimal() * details.Volume24Hour.ToDecimal(),
	}
}

func printReport(r *report) {
	fmt.Println(r.currencyPair)
	fmt.Println(fmt.Sprintf("- Bid: %.08f", r.bid))
	fmt.Println(fmt.Sprintf("- Ask: %.08f", r.ask))
	fmt.Println(fmt.Sprintf("- Spread: %.08f", r.spread))
	fmt.Println(fmt.Sprintf("- Volume: %.08f", r.volume24Hr))
	fmt.Println(fmt.Sprintf("- Volume (BTC): %.08f", r.volume24HrBtc))
	fmt.Println(fmt.Sprintf("- Weight: %.08f", r.weight))

	fmt.Println()
}

type Reports []*report

func (s Reports) Len() int           { return len(s) }
func (s Reports) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Reports) Less(i, j int) bool { return s[i].weight < s[j].weight }
