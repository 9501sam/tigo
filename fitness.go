package main

import (
	"fmt"
	"log"
)

func CalculateProbability(deploymentConfig map[string]map[string]int, targetHost string) map[string]float64 {
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
	probC map[string]float64) float64 {
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
	calculateAverageDuration(traceData)
	return float64(traceData.AveragePredictedDuration)
}
