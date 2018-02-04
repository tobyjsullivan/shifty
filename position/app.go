package main

import (
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
	"os"
	"time"
	"errors"
)

const (
	capitalAmount = 0.01
	loopDelay     = 30 * time.Second
)


var (
	apiTokenId   = os.Getenv("QRYPTOS_API_TOKEN_ID")
	apiSecretKey = os.Getenv("QRYPTOS_API_SECRET_KEY")
	baseCurrency  = os.Getenv("POSITION_BASE_CURRENCY")
	quoteCurrency = os.Getenv("POSITION_QUOTE_CURRENCY")

	client           = &qryptos.PrivateClient{
		ApiTokenID:   apiTokenId,
		ApiSecretKey: apiSecretKey,
	}
)

func main() {
	fmt.Println("[main] Running with token ID:", apiTokenId)

	for {
		err := runBudget()
		if err != nil {
			fmt.Println("[main]", "error in runBudget:", err.Error())
		}
	}
}

func getProductDetails() (*qryptos.ProductDetails, error) {
	// Get currency details
	allProducts, err := qryptos.DefaultClient().FetchProducts()
	if err != nil {
		return nil, err
	}

	for _, product := range allProducts {
		if product.BaseCurrency == baseCurrency && product.QuotedCurrency == quoteCurrency {
			return product, nil
		}
	}

	return nil, errors.New("product details not found")
}
