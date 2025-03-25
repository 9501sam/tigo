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
	selfResponse := make(map[string]map[string]int64)      // [service][operation] 的 response time
	selfResponseCloud := make(map[string]map[string]int64) // [service][operation] 的 response time

	if err := loadJSONFile("path_durations.json", &traceData); err != nil {
		fmt.Println("Error loading path_durations.json:", err)
		return
	}

	if err := loadJSONFile("self_durations.json", &selfResponse); err != nil {
		fmt.Println("Error loading self_durations.json:", err)
		return
	}

	if err := loadJSONFile("self_durationsCloud.json", &selfResponseCloud); err != nil {
		fmt.Println("Error loading self_durationsCloud.json:", err)
		return
	}

	for i := range traceData.Data {
		// TODO: calculate traceData.Data[i].PredictedDuration = .......
		var predictDuration int64 = 0

		// add response time
		for _, span := range traceData.Data[i].Spans {
			if responseTime, ok := selfResponse[span.ProcessID][span.OperationName]; ok {
				if span.ProcessID == "checkoutservice" || span.ProcessID == "currencyservice" {
					predictDuration += (responseTime + selfResponseCloud[span.ProcessID][span.OperationName]) / 2 // TODO: selfResponseCloud
				} else {
					predictDuration += responseTime

				}
			} else {
				fmt.Println("Error: No response time found for operation ", span.OperationName, " in process ", span.ProcessID)
			}
		}

		// TODO: add the network part
		var networkDelay int64 = 50 * 1000
		for _, span := range traceData.Data[i].Spans {
			if span.ProcessID == "checkoutservice" || span.ProcessID == "currencyservice" {
				predictDuration += networkDelay // 來回各一次，機率 1 / 2 所以為一倍 networkDelay
			}
		}

		// Finally
		traceData.Data[i].PredictedDuration = predictDuration

	}
	printJSON(traceData, "fitness.json")
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
