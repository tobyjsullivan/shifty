package main

import (
	"errors"
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
	"os"
	"time"
)

const (
	capitalAmount = 0.01
	minimumSplit = 1.02
	loopDelay     = 10 * time.Second
)

var (
	apiTokenId    = os.Getenv("QRYPTOS_API_TOKEN_ID")
	apiSecretKey  = os.Getenv("QRYPTOS_API_SECRET_KEY")
	baseCurrency  = os.Getenv("POSITION_BASE_CURRENCY")
	quoteCurrency = os.Getenv("POSITION_QUOTE_CURRENCY")

	client = &qryptos.PrivateClient{
		ApiTokenID:   apiTokenId,
		ApiSecretKey: apiSecretKey,
	}
)

func main() {
	fmt.Println("[main] Running with token ID:", apiTokenId)

	runBudget()
}

func getProductDetails() (*qryptos.ProductDetails, error) {
	// Get currency details
	allProducts, err := qryptos.DefaultClient().FetchProducts()
	if err != nil {
		fmt.Println("[getProductDetails] Error fetching products:", err.Error())
		return nil, err
	}

	for _, product := range allProducts {
		if product.BaseCurrency == baseCurrency && product.QuotedCurrency == quoteCurrency {
			return product, nil
		}
	}

	return nil, errors.New("product details not found")
}
