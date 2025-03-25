package main

import (
	"encoding/json"
	"fmt"
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
	if err := loadJSONFile("fitness.json", &traceData); err != nil {
		fmt.Println("Error loading path_durations.json:", err)
		return
	}

	var totalDuration int64
	var totalPredictedDuration int64
	var count int64

	for _, trace := range traceData.Data {
		totalDuration += trace.Duration
		totalPredictedDuration += trace.PredictedDuration
		count++

	}

	if count > 0 {
		avgDuration := (totalDuration / count) / 1000
		avgPredictedDuration := (totalPredictedDuration / count) / 1000
		fmt.Printf("Average Duration: %d ms\n", avgDuration)
		fmt.Printf("Average Predicted Duration: %d ms\n", avgPredictedDuration)

	} else {
		fmt.Println("No data available to calculate averages.")

	}
}
