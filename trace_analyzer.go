package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

type CallKey struct {
	From string
	To   string
}

func ExportCallCountsToCSV(callCount map[CallKey]int, filename string) error {
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

func CountServiceCalls(traceData TraceData) map[CallKey]int {
	callCount := make(map[CallKey]int)

	for _, trace := range traceData.Data {
		for _, span := range trace.Spans {
			if span.ParentService != "" && span.ParentService != span.ServiceName {
				key := CallKey{From: span.ParentService, To: span.ServiceName}
				callCount[key]++
			}
		}
	}

	return callCount
}

func RunCountServiceCalls() {

	loadJSONFile("app.json", &traceData)

	callCounts := CountServiceCalls(traceData)
	for k, v := range callCounts {
		fmt.Printf("%s -> %s: %d times\n", k.From, k.To, v)
	}

	ExportCallCountsToCSV(callCounts, "service_calls.csv")
}
