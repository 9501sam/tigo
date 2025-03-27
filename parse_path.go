package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TraceData represents both the raw Jaeger API response and the target structure.
type TraceData struct {
	Data []struct {
		TraceID           string `json:"traceID"`
		Duration          int64  `json:"duration"`          // Microseconds (µs)
		PredictedDuration int64  `json:"predictedDuration"` // Microseconds (µs)
		Spans             []Span `json:"spans"`
		Processes         map[string]struct {
			ServiceName string `json:"serviceName"`
		} `json:"processes"`
	} `json:"data"`
}

// Span represents a span within a trace, used in both Spans and spanMap.
type Span struct {
	TraceID       string `json:"traceID"`
	SpanID        string `json:"spanID"`
	OperationName string `json:"operationName"`
	References    []struct {
		RefType string `json:"refType"`
		SpanID  string `json:"spanID"`
	} `json:"references"`
	StartTime       int64  `json:"startTime"`
	Duration        int64  `json:"duration"`
	ProcessID       string `json:"processID"`
	ParentService   string `json:"parentService"`
	ParentOperation string `json:"parentOperation"`
}

func main() {
	// Fetch traces from Jaeger API
	url := "http://localhost:16686/api/traces?service=frontend&limit=4"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Failed to fetch traces: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		return
	}

	// Parse raw Jaeger JSON response directly into TraceData
	var traceData TraceData
	if err := json.Unmarshal(body, &traceData); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		return
	}

	// Populate ParentService and ParentOperation
	traceData = populateParentFields(traceData)

	// Print the resulting TraceData as JSON for verification
	jsonOutput, err := json.MarshalIndent(traceData, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal TraceData: %v\n", err)
		return
	}
	fmt.Println(string(jsonOutput))
}

// populateParentFields fills in ParentService and ParentOperation for each span.
func populateParentFields(traceData TraceData) TraceData {
	for i, trace := range traceData.Data {
		// Build a map of spanID to Span for parent lookup
		spanMap := make(map[string]Span)
		for _, span := range trace.Spans {
			spanMap[span.SpanID] = span
		}

		// Calculate total duration of the trace (using root span duration or max span duration)
		var totalDuration int64
		for _, span := range trace.Spans {
			hasParent := false
			for _, ref := range span.References {
				if ref.RefType == "CHILD_OF" {
					hasParent = true
					break
				}
			}
			if !hasParent && span.Duration > totalDuration {
				totalDuration = span.Duration
			}
		}
		traceData.Data[i].Duration = totalDuration
		traceData.Data[i].PredictedDuration = 0 // Default to 0 as not provided by Jaeger

		// Update spans with parent information
		for j, span := range trace.Spans {
			parentService := "none"
			parentOperation := "none"
			for _, ref := range span.References {
				if ref.RefType == "CHILD_OF" {
					if parentSpan, exists := spanMap[ref.SpanID]; exists {
						parentService = trace.Processes[parentSpan.ProcessID].ServiceName
						parentOperation = parentSpan.OperationName
					}
					break
				}
			}
			traceData.Data[i].Spans[j].ParentService = parentService
			traceData.Data[i].Spans[j].ParentOperation = parentOperation
		}
	}
	return traceData
}
