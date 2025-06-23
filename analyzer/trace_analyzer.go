package analyzer

import (
	"encoding/csv"
	"fmt"
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

func CountServiceCalls(traceData common.TraceData) map[common.CallKey]int {
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

/// *** ExtICsFromCallGraph *** ///
// Input: A trace: t; a call graph: G;
func ExtICsFromCallGraph(t common.Trace) {
	stack := NewStack()
	AddNode := NewStack()
	CurrentNum := NewStack()
	stack := NewStack()
	var InvChains [][]string
	root := "frontend"
	stack.push(root)
	var IC []string
}

func RunAnalyzer() {

	common.LoadJSONFile("app.json", &traceData)

	callCounts := CountServiceCalls(traceData)
	for k, v := range callCounts {
		fmt.Printf("%s -> %s: %d times\n", k.From, k.To, v)
	}

	// ExportCallCountsToCSV(callCounts, "service_calls.csv")

}
