package main

import (
	"time"
	"github.com/tobyjsullivan/shifty/qryptos"
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	loopDelay = 30 * time.Second
)

func main() {
	fmt.Println("INFO [main] Starting...")
	sess := session.Must(session.NewSession(aws.NewConfig().WithCredentials(credentials.NewEnvCredentials())))
	cw := cloudwatch.New(sess, aws.NewConfig().WithRegion("us-east-1"))

	productBuffer := make(chan *qryptos.ProductDetails, 40)
	go metricsLoop(cw, productBuffer)

	ticker := time.NewTicker(loopDelay)
	for range ticker.C {
		fetchProducts(productBuffer)
	}
}

func fetchProducts(productBuffer chan *qryptos.ProductDetails) {
	fmt.Println("DEBUG [fetchProducts] Fetching products...")
	products, err := qryptos.DefaultClient().FetchProducts()
	if err != nil {
		fmt.Println("ERROR [fetchProducts] Error fetching products:", err.Error())
		return
	}

	fmt.Println("DEBUG [fetchProducts] Product details received.")
	for _, product := range products {
		productBuffer <- product
	}
}

func metricsLoop(cw *cloudwatch.CloudWatch, productBuffer chan *qryptos.ProductDetails) {
	fmt.Println("INFO [metricsLoop] Initialized.")
	for prod := range productBuffer {
		go reportProductMetrics(cw, prod)
	}
}

func reportProductMetrics(cw *cloudwatch.CloudWatch, product *qryptos.ProductDetails) {
	fmt.Println("DEBUG [reportProductMetrics] Reporting product:", product.CurrencyPairCode)
	_, err := cw.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace:  aws.String("Shifty/Position"),
		MetricData: buildMetricData(product),
	})
	if err != nil {
		fmt.Println("ERROR [reportMarketMetrics] Error during PutMetricData:", err.Error())
		return
	}
	fmt.Println("DEBUG [reportProductMetrics] Metrics reported successfully.", product.CurrencyPairCode)
}

func buildMetricData(product *qryptos.ProductDetails) []*cloudwatch.MetricDatum {
	pairCode := product.CurrencyPairCode
	mktAsk := product.MarketAsk
	mktBid := product.MarketBid
	vol := product.Volume24Hour

	return []*cloudwatch.MetricDatum{
		buildMetricDatum(pairCode, "MarketAsk", mktAsk.ToDecimal()),
		buildMetricDatum(pairCode, "MarketBid", mktBid.ToDecimal()),
		buildMetricDatum(pairCode, "Volume24Hr", vol.ToDecimal()),
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
