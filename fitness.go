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

func calculateProbability(deploymentConfig map[string]map[string]int, targetHost string) map[string]float64 {
	// Initialize map to store total amounts of each service
	totalServiceAmounts := make(map[string]int)

	// Sum up total amounts of each service across all hosts
	for _, hostServices := range deploymentConfig {
		for service, amount := range hostServices {
			totalServiceAmounts[service] += amount
		}
	}

	// Initialize map to store ratios
	ratios := make(map[string]float64)

	// Check if target host exists
	if _, ok := deploymentConfig[targetHost]; !ok {
		log.Printf("Target host '%s' not found in deployment config.\n", targetHost)
		return ratios

	}

	// Calculate ratios for each service on the target host
	for service, totalAmount := range totalServiceAmounts {
		targetAmount := deploymentConfig[targetHost][service]
		if totalAmount > 0 {
			ratios[service] = float64(targetAmount) / float64(totalAmount)
		} else {
			ratios[service] = 0 // Avoid division by zero
		}
	}
	return ratios
}

// should input 1. app(TraceData) 2. deploymentConfig, 3. processing time 4. processing time on cloud
func fitness(traceData *TraceData, deploymentConfig map[string]map[string]int,
	processTimeMap map[string]map[string]int64, processTimeCloudMap map[string]map[string]int64,
	probC map[string]float64) int64 {
	for i := range traceData.Data {
		var predictDuration float64 = 0

		// add response time
		for _, span := range traceData.Data[i].Spans {
			var processTimeOnEdge int64
			var processTimeOnCloud int64
			var ok bool

			if processTimeOnEdge, ok = processTimeMap[span.ProcessID][span.OperationName]; ok {
			} else {
				fmt.Println("Error: No response time found for operation ", span.OperationName, " in process ", span.ProcessID)
			}

			if processTimeOnEdge, ok = processTimeCloudMap[span.ProcessID][span.OperationName]; ok {
			} else {
				fmt.Println("Error: No response time found for operation ", span.OperationName, " in process ", span.ProcessID)
			}

			// add response time (edge or cloud)
			predictDuration += probC[span.ProcessID]*float64(processTimeOnCloud) + (1-probC[span.ProcessID])*float64(processTimeOnEdge)
		}

		// TODO: might be more accurate
		var networkDelay float64 = 50 * 1000 // 50 ms
		for _, span := range traceData.Data[i].Spans {
			if span.ProcessID != "frontend" {
				predictDuration += networkDelay
			}
		}

		// Finally
		traceData.Data[i].PredictedDuration = int64(predictDuration)
	}
	return 0
}

func main() {
	var traceData TraceData
	processTimeMap := make(map[string]map[string]int64)      // [service][operation] 的 process time
	processTimeCloudMap := make(map[string]map[string]int64) // [service][operation] 的 process time

	if err := loadJSONFile("path_durations.json", &traceData); err != nil {
		fmt.Println("Error loading path_durations.json:", err)
		return
	}

	if err := loadJSONFile("self_durations.json", &processTimeMap); err != nil {
		fmt.Println("Error loading self_durations.json:", err)
		return
	}

	if err := loadJSONFile("self_durationsCloud.json", &processTimeCloudMap); err != nil {
		fmt.Println("Error loading self_durationsCloud.json:", err)
		return
	}

	var jsonStr = `{
		"vm1": {
			"cartservice": 1,
			"checkoutservice": 1,
			"currencyservice": 1,
			"emailservice": 0,
			"frontend": 0,
			"paymentservice": 0,
			"productcatalogservice": 0,
			"recommendationservice": 0,
			"redis-cart": 0,
			"shippingservice": 0
		},
		"vm2": {
			"cartservice": 0,
			"checkoutservice": 0,
			"currencyservice": 0,
			"emailservice": 1,
			"frontend": 1,
			"paymentservice": 1,
			"productcatalogservice": 0,
			"recommendationservice": 0,
			"redis-cart": 0,
			"shippingservice": 0
		},
		"vm3": {
			"cartservice": 0,
			"checkoutservice": 0,
			"currencyservice": 0,
			"emailservice": 0,
			"frontend": 0,
			"paymentservice": 0,
			"productcatalogservice": 1,
			"recommendationservice": 1,
			"redis-cart": 1,
			"shippingservice": 1
		},
		"asus": {
			"cartservice": 0,
			"checkoutservice": 0,
			"currencyservice": 0,
			"emailservice": 3,
			"frontend": 0,
			"paymentservice": 0,
			"productcatalogservice": 0,
			"recommendationservice": 0,
			"redis-cart": 0,
			"shippingservice": 1
		}
	}`

	var deploymentConfig map[string]map[string]int
	err := json.Unmarshal([]byte(jsonStr), &deploymentConfig)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}
	probC := calculateProbability(deploymentConfig, "asus")
	// printJSON(deploymentConfig, "")

	fitness(&traceData, deploymentConfig, processTimeMap, processTimeCloudMap, probC)

	printJSON(&traceData, "fitness.json")
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
