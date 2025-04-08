package main

import (
	"fmt"
)

type Solution map[string]map[string]int // Solution[node][service] = <replica>

// var traceData TraceData

func InitTIGO() {
	loadJSONFile("path_durations.json", &traceData)
	loadJSONFile("resources_services.json", &serviceConstraints)
	loadJSONFile("resources_nodes.json", &nodeConstraints)
}

// 初始化路由
func initRouting() ([]int, int) {
	// 假設初始 R 是空的，成本為 0
	return []int{}, 0
}

func CopySolution(original Solution) Solution {
	copy := make(Solution)

	for outerKey, innerMap := range original {
		innerCopy := make(map[string]int)
		for innerKey, value := range innerMap {
			innerCopy[innerKey] = value
		}
		copy[outerKey] = innerCopy

	}
	return copy
}

// 雲端執行方案改進
func cloudExecSchemeImprove(solution Solution, BS int) []Solution {
	prevT := evaluate(solution)
	cands := []Solution{}
	for i := range traceData.Data {
		var predictDuration float64 = 0

		L := len(traceData.Data[i].Spans)
		for j := 0; j < L; j++ {
			for k := j; k < L; k++ {
				// build a solution (tempSolution)
				tempSolution := CopySolution(solution)
				onCloudServices := make([]string, 0)
				for t := j; t <= k; t++ {
					onCloudServices := append(onCloudServices, traceData.Data[i].Spans[t].ServiceName)
				}

				for _, service := range services {
					tempSolution["asus"][service] = 0
				}

				for _, service := range onCloudServices {
					tempSolution["asus"][service] = 1
				}

				// evaluate a solution
				tempT := evalutate(tempSolution)
				if tempT < prevT {
					cands := append(cands, tempSolution)
				}
			}
		}
	}

	return cands
}

func calculateNeeded(service string) int {
	totalNumber := 0

	for i := range traceData.Data {
		L := len(traceData.Data[i].Spans)
		for j := 0; j < L; j++ {
			if traceData.Data[i].Spans[j].ServiceName == service {
				totalNumber++
			}
		}
	}

	// TODO: the mu 100 should be better decided
	return totalNumber / 100
}

func bestServer(solution Solution, service string) (string, int) {
	edgeNodes := []string{"vm1", "vm2", "vm3"}

	remaining := make(map[string]int64)
	for _, e := range edgeNodes {
		remaining = nodeConstraint[e].CPU
	}

	for _, node := range nodes {
		for _, service := range services {
			remaining[node] += -(solution[node][service] * serviceConstraints[service])
		}
	}

	var maxKey string
	maxValue := 0
	for key, value := range remaining {
		if value > maxValue {
			maxValue = value
			maxKey = key
		}
	}
	return maxKey, maxValue / nodeConstraint[service].CPU
}

// 邊緣替換策略
func edgeReplacement(solution Solution) Solution {
	onCloudServices := make([]string, 0) // TODO: get from solution
	for service, instanceNumber := range solution["asus"] {
		if instanceNumber != 0 {
			onCloudServices := append(onCloudServices, service)
		}
	}

	// Step 1: Calculate total required instances based on user requests and processing capacity
	requiredInstances := make(map[string]int) // ServiceName -> number of instances needed

	for _, service := range services {
		needed := calculateNeeded(service) // TODO: calculateNeeded
		deployed := 0
		for deployed < needed {
			bestS, maxInstances := bestServer(solution, service) // TODO: bestServer (most CPU)
			count := min(needed-deployed, maxInstances)          // TODO: nodeCapability()
			// TODO: do something to solution
			solution[bestS][service] = count
		}
	}

	return nil
}

func tigo(BS int) Solution {
	tempSls := []Solution{}
	tempSls = append(tempSls, randomSolution())
	var SLs []Solution

	for {
		nextSls := []Solution{}
		for _, sl := range tempSls {
			cands := cloudExecSchemeImprove(sl, BS) // input solution and BS
			if len(cands) == 0 {
				SLs = append(SLs, sl)
			} else {
				for _, cand := range cands {
					Xi1 := edgeReplacement(cand)
					nextSls = append(nextSls, Xi1)
				}
			}
		}

		if len(nextSls) == 0 {
			break
		} else {
			if len(nextSls) > BS {
				nextSls = nextSls[:BS]
			}
			tempSls = nextSls
		}
	}

	return SLs[0]
}

func RunTIGO() {
	InitTIGO()
	BS := 5 // 設定 Branch Search Size
	bestSolution := tigo(BS)
	fmt.Println("Best Solution:", bestSolution)
}
