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
	for !stack.IsEmpty() {
		n := stack.Pop().(string)
		if IC.IsEmpty() {
			IC.Append(n)
			// AddNode.Push(n)
			AddNode.Push(StackNodeInfo{Name: n, ICLength: len(IC.Microservices)})
			IC.NumIC_t_IC = 0
			CurrentNum.Push(0)
		} else {
			if _, ok := NumI_t[common.CallKey{From: IC.GetTail(), To: n}]; ok {
				IC.Append(n)
				// AddNode.Push(n)
				AddNode.Push(StackNodeInfo{Name: n, ICLength: len(IC.Microservices)})
				if IC.NumIC_t_IC == 0 {
					IC.NumIC_t_IC = NumI_t[common.CallKey{From: IC.GetTail(), To: n}]
				} else {
					IC.NumIC_t_IC = min(IC.NumIC_t_IC, NumI_t[common.CallKey{From: IC.GetTail(), To: n}])
				}
				CurrentNum.Push(IC.NumIC_t_IC)
			} else {
				InvChains.Append(IC)
				// P := AddNode.Top().(string)
				var candNum int
				// var Junc string
				var juncNodeInfo StackNodeInfo
				for {
					if AddNode.IsEmpty() {
						// This case implies 'n' cannot be connected to any prior node in the path.
						// This means 'n' likely starts a new, disjoint invocation chain, or is unreachable.
						// Reset IC to empty and handle 'n' as a new start.
						fmt.Printf("Warning: AddNode is empty while searching for junction for node %s. Starting new chain.\n", n)
						IC = NewInvocationChain() // Reset IC
						// juncNodeInfo will remain default (empty) indicating no junction from previous path.
						candNum = 0 // Default for a new chain starting with 'n' if no prior connection.
						break       // Exit loop, no valid junction found from previous path.
					}
					// Get current potential junction candidate from top of stack without popping yet
					currentJuncCandidateInfo := AddNode.Top().(StackNodeInfo)
					currentCandNum := CurrentNum.Top().(int)

					// _, ok := NumI_t[common.CallKey{From: IC.GetTail(), To: n}]
					// if !ok {
					// 	break
					// }
					// Junc = AddNode.Pop().(string)
					// candNum = CurrentNum.Pop().(int)

					// Check connectivity from current candidate to 'n'
					_, okConnect := NumI_t[common.CallKey{From: currentJuncCandidateInfo.Name, To: n}]

					if okConnect {
						// This 'currentJuncCandidateInfo' is our junction.
						// The new IC will be a copy up to and including this node.
						juncNodeInfo = currentJuncCandidateInfo
						candNum = currentCandNum
						break // Found the junction, exit the backtracking loop
					} else {
						// No direct connection from current candidate to 'n'. Pop to backtrack further.
						AddNode.Pop()
						CurrentNum.Pop()
					}
				}
				if juncNodeInfo.Name != "" { // Check if a valid junction was found
					IC = IC.Copy(0, juncNodeInfo.ICLength) // Implements: List IC = List.copy(IC, 0, Junc.index);
					IC.NumIC_t_IC = candNum                // Set the chain's min count to the value at the junction
				} else {
					// If juncNodeInfo.Name is empty, it means AddNode became empty in the loop.
					// IC was already reset to NewInvocationChain() above.
				}
				IC.Append(n)
				// AddNode.Push(n)
				// Push the new node info for 'n' onto AddNode
				AddNode.Push(StackNodeInfo{Name: n, ICLength: len(IC.Microservices)})

				// IC.NumIC_t_IC = min(candNum, NumI_t[common.CallKey{From: Junc, To: n}])
				if juncNodeInfo.Name != "" { // If a valid junction was found
					IC.NumIC_t_IC = min(candNum, NumI_t[common.CallKey{From: juncNodeInfo.Name, To: n}])
				} else {
					// If 'n' is the first node in a new chain (no junction found for it),
					// its NumIC_t_IC depends on how a single-node chain's min count is defined.
					// For now, setting to 0, which will be updated as it forms new invocations.
					IC.NumIC_t_IC = 0
				}
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

	// Iterate through each trace in the loaded traceData
	for i, trace := range traceData.Data {
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
}
