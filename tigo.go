package main

import (
	"fmt"
)

type Solution map[string]map[string]int // Solution[node][service] = <replica>

// 初始化路由
func initRouting() ([]int, int) {
	// 假設初始 R 是空的，成本為 0
	return []int{}, 0
}

// 雲端執行方案改進
func cloudExecSchemeImprove(solution Solution, BS int) []Solution {
	// 假設這裡返回一些可能的解決方案
	return []Solution{}
}

// 邊緣替換策略
func edgeReplacement(solution Solution) Solution {
	// 這裡模擬邊緣替換的過程
	return nil
}

func tigo(BS int) Solution {
	// var X []int
	// R, cost := initRouting()

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
			// sort.Slice(nextSls, func(i, j int) bool {
			// 	return nextSls[i].Cost > nextSls[j].Cost // 逆序排列
			// })
			if len(nextSls) > BS {
				nextSls = nextSls[:BS]
			}
			tempSls = nextSls
		}
	}

	// sort.Slice(SLs, func(i, j int) bool { 				TODO: after adding the cost then sort
	// 	return SLs[i].Cost > SLs[j].Cost // 逆序排列
	// })

	return SLs[0]
}

func RunTIGO() {
	BS := 5 // 設定 Branch Search Size
	bestSolution := tigo(BS)
	fmt.Println("Best Solution:", bestSolution)
}
