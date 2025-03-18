package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const jaegerBaseURL = "http://localhost:16686/api"

// 定義 JSON 結構
type ServicesResponse struct {
	Data []string `json:"data"`
}

type OperationsResponse struct {
	Data []string `json:"data"`
}

type TraceResponse struct {
	Data []struct {
		Spans []struct {
			SpanID        string `json:"spanID"`
			OperationName string `json:"operationName"`
			Duration      int64  `json:"duration"` // 微秒 (µs)
			References    []struct {
				RefType string `json:"refType"`
				SpanID  string `json:"spanID"`
			} `json:"references"`
		} `json:"spans"`
	} `json:"data"`
}

// 取得所有 service
func getServices() ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/services", jaegerBaseURL))
	if err != nil {
		return nil, fmt.Errorf("error fetching services: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var result ServicesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return result.Data, nil
}

// 取得某個 service 的所有 operations
func getOperations(service string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/services/%s/operations", jaegerBaseURL, service))
	if err != nil {
		return nil, fmt.Errorf("error fetching operations for %s: %v", service, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var result OperationsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return result.Data, nil
}

// 計算某個 API operation 的 self duration
func getOperationSelfDuration(service, operation string) (int64, error) {
	url := fmt.Sprintf("%s/traces?service=%s&operation=%s&limit=10", jaegerBaseURL, service, operation)
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("error fetching traces: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response: %v", err)
	}

	var result TraceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("error parsing JSON: %v", err)
	}

	// 計算 self duration
	var totalSelfDuration int64
	var count int64

	for _, trace := range result.Data {
		spanMap := make(map[string]int64)
		childMap := make(map[string][]string)

		for _, span := range trace.Spans {
			spanMap[span.SpanID] = span.Duration
			for _, ref := range span.References {
				if ref.RefType == "CHILD_OF" {
					childMap[ref.SpanID] = append(childMap[ref.SpanID], span.SpanID)
				}
			}
		}

		for _, span := range trace.Spans {
			if span.OperationName == operation {
				childDuration := int64(0)
				for _, childID := range childMap[span.SpanID] {
					childDuration += spanMap[childID]
				}
				selfDuration := span.Duration - childDuration
				totalSelfDuration += selfDuration
				count++
			}
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("no traces found for %s/%s", service, operation)
	}

	return totalSelfDuration / count, nil
}

// 遍歷所有 service 和 operations，計算 self duration
func main() {
	services, err := getServices()
	if err != nil {
		fmt.Printf("Error getting services: %v\n", err)
		return
	}

	for _, service := range services {
		operations, err := getOperations(service)
		if err != nil {
			fmt.Printf("Error getting operations for %s: %v\n", service, err)
			continue
		}

		for _, operation := range operations {
			selfDuration, err := getOperationSelfDuration(service, operation)
			if err != nil {
				fmt.Printf("Error getting self duration for %s/%s: %v\n", service, operation, err)
				continue
			}
			fmt.Printf("Self Duration for %s/%s: %d µs\n", service, operation, selfDuration)
		}
	}
}
