package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// Trace represents the top-level trace object from the API.
type Trace struct {
	TraceID   string             `json:"traceID"`
	Spans     []Span             `json:"spans"`
	Processes map[string]Process `json:"processes"`
}

// Span represents an individual span within a trace.
type Span struct {
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
	url := "http://localhost:16686/api/traces?service=frontend&limit=2"
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

	// Parse JSON response
	var response struct {
		Data []Trace `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		return
	}

	// Process each trace
	for _, trace := range response.Data {
		fmt.Printf("Trace ID: %s\n", trace.TraceID)

		// Build a map of spanID to Span for easy lookup
		spanMap := make(map[string]Span)
		for _, span := range trace.Spans {
			spanMap[span.SpanID] = span
		}

		// Find root spans (no parent) and build hierarchy
		var rootSpans []Span
		for _, span := range trace.Spans {
			hasParent := false
			for _, ref := range span.References {
				if ref.RefType == "CHILD_OF" {
					hasParent = true
					break
				}
			}
			if !hasParent {
				rootSpans = append(rootSpans, span)
			}
		}

		// Print calling hierarchy
		fmt.Println("Calling Hierarchy:")
		for _, root := range rootSpans {
			printSpanHierarchy(root, spanMap, trace.Processes, 0)
		}

		// Sort spans by start time for sequence
		sort.Slice(trace.Spans, func(i, j int) bool {
			return trace.Spans[i].StartTime < trace.Spans[j].StartTime
		})

		// Print sequence
		fmt.Println("Sequence of Operations:")
		for _, span := range trace.Spans {
			start := time.UnixMicro(span.StartTime)
			service := trace.Processes[span.ProcessID].ServiceName
			fmt.Printf("%s: %s (%s) - Duration: %dµs\n", start.Format(time.RFC3339Nano), span.OperationName, service, span.Duration)
		}
		fmt.Println("---")
	}
}

// printSpanHierarchy recursively prints the span hierarchy.
func printSpanHierarchy(span Span, spanMap map[string]Span, processes map[string]Process, depth int) {
	service := processes[span.ProcessID].ServiceName
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "  "
	}
	fmt.Printf("%s%s (%s)\n", prefix, span.OperationName, service)

	// Find children
	for _, s := range spanMap {
		for _, ref := range s.References {
			if ref.RefType == "CHILD_OF" && ref.SpanID == span.SpanID {
				printSpanHierarchy(s, spanMap, processes, depth+1)
			}
		}
	}
}
