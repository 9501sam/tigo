package main

import (
	"fmt"
)

type Solution map[string]map[string]int // Solution[node][service] = <replica>

// var traceData TraceData

func InitTIGO() {
	loadJSONFile("app.json", &traceData)
	loadJSONFile("resources_services.json", &serviceConstraints)
	loadJSONFile("resources_nodes.json", &nodeConstraints)
	loadJSONFile("processing_time_edge.json", &processTimeMap)
	loadJSONFile("processing_time_cloud.json", &processTimeCloudMap)
}

// 初始化路由
func initRouting() ([]int, int) {
	// 假設初始 R 是空的，成本為 0
	return []int{}, 0
}

func CopySolution(original Solution) Solution {
	copy := make(Solution)
	for _, node := range nodes {
		copy[node] = make(map[string]int)
	}

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
		// var predictDuration float64 = 0

		L := len(traceData.Data[i].Spans)
		for j := 0; j < L; j++ {
			for k := j; k < L; k++ {
				// build a solution (tempSolution)
				tempSolution := CopySolution(solution)
				onCloudServices := make([]string, 0)
				for t := j; t <= k; t++ {
					onCloudServices = append(onCloudServices, traceData.Data[i].Spans[t].ServiceName)
				}

				for _, service := range services {
					tempSolution["asus"][service] = 0
				}

				for _, service := range onCloudServices {
					tempSolution["asus"][service] = 1
				}

				// evaluate a solution
				tempT := evaluate(tempSolution)
				if tempT < prevT {
					cands = append(cands, tempSolution)
				}
			}
		}
	}

	if len(cands) < BS {
		return cands
	}
	return cands[0:BS]
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
	return totalNumber / 50
}

func bestServer(solution Solution, service string) (string, int64) {
	edgeNodes := []string{"vm1", "vm2", "vm3"}

	remaining := make(map[string]int64)
	for _, e := range edgeNodes {
		remaining[e] = int64(nodeConstraints[e].CPU)
	}

	for _, node := range nodes {
		for _, service := range services {
			remaining[node] += -(int64(solution[node][service]) * int64(serviceConstraints[service].CPU))
		}
	}

	var maxKey string
	maxValue := int64(0)
	for key, value := range remaining {
		if value > maxValue {
			maxValue = value
			maxKey = key
		}
	}
	return maxKey, maxValue / int64(serviceConstraints[service].CPU)
}

// 邊緣替換策略
func edgeReplacement(solution Solution) Solution {
	retSolution := CopySolution(solution)

	for _, service := range services {
		needed := calculateNeeded(service) // TODO: calculateNeeded
		deployed := 0
		var prevBestS string
		for deployed < needed {
			fmt.Printf("\niteration start\n")
			fmt.Printf("deployed = %d, needed = %d, service = %s\n", deployed, needed, service)

			bestS, maxInstances := bestServer(retSolution, service)
			fmt.Printf("bestS = %s, maxInstances = %d\n", bestS, maxInstances)

			if prevBestS == bestS {
				break
			}
			prevBestS = bestS
			count := min(int64(needed-deployed), maxInstances) // TODO: nodeCapability()
			fmt.Printf("count = %d, needed = %d, deployed = %d, needed-deployed = %d, maxInstances = %d\n",
				count, needed, deployed, needed-deployed, maxInstances)
			retSolution[bestS][service] = int(count)
			deployed += int(count)
		}
	}

	return retSolution
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
					// fmt.Println("enter edgeReplacement()")
					Xi1 := edgeReplacement(cand)
					nextSls = append(nextSls, Xi1)
				}
			}
		}

		if len(nextSls) == 0 {
			break
		} else {
			fmt.Printf("len(nextSls) = %d\n", len(nextSls))
			for i, s := range nextSls {
				fmt.Printf("\n\nnextSls[%d]: \n", i)
				printJSON(s, "")
			}
			printJSON(nextSls[len(nextSls)-1], "")
			// TODO: SORT HERE!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
			if len(nextSls) > BS {
				nextSls = nextSls[:BS]
			}
			tempSls = nextSls
		}
	}

	fmt.Printf("len(SLs) = %d\n", len(SLs))
	for i, s := range SLs {
		fmt.Printf("\n\nSLs[%d]: \n", i)
		printJSON(s, "")
	}

	// TODO: SORT HERE!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	return SLs[4]
}

func RunTIGO() {
	InitTIGO()
	BS := 5 // Branch Search Size
	bestSolution := tigo(BS)
	fmt.Println("Best Solution:")
	printJSON(bestSolution, "")
	printJSON(bestSolution, "tigo_solution2.json")
	UpDateDeploymentsByJSON("tigo_solution2.json")
}
