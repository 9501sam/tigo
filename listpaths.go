package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var services = []string{
	// "cartservice", "checkoutservice", "currencyservice", "emailservice", "frontend",
	// "productcatalogservice", "paymentservice", "recommendationservice", "shippingservice",
	"frontend",
}

const jaegerBaseURL = "http://localhost:16686/api/traces?service=%s&start=%d&end=%d" // Jaeger API URL

type TraceData struct {
	Data []struct {
		TraceID  string `json:"traceID"`
		Duration int64  `json:"duration"` // 加了這個 (in microseconds)
		Spans    []struct {
			OperationName string `json:"operationName"`
			ProcessID     string `json:"processID"`
			StartTime     int64  `json:"startTime"`
			Duration      int64  `json:"duration"`
		} `json:"spans"`
		Processes map[string]struct {
			ServiceName string `json:"serviceName"`
		} `json:"processes"`
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
	start := time.Now().Add(-1*time.Minute).UnixNano() / 1000 // 10 分鐘前
	end := time.Now().UnixNano() / 1000                       // 現在時間

	traces, err := fetchTraces("frontend", start, end)
	if err != nil {
		fmt.Println(err)
	}

	// replace ProcessID with actual service Name
	for i := range traces.Data {
		for j := range traces.Data[i].Spans {
			op := traces.Data[i].Spans[j].OperationName
			if op == "RedisAddItem" || op == "RedisEmptyCart" || op == "RedisGetCart" {
				traces.Data[i].Spans[j].ProcessID = "redis"
			} else if serviceName, ok := traces.Data[i].Processes[traces.Data[i].Spans[j].ProcessID]; ok {
				traces.Data[i].Spans[j].ProcessID = serviceName.ServiceName
			}
		}
	}

	for i := range traces.Data {
		earliestStart := int64(1<<63 - 1) // 最大 int64
		latestEnd := int64(0)

		for j := range traces.Data[i].Spans {
			span := &traces.Data[i].Spans[j]

			if span.StartTime < earliestStart {
				earliestStart = span.StartTime
			}
			endTime := span.StartTime + span.Duration
			if endTime > latestEnd {
				latestEnd = endTime
			}
		}

		traces.Data[i].Duration = latestEnd - earliestStart // 計算 trace duration
	}

	printJSON(traces)
}

func printJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return

	}
	fmt.Println(string(jsonData))
}
