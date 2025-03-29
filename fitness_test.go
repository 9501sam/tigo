package main

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"
)

func TestFitness(t *testing.T) {
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
	probC := CalculateProbability(deploymentConfig, "asus")
	// printJSON(deploymentConfig, "")

	fitness(&traceData, deploymentConfig, processTimeMap, processTimeCloudMap, probC)

	printJSON(&traceData, "fitness.json")
}
