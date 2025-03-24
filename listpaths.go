package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const jaegerURL = "http://localhost:16686/api/traces?service=checkoutservice" // 替換為你的 Service 名稱

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

func main() {
	// 呼叫 Jaeger API 取得 Traces
	resp, err := http.Get(jaegerURL)
	if err != nil {
		fmt.Println("Error fetching traces:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// 解析 JSON
	var traces TraceData
	err = json.Unmarshal(body, &traces)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	// 建立不同 Request Path 的 Map
	paths := make(map[string]bool)

	// 解析每個 Trace
	for _, trace := range traces.Data {
		var path string
		for _, span := range trace.Spans {
			path += fmt.Sprintf(" -> %s(%s)", span.OperationName, span.Process.ServiceName)
		}
		paths[path] = true
	}

	// 輸出所有種類的 Path
	fmt.Println("Unique Paths:")
	for path := range paths {
		fmt.Println(path)
	}
}
