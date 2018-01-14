package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
)

const qryptosApiUrl = "https://api.qryptos.com"

func main() {
	endpoint := qryptosApiUrl + "/products"
	res, err := http.DefaultClient.Get(endpoint)
	if err != nil {
		panic(err)
	}

	var parsedResponse []*productDetails
	if err := json.NewDecoder(res.Body).Decode(&parsedResponse); err != nil {
		panic(err)
	}

	var reports []*report

	for _, prodData := range parsedResponse {
		if prodData.Currency != "BTC" {
			continue
		}

		r := buildReport(prodData)
		printReport(r)

		reports = append(reports, r)
	}

	sort.Sort(Reports(reports))

	fmt.Println("TOP 3:")
	for i := 0; i < 3; i++ {
		printReport(reports[len(reports)-(i+1)])
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
	bid           float32
	ask           float32
	spread        float32
	volume24Hr    float32
	volume24HrBtc float32
	weight        float32
}

func buildReport(details *productDetails) *report {
	vol, err := strconv.ParseFloat(details.Volume24Hr, 10)
	if err != nil {
		panic(err)
	}

	spread := details.MarketAsk - details.MarketBid
	currentRate := (details.MarketBid + details.MarketAsk) / 2.0
	volume24HrBtc := float32(vol) * currentRate

	return &report{
		currencyPair:  details.CurrencyPairCode,
		bid:           details.MarketBid,
		ask:           details.MarketAsk,
		spread:        spread,
		volume24Hr:    float32(vol),
		volume24HrBtc: volume24HrBtc,
		weight:        spread * float32(vol),
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
