package qryptos

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/dgrijalva/jwt-go"
)

func TestPrivateClient_FetchOrder(t *testing.T) {
	secretKey := "ZmFrZSBrZXkgc3R1ZmYhIDEyMzQ1Ng=="
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if urlPath := r.URL.Path; urlPath != "/orders/983487134" {
			t.Errorf("Unexpected request path: %s", urlPath)
		}

		if r.Method != http.MethodGet {
			t.Errorf("Unexpected request method: %s", r.Method)
		}

		authHeader := r.Header.Get("X-Quoine-Auth")
		token, err := jwt.Parse(authHeader, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if err != nil {
			t.Fatalf("Error parsing auth header: %s", err.Error())
		}
		if !token.Valid {
			t.Errorf("Invalid auth header: %s", authHeader)
		}

		respBody := `
{
	"id": 983487134,
	"order_type": "limit",
	"quantity": "105.8632",
	"side": "sell",
	"filled_quantity": "0.0",
	"price": 0.00010366,
	"created_at": 1516007675,
	"updated_at": 1516007675,
	"status": "live",
	"product_id": 56,
	"product_code": "CASH",
	"funding_currency": "BTC",
	"currency_pair_code": "VZTBTC",
	"average_price": "0.0",
	"executions": []
}`
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(respBody))
	}))
	defer ts.Close()

	client := &PrivateClient{
		tokenId: "123456",
		secretKey: secretKey,
		apiBaseUrl: ts.URL,
	}

	testOrderId := 983487134
	order, err := client.FetchOrder(testOrderId)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	expectedId := 983487134
	if actualId := order.ID; actualId != expectedId {
		t.Errorf("Unexpected ID. Expected: %d; Actual: %d.", expectedId, actualId)
	}

	expectedPrice := Amount(10366)
	if actualPrice := order.Price; expectedPrice != actualPrice {
		t.Errorf("Unexpected price. Expected: %d; Actual: %d.", expectedPrice, actualPrice)
	}
}

func TestPrivateClient_CreateLimitOrder(t *testing.T) {
	secretKey := "ZmFrZSBrZXkgc3R1ZmYhIDEyMzQ1Ng=="
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if urlPath := r.URL.Path; urlPath != "/orders" {
			t.Errorf("Unexpected request path: %s", urlPath)
		}

		if r.Method != http.MethodPost {
			t.Errorf("Unexpected request method: %s", r.Method)
		}

		authHeader := r.Header.Get("X-Quoine-Auth")
		token, err := jwt.Parse(authHeader, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if err != nil {
			t.Fatalf("Error parsing auth header: %s", err.Error())
		}
		if !token.Valid {
			t.Errorf("Invalid auth header: %s", authHeader)
		}

		respBody := `
{
	"id": 148797141,
	"order_type": "limit",
	"quantity": "231.8068",
	"side": "buy",
	"filled_quantity": "0.0",
	"price": 0.00004754,
	"created_at": 1516007675,
	"updated_at": 1516007675,
	"status": "live",
	"product_id": 4,
	"product_code": "CASH",
	"funding_currency": "BTC",
	"currency_pair_code": "VZTBTC",
	"average_price": "0.0",
	"executions": []
}`
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(respBody))
	}))
	defer ts.Close()

	client := &PrivateClient{
		tokenId: "123456",
		secretKey: secretKey,
		apiBaseUrl: ts.URL,
	}

	orderId, err := client.CreateLimitOrder(4, "buy", Amount(23180680000), Amount(4754))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	expectedId := 148797141
	if orderId != expectedId {
		t.Errorf("Unexpected ID. Expected: %d; Actual: %d.", expectedId, orderId)
	}
}