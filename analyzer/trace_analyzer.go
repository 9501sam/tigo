package analyzer

import (
	"encoding/csv"
	"fmt"
	"optimizer/common"
	"optimizer/utils"
	"os"
	"strconv"
	"strings"
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

func CountInvocationOfTraces(traceData common.TraceData) InvocationCount {
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

func CountInvocationOfOneTrace(trace common.Trace) InvocationCount {
	callCount := make(map[common.CallKey]int)
	for _, span := range trace.Spans {
		if span.ParentService != "" && span.ParentService != span.ServiceName {
			key := common.CallKey{From: span.ParentService, To: span.ServiceName}
			callCount[key]++
		}
	}
	return callCount
}

func RemoveNone(invCount InvocationCount) InvocationCount {
	newInvCount := make(InvocationCount) // Create a new map for the filtered results

	for key, count := range invCount {
		// If neither 'From' nor 'To' is "none", keep the entry
		if key.From != "none" && key.To != "none" {
			newInvCount[key] = count

		}

	}
	return newInvCount
}

func FindAllRoots(invCount InvocationCount) []string {
	fromNodes := make(map[string]struct{}) // Set of all 'From' entities
	toNodes := make(map[string]struct{})   // Set of all 'To' entities
	for k := range invCount {
		fromNodes[k.From] = struct{}{}
		toNodes[k.To] = struct{}{}
	}
	var roots []string
	for from := range fromNodes {
		// If a 'From' node is not present in the 'To' nodes, it's a root
		if _, exists := toNodes[from]; !exists {
			roots = append(roots, from)
		}
	}
	return roots
}

// / *** ExtICsFromCallGraph *** ///
// Input: A trace: t; a call graph: G;
func ExtICsFromCallGraph(NumI_t InvocationCount) (*InvocationChains, InvocationCount) {
	stack := utils.NewStack()
	AddNode := utils.NewStack()
	CurrentNum := utils.NewStack()
	InvChains := NewInvocationChains()
	IC := NewInvocationChain()
	newNumI_t := NumI_t.Copy()
	roots := FindAllRoots(NumI_t)
	for _, r := range roots {
		fmt.Printf("r = %s\n", r)
	}
	root := common.CallKey{From: "none", To: roots[0]}
	stack.Push(root)

	for key, count := range NumI_t {
		fmt.Printf("From: %s, To: %s, Count: %d\n", key.From, key.To, count)
	}

	iter := 0
	for !stack.IsEmpty() {
		fmt.Println("\n----------------------------------------")
		node := stack.Pop().(common.CallKey)
		n := node.To
		fmt.Printf("Iteration %d start\n", iter)
		fmt.Printf("n = %s\n", n)
		fmt.Printf("IC = %s\n", IC.String())
		fmt.Printf("IC.NumIC_t_IC = %d\n", IC.NumIC_t_IC)
		fmt.Println("----")
		iter++

		if IC.IsEmpty() {
			IC.Append(n)
			IC.NumIC_t_IC = 0
			AddNode.Push(n)
			CurrentNum.Push(0)
		} else {
			if NumI_t.Exist(IC.GetTail(), n) {
				fmt.Printf("Extend n = %s the current IC\n", n)
				if IC.NumIC_t_IC == 0 {
					IC.NumIC_t_IC = NumI_t.GetCount(IC.GetTail(), n)
				} else {
					IC.NumIC_t_IC = min(IC.NumIC_t_IC, NumI_t.GetCount(IC.GetTail(), n))
				}
				IC.Append(n)
				AddNode.Push(n)
				CurrentNum.Push(IC.NumIC_t_IC)
				// newNumI_t[common.CallKey{From: node.From, To: node.To}] -= IC.NumIC_t_IC
				newNumI_t.Decrease(common.CallKey{From: node.From, To: node.To}, IC.NumIC_t_IC)
				fmt.Printf("IC = %s\n", IC.String())
				fmt.Printf("IC.NumIC_t_IC = %d\n", IC.NumIC_t_IC)

			} else {
				fmt.Printf("The current IC ends, create a new IC\n")
				InvChains.Append(IC)
				fmt.Printf("push IC = %s, IC.NumIC_t_IC = %d to InvChains\n", IC.String(), IC.NumIC_t_IC)

				Junc := AddNode.Top().(string)
				candNum := CurrentNum.Top().(int)
				for Junc != node.From {
					AddNode.Pop()
					CurrentNum.Pop()
					Junc = AddNode.Top().(string)
					candNum = CurrentNum.Top().(int)
				}

				// copy eletent in stack `AddNode` to IC
				// make IC to size AddNode.Size()
				IC.Microservices = IC.Microservices[:AddNode.Size()]
				if candNum == 0 {
					IC.NumIC_t_IC = NumI_t.GetCount(Junc, n)
				} else {
					IC.NumIC_t_IC = min(candNum, NumI_t.GetCount(Junc, n))
				}

				fmt.Printf("candNum = %d, NumI_t.GetCount(Junc, n) = %d\n", candNum, NumI_t.GetCount(Junc, n))
				fmt.Printf("IC = %s\n", IC.String())
				fmt.Printf("IC.NumIC_t_IC = %d\n", IC.NumIC_t_IC)

				IC.Append(n)
				AddNode.Push(n)
				CurrentNum.Push(IC.NumIC_t_IC)
				// newNumI_t[common.CallKey{From: node.From, To: node.To}] -= IC.NumIC_t_IC
				newNumI_t.Decrease(common.CallKey{From: node.From, To: node.To}, IC.NumIC_t_IC)
			}
		}

		// push childs of `n` to the `stack`
		for _, s := range common.Services {
			if NumI_t.Exist(n, s) {
				stack.Push(common.CallKey{From: n, To: s})
			}
		}
	}
	InvChains.Append(IC)
	return InvChains, newNumI_t
}

func getTotalInvChains() *InvocationChains {
	common.LoadJSONFile("app.json", &traceData)

	totalInvChains := NewInvocationChains()
	for i, trace := range traceData.Data {
		fmt.Printf("--- Processing Trace %d (TraceID: %s) ---\n", i+1, trace.TraceID)
		// Extract invocation chains from the current trace using the updated algorithm
		NumI_t := CountInvocationOfOneTrace(trace)
		NumI_t = RemoveNone(NumI_t)

		InvChains := NewInvocationChains()
		for len(NumI_t) > 0 {
			tmpInvChains, newNumI_t := ExtICsFromCallGraph(NumI_t)

			// // Check if any invocation chains were extracted
			// if len(tmpInvChains.Chains) > 0 {
			// 	fmt.Println("Extracted Invocation Chains for this trace:")
			// 	// Iterate through the map of invocation chains and their total occurrences
			// 	for chainStr, count := range tmpInvChains.Chains {
			// 		fmt.Printf("  Chain: %s, Occurrences: %d\n", chainStr, count)
			// 	}
			// } else {
			// 	fmt.Println("No invocation chains extracted for this trace.")
			// }
			// fmt.Println("----------------------------------------")
			// fmt.Println("Before:")
			// for key, count := range NumI_t {
			// 	fmt.Printf("From: %s, To: %s, Count: %d\n", key.From, key.To, count)
			// }
			// fmt.Println("After:")
			// for key, count := range newNumI_t {
			// 	fmt.Printf("From: %s, To: %s, Count: %d\n", key.From, key.To, count)
			// }

			InvChains.Add(tmpInvChains)
			NumI_t = newNumI_t
		}

		fmt.Printf("--- InvChains for Trace %d (TraceID: %s) ---\n", i+1, trace.TraceID)
		for chainStr, count := range InvChains.Chains {
			fmt.Printf("  Chain: %s, Occurrences: %d\n", chainStr, count)
		}
		totalInvChains.Add(InvChains)
	}
	// fmt.Println("----------------------------------------")
	// for chainStr, count := range totalInvChains.Chains {
	// 	fmt.Printf("  Chain: %s, Occurrences: %d\n", chainStr, count)
	// }

	return totalInvChains
}

func DepIC(mi, mj string, InvChains InvocationChains) float64 {
	Num_mi_mj := 0
	Num_mi := 0
	Num_mj := 0
	Cd_mi := 0.0
	Cd_mj := 0.0
	invocationNum := 0
	invocationMi := 0
	invocationMj := 0

	for chainStr, count := range InvChains.Chains {
		mi_flag := strings.Contains(chainStr, mi)
		mj_flag := strings.Contains(chainStr, mj)
		if mi_flag {
			Num_mi += count
		}
		if mj_flag {
			Num_mj += count
		}
		if mi_flag && mj_flag {
			Num_mi_mj += count
		}
	}

	NumI_t := CountInvocationOfTraces(traceData)
	for _, mx := range common.Services {
		for _, my := range common.Services {
			invocationNum += NumI_t.GetCount(mx, my)
		}
	}
	for _, mx := range common.Services {
		invocationMi += NumI_t.GetCount(mi, mx)
	}
	for _, mx := range common.Services {
		invocationMj += NumI_t.GetCount(mj, mx)
	}
	Cd_mi = float64(invocationMi) / float64(invocationNum)
	Cd_mj = float64(invocationMj) / float64(invocationNum)

	// fmt.Println("----------------------------------------")
	// fmt.Printf("mi = %s, mj = %s\n", mi, mj)
	// fmt.Printf("Num_mi_mj = %d\n", Num_mi_mj)
	// fmt.Printf("Num_mi = %d\n", Num_mi)
	// fmt.Printf("Num_mj = %d\n", Num_mj)
	// fmt.Printf("Cd_mi = %.6f\n", Cd_mi)
	// fmt.Printf("Cd_mj = %.6f\n", Cd_mj)
	// fmt.Printf("invocationNum = %d\n", invocationNum)
	// fmt.Printf("invocationMi = %d\n", invocationMi)
	// fmt.Printf("invocationMj = %d\n", invocationMj)
	// fmt.Println("----------------------------------------")

	epslon := 1.0
	return (1/(Cd_mi+epslon))*(float64(Num_mi_mj)/float64(Num_mi)) + (1/(Cd_mj+epslon))*(float64(Num_mi_mj)/float64(Num_mj))
}

func ExportDepICsToCSV(DepICs map[common.CallKey]float64, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	// Write header
	writer.Write([]string{"from", "to", "DepIC_Value"})

	for k, v := range DepICs {
		// writer.Write([]string{k.From, k.To, strconv.Itoa(v)}) // TODO: should not use Itoa cause it's float64
		depICStr := strconv.FormatFloat(v, 'f', -1, 64)
		writer.Write([]string{k.From, k.To, depICStr})
	}
	return nil
}

func GenerateAndSaveDepICs() {
	totalInvChains := getTotalInvChains()
	DepICs := make(map[common.CallKey]float64)
	for _, mx := range common.Services {
		for _, my := range common.Services {
			depIC := 0.0
			if mx != my {
				depIC = DepIC(mx, my, *totalInvChains)
			}
			DepICs[common.CallKey{From: mx, To: my}] = depIC
			// fmt.Printf("DepIC(%s, %s) = %f\n", mx, my, depIC)
		}
	}

	// ExportCallCountsToCSV(DepICs, "depICs.csv")
	err := ExportDepICsToCSV(DepICs, "depICs.csv")
	if err != nil {
		fmt.Printf("Error exporting DepICs to CSV: %v\n", err)
	} else {
		fmt.Println("DepICs data successfully exported to depICs.csv")
	}
}

func RunAnalyzer() {
	totalInvChains := getTotalInvChains()
	fmt.Println("----------------------------------------")
	for chainStr, count := range totalInvChains.Chains {
		fmt.Printf("  Chain: %s, Occurrences: %d\n", chainStr, count)
	}

	fmt.Println("-----------DepIC--------------------------")
	for _, mx := range common.Services {
		for _, my := range common.Services {
			depIC := DepIC(mx, my, *totalInvChains)
			fmt.Printf("DepIC(%s, %s) = %f\n", mx, my, depIC)
		}
	}

	GenerateAndSaveDepICs()

	// fmt.Println("-----------DepIC--------------------------")
	// depIC := DepIC("checkoutservice", "emailservice", *totalInvChains)
	// fmt.Printf("DepIC(%s, %s) = %f\n", "checkoutservice", "emailservice", depIC)
}
