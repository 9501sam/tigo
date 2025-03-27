package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TraceData is the target struct to fill with Jaeger data.
type TraceData struct {
	Data []struct {
		TraceID           string `json:"traceID"`
		Duration          int64  `json:"duration"`          // Microseconds (µs)
		PredictedDuration int64  `json:"predictedDuration"` // Microseconds (µs)
		Spans             []struct {
			OperationName   string `json:"operationName"`
			ProcessID       string `json:"processID"`
			ParentService   string `json:"parentService"`
			ParentOperation string `json:"parentOperation"`
			StartTime       int64  `json:"startTime"`
			Duration        int64  `json:"duration"`
		} `json:"spans"`
	} `json:"data"`
}

// RawTrace represents the raw Jaeger API response structure.
type RawTrace struct {
	TraceID   string             `json:"traceID"`
	Spans     []RawSpan          `json:"spans"`
	Processes map[string]Process `json:"processes"`
}

// RawSpan represents a span in the raw Jaeger API response.
type RawSpan struct {
	TraceID       string      `json:"traceID"`
	SpanID        string      `json:"spanID"`
	OperationName string      `json:"operationName"`
	References    []Reference `json:"references"`
	StartTime     int64       `json:"startTime"` // Microseconds since epoch
	Duration      int64       `json:"duration"`  // Microseconds
	ProcessID     string      `json:"processID"`
}

// Reference defines a relationship between spans (e.g., parent-child).
type Reference struct {
	RefType string `json:"refType"`
	SpanID  string `json:"spanID"`
}

// Process contains service metadata.
type Process struct {
	ServiceName string `json:"serviceName"`
}

func main() {
	// Fetch traces from Jaeger API
	url := "http://localhost:16686/api/traces?service=frontend&limit=1"
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

	// Parse raw Jaeger JSON response
	var rawResponse struct {
		Data []RawTrace `json:"data"`
	}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		return
	}

	// Convert raw data to TraceData struct
	traceData := convertToTraceData(rawResponse.Data)

	// Print the resulting TraceData as JSON for verification
	jsonOutput, err := json.MarshalIndent(traceData, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal TraceData: %v\n", err)
		return
	}
	fmt.Println(string(jsonOutput))
}

// convertToTraceData transforms raw Jaeger data into the TraceData struct.
func convertToTraceData(rawTraces []RawTrace) TraceData {
	traceData := TraceData{
		Data: make([]struct {
			TraceID           string `json:"traceID"`
			Duration          int64  `json:"duration"`
			PredictedDuration int64  `json:"predictedDuration"`
			Spans             []struct {
				OperationName   string `json:"operationName"`
				ProcessID       string `json:"processID"`
				ParentService   string `json:"parentService"`
				ParentOperation string `json:"parentOperation"`
				StartTime       int64  `json:"startTime"`
				Duration        int64  `json:"duration"`
			} `json:"spans"`
		}, len(rawTraces)),
	}

	for i, rawTrace := range rawTraces {
		// Build a map of spanID to RawSpan for parent lookup
		spanMap := make(map[string]RawSpan)
		for _, span := range rawTrace.Spans {
			spanMap[span.SpanID] = span
		}

		// Calculate total duration of the trace (sum of root span durations or max span duration)
		var totalDuration int64
		for _, span := range rawTrace.Spans {
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

		// Fill trace-level fields
		traceData.Data[i].TraceID = rawTrace.TraceID
		traceData.Data[i].Duration = totalDuration
		traceData.Data[i].PredictedDuration = 0 // Not provided by Jaeger, default to 0
		traceData.Data[i].Spans = make([]struct {
			OperationName   string `json:"operationName"`
			ProcessID       string `json:"processID"`
			ParentService   string `json:"parentService"`
			ParentOperation string `json:"parentOperation"`
			StartTime       int64  `json:"startTime"`
			Duration        int64  `json:"duration"`
		}, len(rawTrace.Spans))

		// Fill span-level fields
		for j, rawSpan := range rawTrace.Spans {
			traceData.Data[i].Spans[j].OperationName = rawSpan.OperationName
			traceData.Data[i].Spans[j].ProcessID = rawSpan.ProcessID
			traceData.Data[i].Spans[j].StartTime = rawSpan.StartTime
			traceData.Data[i].Spans[j].Duration = rawSpan.Duration

			// Determine parent service and operation
			parentService := "none"
			parentOperation := "none"
			for _, ref := range rawSpan.References {
				if ref.RefType == "CHILD_OF" {
					if parentSpan, exists := spanMap[ref.SpanID]; exists {
						parentService = rawTrace.Processes[parentSpan.ProcessID].ServiceName
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
