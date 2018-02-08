package main

import (
	"errors"
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
	"os"
	"time"
	"strconv"
)

const (
	capitalAmount = qryptos.Amount(1000000)
	loopDelay     = 10 * time.Second
)

var (
	apiTokenId    = os.Getenv("QRYPTOS_API_TOKEN_ID")
	apiSecretKey  = os.Getenv("QRYPTOS_API_SECRET_KEY")
	baseCurrency  = os.Getenv("POSITION_BASE_CURRENCY")
	quoteCurrency = os.Getenv("POSITION_QUOTE_CURRENCY")
	minimumSplit = 1.01

	client = qryptos.NewPrivateClient(apiTokenId, apiSecretKey)
)

func init() {
	minSplitVar := os.Getenv("MIN_SPLIT")
	if minSplitVar != "" {
		var err error
		minimumSplit, err = strconv.ParseFloat(minSplitVar, 64)
		if err != nil {
			panic("Error parsing MIN_SPLIT: "+err.Error())
		}
	}
}

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
