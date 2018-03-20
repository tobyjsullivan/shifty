package qryptos

import (
	"encoding/json"
	"net/http"
	"strconv"
	"fmt"
)

const (
	qryptosApiBaseUrl = "https://api.qryptos.com"
	productsEndpoint  = "/products"
)

var (
	minOrderQuantities = map[string]Amount {
		"BCH": Amount(1000000),
		"BMC": Amount(100000000),
		"BTC": Amount(100000),
		"DASH": Amount(50000000),
		"DENT": Amount(100000000),
		"DRG": Amount(100000000),
		"ECH": Amount(100000000),
		"ETC": Amount(50000000),
		"ETH": Amount(1000000),
		"ETN": Amount(100000000),
		"LTC": Amount(50000000),
		"QASH": Amount(100000000),
		"STORJ": Amount(100000000),
		"UBTC": Amount(1000000),
		"VET": Amount(10000000),
		"VZT": Amount(100000000),
		"XLM": Amount(50000000),
		"XMR": Amount(50000000),
		"XRP": Amount(50000000),
		"ZEC": Amount(1000000),
	}
)

type PublicClient struct {
}

type ProductDetails struct {
	ProductID        int
	Currency         string
	BaseCurrency     string
	QuotedCurrency   string
	CurrencyPairCode string
	MarketAsk        Amount
	MarketBid        Amount
	Volume24Hour     Amount
	Disabled         bool
}

func DefaultClient() *PublicClient {
	return &PublicClient{}
}

func (c *PublicClient) FetchProducts() ([]*ProductDetails, error) {
	endpoint := qryptosApiBaseUrl + productsEndpoint
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return []*ProductDetails{}, err
	}
	req.Header.Set("X-Quoine-API-Version", "2")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []*ProductDetails{}, err
	}

	var parsedResponse []*productsResponse
	if err := json.NewDecoder(res.Body).Decode(&parsedResponse); err != nil {
		return []*ProductDetails{}, err
	}

	out := make([]*ProductDetails, 0, len(parsedResponse))
	for _, respDetail := range parsedResponse {
		prodId, err := strconv.Atoi(respDetail.ID)
		if err != nil {
			fmt.Println("[FetchProducts] Error parsing product ID:", err.Error())
			return []*ProductDetails{}, err
		}

		if respDetail.MarketAsk == "" {
			continue
		}
		var marketAsk Amount
		fMarketAsk, err := strconv.ParseFloat(respDetail.MarketAsk, 64)
		if err != nil {
			fmt.Println("[FetchProducts] Error parsing MarketAsk:", err.Error())
			return []*ProductDetails{}, err
		}
		marketAsk.FromDecimal(fMarketAsk)

		if respDetail.MarketBid == "" {
			continue
		}
		var marketBid Amount
		fMarketBid, err := strconv.ParseFloat(respDetail.MarketBid, 64)
		if err != nil {
			fmt.Println("[FetchProducts] Error parsing MarketBid:", err.Error())
			return []*ProductDetails{}, err
		}
		marketBid.FromDecimal(fMarketBid)

		var vol24Hr Amount
		fVol24Hr, err := strconv.ParseFloat(respDetail.Volume24Hr, 64)
		if err != nil {
			fmt.Println("[FetchProducts] Error parsing Volume24Hr:", err.Error())
			return []*ProductDetails{}, err
		}
		vol24Hr.FromDecimal(fVol24Hr)

		out = append(out, &ProductDetails{
			ProductID:        prodId,
			Currency:         respDetail.Currency,
			BaseCurrency:     respDetail.BaseCurrency,
			QuotedCurrency:   respDetail.QuotedCurrency,
			CurrencyPairCode: respDetail.CurrencyPairCode,
			MarketAsk:        marketAsk,
			MarketBid:        marketBid,
			Volume24Hour:     vol24Hr,
			Disabled:         respDetail.Disabled,
		})
	}

	return out, nil
}

func MinimumOrderQuantity(currency string) Amount {
	qty, ok := minOrderQuantities[currency]
	if !ok {
		return Amount(1000000)
	}

	return qty
}

type productsResponse struct {
	ID               string `json:"id"`
	Currency         string `json:"currency"`
	CurrencyPairCode string `json:"currency_pair_code"`
	BaseCurrency     string `json:"base_currency"`
	QuotedCurrency   string `json:"quoted_currency"`
	MarketAsk        string `json:"market_ask"`
	MarketBid        string `json:"market_bid"`
	Volume24Hr       string `json:"volume_24h"`
	Disabled         bool   `json:"disabled"`
}
