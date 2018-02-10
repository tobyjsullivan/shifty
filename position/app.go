package main

import (
	"errors"
	"fmt"
	"github.com/tobyjsullivan/shifty/qryptos"
	"os"
	"time"
	"strconv"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	capitalAmount = qryptos.Amount(1000000)
	loopDelay     = 20 * time.Second
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

	productUpdates := make(chan *qryptos.ProductDetails)

	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		sess := session.Must(session.NewSession(aws.NewConfig().WithCredentials(credentials.NewEnvCredentials())))
		cw := cloudwatch.New(sess, aws.NewConfig().WithRegion("us-east-1"))

		go reportMarketMetrics(cw, productUpdates)
	} else {
		fmt.Println("INFO [main] AWS keys not configured.")
	}

	runBudget(productUpdates)
}

func reportMarketMetrics(cw *cloudwatch.CloudWatch, productUpdates chan *qryptos.ProductDetails) {
	fmt.Println("INFO [reportMarketMetrics] Initialized.")
	for details := range productUpdates {
		fmt.Println("DEBUG [reportMarketMetrics] Update received.")

		pairCode := details.CurrencyPairCode
		mktAsk := details.MarketAsk
		mktBid := details.MarketBid
		vol := details.Volume24Hour

		_, err := cw.PutMetricData(&cloudwatch.PutMetricDataInput{
			Namespace: aws.String("Shifty/Position"),
			MetricData: []*cloudwatch.MetricDatum{
				buildMetricDatum(pairCode, "MarketAsk", mktAsk.ToDecimal()),
				buildMetricDatum(pairCode, "MarketBid", mktBid.ToDecimal()),
				buildMetricDatum(pairCode, "Volume24Hr", vol.ToDecimal()),
			},
		})
		if err != nil {
			fmt.Println("ERROR [reportMarketMetrics] Error during PutMetricData:", err.Error())
		}

	}
}

func buildMetricDatum(orderBook, metricName string, value float64) (*cloudwatch.MetricDatum) {
	return &cloudwatch.MetricDatum{
			Dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("OrderBook"),
					Value: aws.String(orderBook),
				},
			},
			MetricName: aws.String(metricName),
			Value: aws.Float64(value),
	}
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
