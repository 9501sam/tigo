package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	// "time"
)

const jaegerURL = "http://localhost:16686/api/traces"

var services = []string{
	"checkoutservice",
	"currencyservice",
	"emailservice",
	"paymentservice",
	"productcatalogservice",
	"recommendationservice",
}

// Jaeger API Response Data Structure
type Trace struct {
	Spans []Span `json:"spans"`
}

type Span struct {
	Duration int64 `json:"duration"` // Duration in microseconds
}

type JaegerResponse struct {
	Data []Trace `json:"data"`
}

// Fetch traces from Jaeger for a given service
func getTraces(service string) ([]Trace, error) {
	url := fmt.Sprintf("%s?service=%s&lookback=1m", jaegerURL, service)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jaegerResp JaegerResponse
	if err := json.Unmarshal(body, &jaegerResp); err != nil {
		return nil, err
	}

	return jaegerResp.Data, nil
}

// Calculate the average response time in milliseconds
func calculateAvgResponseTime(service string) (float64, error) {
	traces, err := getTraces(service)
	if err != nil {
		return 0, err
	}

	var totalDuration int64
	var count int64

	for _, trace := range traces {
		for _, span := range trace.Spans {
			totalDuration += span.Duration // Duration is in microseconds
			count++
		}
	}

	if count == 0 {
		fmt.Println("No traces found")
		return 0, nil // No traces found
	}

	// avgResponseTime := float64(totalDuration) / float64(count) / 1000 // Convert to milliseconds
	avgResponseTime := float64(totalDuration) / float64(count)
	return avgResponseTime, nil
}

func main() {
	fmt.Println("Fetching average response time for services in the past 1 minute...")

	for _, service := range services {
		avgTime, err := calculateAvgResponseTime(service)
		if err != nil {
			fmt.Printf("Error fetching %s: %v\n", service, err)
			continue
		}
		fmt.Printf("%s: %.2f ms\n", service, avgTime)
	}
}
