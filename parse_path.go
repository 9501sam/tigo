package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// fetch from jaeger API
const jaegerBaseURLWithTime = "http://localhost:16686/api/traces?service=%s&start=%d&end=%d" // Jaeger API URL

func fetchTraces(service string, start, end int64) (*TraceData, error) {
	url := fmt.Sprintf(jaegerBaseURLWithTime, service, start, end)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching traces for %s: %w", service, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body for %s: %w", service, err)
	}

	var traces TraceData
	if err := json.Unmarshal(body, &traces); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON for %s: %w", service, err)
	}

	return &traces, nil
}

func calculateAverageDuration(traceData *TraceData) *TraceData {
	var totalDuration int64
	var totalPredictedDuration int64
	var count int64

	for _, trace := range traceData.Data {
		totalDuration += trace.Duration
		totalPredictedDuration += trace.PredictedDuration
		count++
	}

	if count > 0 {
		avgDuration := (totalDuration / count) / 1000
		avgPredictedDuration := (totalPredictedDuration / count) / 1000
		traceData.AverageDuration = avgDuration
		traceData.AveragePredictedDuration = avgPredictedDuration
	} else {
		fmt.Println("No data available to calculate averages.")
	}

	return traceData
}

func setTraceDuration(traceData *TraceData) *TraceData {
	for i := range traceData.Data {
		earliestStart := int64(1<<63 - 1) // 最大 int64
		latestEnd := int64(0)

		for j := range traceData.Data[i].Spans {
			span := &traceData.Data[i].Spans[j]

			if span.StartTime < earliestStart {
				earliestStart = span.StartTime
			}
			endTime := span.StartTime + span.Duration
			if endTime > latestEnd {
				latestEnd = endTime
			}
		}
		traceData.Data[i].Duration = latestEnd - earliestStart // 計算 trace duration
	}
	return traceData
}

func setServiceName(traceData *TraceData) *TraceData {
	for i := range traceData.Data {
		for j := range traceData.Data[i].Spans {
			op := traceData.Data[i].Spans[j].OperationName
			if op == "RedisAddItem" || op == "RedisEmptyCart" || op == "RedisGetCart" {
				traceData.Data[i].Spans[j].ServiceName = "redis-cart"
			} else if serviceName, ok := traceData.Data[i].Processes[traceData.Data[i].Spans[j].ProcessID]; ok {
				traceData.Data[i].Spans[j].ServiceName = serviceName.ServiceName
			}
		}
	}
	return traceData
}

// populateParentFields fills in ParentService and ParentOperation for each span.
func populateParentFields(traceData *TraceData) *TraceData {
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

func GetTraceData(filename string) {
	start := time.Now().Add(-1*time.Minute).UnixNano() / 1000 // 10 分鐘前
	end := time.Now().UnixNano() / 1000                       // 現在時間

	traceData, err := fetchTraces("frontend", start, end)
	if err != nil {
		fmt.Println(err)
	}

	// Populate ParentService and ParentOperation
	traceData = populateParentFields(traceData)

	// replace ProcessID with actual service Name
	traceData = setServiceName(traceData)

	// calculate trace duration of each span
	traceData = setTraceDuration(traceData)

	// calculate average (predicte) duration of traces
	traceData = calculateAverageDuration(traceData)

	printJSON(traceData, filename)
}
