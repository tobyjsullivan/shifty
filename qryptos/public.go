package qryptos

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	apiBaseUrl       = "https://api.qryptos.com"
	productsEndpoint = "/products"
)

type PublicClient struct {
}

type ProductDetails struct {
	ProductID        int
	Currency         string
	BaseCurrency     string
	QuotedCurrency   string
	CurrencyPairCode string
	MarketAsk        float64
	MarketBid        float64
	Volume24Hour     float64
	Disabled         bool
}

func DefaultClient() *PublicClient {
	return &PublicClient{}
}

func (c *PublicClient) FetchProducts() ([]*ProductDetails, error) {
	endpoint := apiBaseUrl + productsEndpoint
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

	out := make([]*ProductDetails, len(parsedResponse))
	for i, respDetail := range parsedResponse {
		prodId, err := strconv.Atoi(respDetail.ID)
		if err != nil {
			return []*ProductDetails{}, err
		}

		marketAsk, err := strconv.ParseFloat(respDetail.MarketAsk, 64)
		if err != nil {
			return []*ProductDetails{}, err
		}

		marketBid, err := strconv.ParseFloat(respDetail.MarketBid, 64)
		if err != nil {
			return []*ProductDetails{}, err
		}

		vol24Hr, err := strconv.ParseFloat(respDetail.Volume24Hr, 64)
		if err != nil {
			return []*ProductDetails{}, err
		}

		out[i] = &ProductDetails{
			ProductID:        prodId,
			Currency:         respDetail.Currency,
			BaseCurrency:     respDetail.BaseCurrency,
			QuotedCurrency:   respDetail.QuotedCurrency,
			CurrencyPairCode: respDetail.CurrencyPairCode,
			MarketAsk:        marketAsk,
			MarketBid:        marketBid,
			Volume24Hour:     vol24Hr,
			Disabled:         respDetail.Disabled,
		}
	}

	return out, nil
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
