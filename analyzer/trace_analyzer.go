package analyzer

import (
	"encoding/csv"
	"optimizer/common"
	"optimizer/utils"
	"os"
	"strconv"
)

var traceData common.TraceData
var G = common.G

func ExportCallCountsToCSV(callCount map[common.CallKey]int, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"from", "to", "count"})

	for k, v := range callCount {
		writer.Write([]string{k.From, k.To, strconv.Itoa(v)})
	}

	return nil
}

func CountInvocationOfTraces(traceData common.TraceData) map[common.CallKey]int {
	callCount := make(map[common.CallKey]int)
	for _, trace := range traceData.Data {
		for _, span := range trace.Spans {
			if span.ParentService != "" && span.ParentService != span.ServiceName {
				key := common.CallKey{From: span.ParentService, To: span.ServiceName}
				callCount[key]++
			}
		}
	}
	return callCount
}

func CountInvocationOfOneTrace(trace common.Trace) map[common.CallKey]int {
	callCount := make(map[common.CallKey]int)
	for _, span := range trace.Spans {
		if span.ParentService != "" && span.ParentService != span.ServiceName {
			key := common.CallKey{From: span.ParentService, To: span.ServiceName}
			callCount[key]++
		}
	}
	return callCount
}

/// *** ExtICsFromCallGraph *** ///
// Input: A trace: t; a call graph: G;
func ExtICsFromCallGraph(t common.Trace) *InvocationChains {
	stack := utils.NewStack()
	AddNode := utils.NewStack()
	CurrentNum := utils.NewStack()
	InvChains := NewInvocationChains()
	root := "frontend"
	stack.Push(root)
	IC := NewInvocationChain()
	NumI_t := CountInvocationOfOneTrace(t)
	for !stack.IsEmpty() {
		n := stack.Pop().(string)
		if IC.IsEmpty() {
			IC.Append(n)
			AddNode.Push(n)
			CurrentNum.Push(0)
		} else {
			if _, ok := NumI_t[common.CallKey{From: IC.GetTail(), To: n}]; ok {
				IC.Append(n)
				AddNode.Push(n)
				if IC.NumIC_t_IC == 0 {
					IC.NumIC_t_IC = NumI_t[common.CallKey{From: IC.GetTail(), To: n}]
				} else {
					IC.NumIC_t_IC = min(IC.NumIC_t_IC, NumI_t[common.CallKey{From: IC.GetTail(), To: n}])
				}
				CurrentNum.Push(IC.NumIC_t_IC)
			} else {
				InvChains.Append(IC)
				P := AddNode.Top().(string)
				var candNum int
				var Junc string
				for {
					_, ok := NumI_t[common.CallKey{From: IC.GetTail(), To: n}]
					if !ok {
						break
					}
					Junc = AddNode.Pop().(string)
					candNum = CurrentNum.Pop().(int)
				}
				// TODO: List IC = List.copy(IC, 0, Junc.index);
				IC.Append(n)
				IC.NumIC_t_IC = min(candNum, NumI_t[common.CallKey{From: Junc, To: n}])
				AddNode.Push(n)
				CurrentNum.Push(IC.NumIC_t_IC)
			}
		}
		if children, exists := G[n]; exists {
			for child := range children {
				stack.Push(child)
			}
		}
	}
	return InvChains
}

func RunAnalyzer() {

	common.LoadJSONFile("app.json", &traceData)

	for _, trace := range traceData.Data {
		InvChains := ExtICsFromCallGraph(trace)
	}
}
