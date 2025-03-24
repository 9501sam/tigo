package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var services = []string{
	"cartservice", "checkoutservice", "currencyservice", "emailservice", "frontend",
	"productcatalogservice", "paymentservice", "recommendationservice", "shippingservice",
}

const jaegerBaseURL = "http://localhost:16686/api/traces?service=%s&start=%d&end=%d" // Jaeger API URL

type TraceData struct {
	Data []struct {
		TraceID string `json:"traceID"`
		Spans   []struct {
			OperationName string `json:"operationName"`
			Process       struct {
				ServiceName string `json:"serviceName"`
			} `json:"process"`
		} `json:"spans"`
	} `json:"data"`
}

func fetchTraces(service string, start, end int64) (*TraceData, error) {
	url := fmt.Sprintf(jaegerBaseURL, service, start, end)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching traces for %s: %w", service, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body for %s: %w", service, err)
	}

	var traces TraceData
	if err := json.Unmarshal(body, &traces); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON for %s: %w", service, err)
	}

	return &traces, nil
}

func main() {
	start := time.Now().Add(-10*time.Minute).UnixNano() / 1000 // 10 分鐘前
	end := time.Now().UnixNano() / 1000                        // 現在時間

	paths := make(map[string]bool)

	for _, service := range services {
		traces, err := fetchTraces(service, start, end)
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, trace := range traces.Data {
			var path string
			for _, span := range trace.Spans {
				path += fmt.Sprintf(" -> %s(%s)", span.OperationName, span.Process.ServiceName)
			}
			paths[path] = true
		}
	}

	fmt.Println("Unique Paths:")
	for path := range paths {
		fmt.Println(path)
	}
}
