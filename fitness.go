package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type TraceData struct {
	Data []struct {
		TraceID           string `json:"traceID"`
		Duration          int64  `json:"duration"`          // 微秒 (µs)
		PredictedDuration int64  `json:"predictedDuration"` // 微秒 (µs)
		Spans             []struct {
			OperationName string `json:"operationName"`
			ProcessID     string `json:"processID"`
			StartTime     int64  `json:"startTime"`
			Duration      int64  `json:"duration"`
		} `json:"spans"`
	} `json:"data"`
}

func loadJSONFile[T any](filename string, target *T) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(target)
}

func main() {
	var traceData TraceData
	traceResponse := make(map[string]map[string]int64) // [service][operation] 的 response time

	if err := loadJSONFile("path_durations.json", &traceData); err != nil {
		fmt.Println("Error loading path_durations.json:", err)
		return
	}

	if err := loadJSONFile("self_durations.json", &traceResponse); err != nil {
		fmt.Println("Error loading self_durations.json:", err)
		return
	}

	// printJSON(traceData, "")
	// printJSON(traceResponse, "")

}

func printJSON(data interface{}, fileName string) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return

	}
	fmt.Println(string(jsonData))

	if fileName != "" {
		err = os.WriteFile(fileName, jsonData, 0644)
		if err != nil {
			log.Fatalf("Error writing JSON to file: %v", err)

		}
	}
}
