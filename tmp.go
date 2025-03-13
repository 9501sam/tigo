package main

import (
	"fmt"
)

// UserRequests represents the request count per node per application
type UserRequests map[int]map[int]float64 // u[j][i] = number of requests for app i on node j

// TransmissionTimes represents the transmission & processing time per app's microservices
type TransmissionTimes map[int][]float64 // T[i] = list of response times between microservices in app i

// ComputeTotalRequests 計算應用程式 i 的總請求數
func ComputeTotalRequests(userRequests UserRequests) map[int]float64 {
	totalRequests := make(map[int]float64)

	for _, appRequests := range userRequests { // 遍歷所有節點 j
		for appID, count := range appRequests { // 遍歷節點上的所有應用 i
			totalRequests[appID] += count
		}
	}
	return totalRequests
}

// ComputeAverageResponseTime 計算系統的平均回應時間 (10)
func ComputeAverageResponseTime(userRequests UserRequests, transmissionTimes TransmissionTimes) float64 {
	totalRequests := ComputeTotalRequests(userRequests)
	overallResponseTime := 0.0

	for appID, times := range transmissionTimes { // 遍歷所有應用 i
		appWeightedTime := 0.0
		for _, time := range times {
			appWeightedTime += time // 累加該應用的所有微服務傳輸 & 處理時間
		}

		// 計算該應用的加權係數
		appTotalReq := totalRequests[appID]
		// for nodeID, requests := range userRequests {
		for _, requests := range userRequests {
			weight := requests[appID] / appTotalReq // u[j,i] / sum_k(u[k,i])
			overallResponseTime += weight * appWeightedTime
		}
	}

	return overallResponseTime
}

func main() {
	// 模擬用戶請求數量 u[j][i]
	userRequests := UserRequests{
		1: {1: 100, 2: 150}, // Node 1: App 1 has 100 requests, App 2 has 150 requests
		2: {1: 200, 2: 250}, // Node 2: App 1 has 200 requests, App 2 has 250 requests
	}

	// 模擬應用間的微服務傳輸 & 處理時間 T[i]
	transmissionTimes := TransmissionTimes{
		1: {1.5, 2.0, 1.2}, // App 1: 微服務 1 → 2: 1.5ms, 2 → 3: 2.0ms, 3 → 4: 1.2ms
		2: {2.5, 1.8, 2.3}, // App 2: 微服務 1 → 2: 2.5ms, 2 → 3: 1.8ms, 3 → 4: 2.3ms
	}

	// 計算系統的平均回應時間
	averageTime := ComputeAverageResponseTime(userRequests, transmissionTimes)
	fmt.Printf("系統的平均回應時間: %.2f ms\n", averageTime)
}
