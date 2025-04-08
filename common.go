package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var nodes = []string{"vm1", "vm2", "vm3", "asus"}

// TraceData represents both the raw Jaeger API response and the target structure.
type TraceData struct {
	AverageDuration          int64 `json:"averageDuration"`          // Microseconds (µs)
	AveragePredictedDuration int64 `json:"averagePredictedDuration"` // Microseconds (µs)
	Data                     []struct {
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
	ServiceName     string `json:"serviceName"`
	ParentService   string `json:"parentService"`
	ParentOperation string `json:"parentOperation"`
}

func loadJSONFile[T any](filename string, target *T) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(target)
}

func printJSON(data interface{}, fileName string) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return

	}
	fmt.Println(string(jsonData))

	if fileName != "" {
		err = os.WriteFile(fileName, jsonData, 0644)
		if err != nil {
			log.Fatalf("Error writing JSON to file: %v", err)

		}
	}
}

type Constraints struct {
	CPU    int `json:"cpu"`
	Memory int `json:"memory"`
}

type ResourceConstraints map[string]Constraints
type NodeConstraints map[string]Constraints

var serviceConstraints ResourceConstraints
var nodeConstraints NodeConstraints
