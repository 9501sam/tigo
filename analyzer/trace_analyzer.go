package analyzer

import (
	"encoding/csv"
	"fmt"
	"optimizer/common"
	"optimizer/utils"
	"os"
	"strconv"
)

type StackNodeInfo struct {
	Name     string
	ICLength int // Length of IC.Microservices slice *after* this node was added. Used as exclusive endIndex for Copy.
}

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
	for key, count := range NumI_t {
		fmt.Printf("From: %s, To: %s, Count: %d\n", key.From, key.To, count)
	}

	iter := 0
	for !stack.IsEmpty() {
		fmt.Println("\n----------------------------------------")
		n := stack.Pop().(string)
		fmt.Printf("Iteration %d start\n", iter)
		fmt.Printf("n = %s\n", n)
		fmt.Printf("IC = %s\n", IC.String())
		fmt.Printf("IC.NumIC_t_IC = %d\n", IC.NumIC_t_IC)
		iter++

		if IC.IsEmpty() {
			IC.Append(n)
			IC.NumIC_t_IC = 0
			AddNode.Push(n)
			CurrentNum.Push(0)
		} else {
			if _, ok := NumI_t[common.CallKey{From: IC.GetTail(), To: n}]; ok {
				if IC.NumIC_t_IC == 0 {
					IC.NumIC_t_IC = NumI_t[common.CallKey{From: IC.GetTail(), To: n}]
					fmt.Printf("IC.NumIC_t_IC = %d\n", IC.NumIC_t_IC)
				} else {
					IC.NumIC_t_IC = min(IC.NumIC_t_IC, NumI_t[common.CallKey{From: IC.GetTail(), To: n}])
				}
				IC.Append(n)
				AddNode.Push(n)
				CurrentNum.Push(IC.NumIC_t_IC)
			} else {
				fmt.Printf("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
				InvChains.Append(IC)
				// *** TODO: 24 ~ 30 *** //

				IC.Append(n)
				// IC.NumIC_t_IC = min(candNum, NumI_t[common.CallKey{From: Junc, To: n}])
				AddNode.Push(n)
				CurrentNum.Push(IC.NumIC_t_IC)
			}
		}

		// push childs of `n` to the `stack`
		for _, s := range common.Services {
			if count, ok := NumI_t[common.CallKey{From: n, To: s}]; ok && (count != 0) {
				stack.Push(s)
			}
		}
	}
	return InvChains
}

func RunAnalyzer() {
	common.LoadJSONFile("app.json", &traceData)

	i := 1
	trace := traceData.Data[i]
	fmt.Printf("--- Processing Trace %d (TraceID: %s) ---\n", i+1, trace.TraceID)
	// Extract invocation chains from the current trace using the updated algorithm
	InvChains := ExtICsFromCallGraph(trace)

	// Check if any invocation chains were extracted
	if len(InvChains.Chains) > 0 {
		fmt.Println("Extracted Invocation Chains for this trace:")
		// Iterate through the map of invocation chains and their total occurrences
		for chainStr, count := range InvChains.Chains {
			fmt.Printf("  Chain: %s, Occurrences: %d\n", chainStr, count)
		}
	} else {
		fmt.Println("No invocation chains extracted for this trace.")
	}
	fmt.Println("----------------------------------------")
}
