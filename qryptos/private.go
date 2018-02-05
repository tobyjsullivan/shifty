package qryptos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	ordersEndpoint = "/orders"
)

type PrivateClient struct {
	ApiTokenID   string
	ApiSecretKey string
}

type OrderDetails struct {
	ID               int
	Side             string
	Status           string
	CurrencyPairCode string
	Price			 float64
	Quantity         float64
	FilledQuantity   float64
	Executions		 []*ExecutionDetails
}

type ExecutionDetails struct {
	ID int
	Quantity float64
	Price float64
}

func (c *PrivateClient) generateJWT(uri *url.URL) (string, error) {
	nonce := time.Now().UnixNano() / 1000000
	path := uri.Path

	if uri.RawQuery != "" {
		path += "?" + uri.RawQuery
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"path":     path,
		"nonce":    strconv.FormatInt(nonce, 10),
		"token_id": c.ApiTokenID,
	})

	tokenString, err := token.SignedString([]byte(c.ApiSecretKey))

	return tokenString, err
}

type ordersResponse struct {
	Models []*orderResponse `json:"models"`
}

type orderResponse struct {
	ID               int    `json:"id"`
	Side             string `json:"side"`
	Status           string `json:"status"`
	CurrencyPairCode string `json:"currency_pair_code"`
	Price 			 float64 `json:"price"`
	Quantity         string `json:"quantity"`
	FilledQuantity   string `json:"filled_quantity"`
	Executions	[]*executionResponse `json:"executions"`
}

type executionResponse struct {
	ID int `json:"id"`
	Quantity string `json:"quantity"`
	Price string `json:"price"`
	TakerSide string `json:"taker_side"`
	MySide string `json:"my_side"`
}

func (c *PrivateClient) FetchOrders() ([]*OrderDetails, error) {
	endpoint := apiBaseUrl + ordersEndpoint
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return []*OrderDetails{}, err
	}

	q := req.URL.Query()
	q.Set("limit", "100")
	q.Set("with_details", "1")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("X-Quoine-API-Version", "2")

	token, err := c.generateJWT(req.URL)
	if err != nil {
		return []*OrderDetails{}, err
	}

	req.Header.Set("X-Quoine-Auth", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []*OrderDetails{}, err
	}

	var parsedResponse ordersResponse
	if err := json.NewDecoder(res.Body).Decode(&parsedResponse); err != nil {
		return []*OrderDetails{}, err
	}

	out := make([]*OrderDetails, len(parsedResponse.Models))
	for i, model := range parsedResponse.Models {
		fmt.Println("[FetchOrders] ID:", model.ID)
		out[i], err = parseOrderDetails(model)
		if err != nil {
			return []*OrderDetails{}, err
		}
	}

	return out, nil
}

func (c *PrivateClient) FetchOrder(orderId int) (*OrderDetails, error) {
	endpoint := fmt.Sprintf("%s%s/%d", apiBaseUrl, ordersEndpoint, orderId)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	req.Header.Set("X-Quoine-API-Version", "2")

	token, err := c.generateJWT(req.URL)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Quoine-Auth", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var parsedResponse orderResponse
	if err := json.NewDecoder(res.Body).Decode(&parsedResponse); err != nil {
		return nil, err
	}

	return parseOrderDetails(&parsedResponse)
}

func (c *PrivateClient) CreateLimitOrder(productId int, side string, quantity, price float64) (int, error) {
	fmt.Println("[CreateLimitOrder] Creating order...")

	qtyString := fmt.Sprintf("%.08f", quantity)
	priceString := fmt.Sprintf("%.08f", price)

	payload := &fmtCreateOrder{
		Order: &fmtCreateOrderModel{
			OrderType: "limit",
			ProductID: productId,
			Side:      side,
			Quantity:  qtyString,
			Price:     priceString,
		},
	}

	bodyString, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	fmt.Printf("[CreateLimitOrder] Body: %s\n", bodyString)

	endpoint := apiBaseUrl + ordersEndpoint
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(bodyString)))
	if err != nil {
		return 0, err
	}

	req.Header.Set("X-Quoine-API-Version", "2")
	req.Header.Set("Content-Type", "application/json")

	token, err := c.generateJWT(req.URL)
	if err != nil {
		return 0, err
	}

	req.Header.Set("X-Quoine-Auth", token)

	fmt.Printf("[CreateLimitOrder] URL: %s\n", req.URL.String())

	//var res *http.Response
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	if res.StatusCode != 200 {
		var buf bytes.Buffer
		buf.ReadFrom(res.Body)

		fmt.Printf("[CreateLimitOrder] Error: %s\n", buf)

		return 0, errors.New(fmt.Sprintf("unexpected status: %d", res.StatusCode))
	}

	var parsedRes struct {
		ID int `json:"id"`
	}

	err = json.NewDecoder(res.Body).Decode(&parsedRes)
	if err != nil {
		return 0, err
	}

	fmt.Printf("[CreateLimitOrder] Created successfully: %d\n", parsedRes.ID)

	return parsedRes.ID, nil
}

func (c *PrivateClient) EditOrder(orderId int, quantity, price float64) error {
	fmt.Println("[EditOrder]", "Updating order:", orderId)

	qtyString := fmt.Sprintf("%.08f", quantity)
	priceString := fmt.Sprintf("%.08f", price)

	payload := &fmtEditOrder{
		Order: &fmtEditOrderModel{
			Quantity: qtyString,
			Price: priceString,
		},
	}

	bodyString, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s%s/%d", apiBaseUrl, ordersEndpoint, orderId)
	req, err := http.NewRequest(http.MethodPut, endpoint, strings.NewReader(string(bodyString)))
	if err != nil {
		return err
	}

	req.Header.Set("X-Quoine-API-Version", "2")
	req.Header.Set("Content-Type", "application/json")

	token, err := c.generateJWT(req.URL)
	if err != nil {
		return err
	}

	req.Header.Set("X-Quoine-Auth", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		var buf bytes.Buffer
		buf.ReadFrom(res.Body)

		fmt.Printf("[EditOrder] Error: %s\n", buf)

		return errors.New(fmt.Sprintf("unexpected status: %d", res.StatusCode))
	}

	return nil
}

func (o *OrderDetails) CanEdit() bool {
	return o.Status == "live" && o.FilledQuantity == 0.0
}

type fmtCreateOrder struct {
	Order *fmtCreateOrderModel `json:"order"`
}

type fmtCreateOrderModel struct {
	OrderType string `json:"order_type"`
	ProductID int    `json:"product_id"`
	Side      string `json:"side"`
	Quantity  string `json:"quantity"`
	Price     string `json:"price"`
}

type fmtEditOrder struct {
	Order *fmtEditOrderModel `json:"order"`
}

type fmtEditOrderModel struct{
	Quantity string `json:"quantity"`
	Price string `json:"price"`
}

func parseExecutions(input []*executionResponse) ([]*ExecutionDetails, error) {
	var out []*ExecutionDetails

	for _, resp := range input {
		quantity, err := strconv.ParseFloat(resp.Quantity, 64)
		if err != nil {
			return []*ExecutionDetails{}, err
		}

		price, err := strconv.ParseFloat(resp.Price, 64)
		if err != nil {
			return []*ExecutionDetails{}, err
		}

		out = append(out, &ExecutionDetails{
			ID: resp.ID,
			Quantity: quantity,
			Price: price,
		})
	}

	return out, nil
}

func parseOrderDetails(input *orderResponse) (*OrderDetails, error) {
	executions, err := parseExecutions(input.Executions)
	if err != nil {
		return nil, err
	}

	quantity, err := strconv.ParseFloat(input.Quantity, 64)
	if err != nil {
		return nil, err
	}

	filledQty, err := strconv.ParseFloat(input.FilledQuantity, 64)
	if err != nil {
		return nil, err
	}

	return &OrderDetails{
		ID:               input.ID,
		Side:             input.Side,
		Status:           input.Status,
		CurrencyPairCode: input.CurrencyPairCode,
		Price:            input.Price,
		Quantity:         quantity,
		FilledQuantity:   filledQty,
		Executions: 	  executions,
	}, nil
}