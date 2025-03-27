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

// should input 1. app(TraceData) 2. deploymentConfig, 3. processing time 4. processing time on cloud
func fitness(traceData TraceData, deploymentConfig map[string]map[string]int,
	selfResponse map[string]map[string]int64, selfResponseCloud map[string]map[string]int64) int64 {
	for i := range traceData.Data {
		var predictDuration int64 = 0

		// add response time
		for _, span := range traceData.Data[i].Spans {
			if responseTime, ok := selfResponse[span.ProcessID][span.OperationName]; ok {
				// add response time (edge or cloud)
			} else {
				fmt.Println("Error: No response time found for operation ", span.OperationName, " in process ", span.ProcessID)
			}
		}

		// the network part
		var networkDelay int64 = 50 * 1000
		for _, span := range traceData.Data[i].Spans {
			// add network delay
		}
		// Finally
		traceData.Data[i].PredictedDuration = predictDuration
	}
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

	fitness(traceData)

	// printJSON(traceData, "fitness.json")
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
