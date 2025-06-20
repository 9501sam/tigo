package main

import (
	"fmt"
	"log"
	"strings"
	// "encoding/json" // Assuming loadJSONFile and printJSON are in common.go or another accessible file
	// "os" // Assuming loadJSONFile and printJSON use os
	// "sync" // Assuming SharedMemory uses sync
)

// InvocationChainType represents a sequence of service calls in a chain
type InvocationChainType []string // Represents a single chain like ["serviceA", "serviceB"]

// ICCallKey represents a unique pair of calling service and called service for invocation chain analysis.
type ICCallKey struct {
	Caller string
	Callee string
}

// Global maps to store counts, renamed to avoid conflicts
var icCallCounts map[ICCallKey]int
var invocationChainCounts map[string]int

// init function to ensure maps are initialized
func init() {
	icCallCounts = make(map[ICCallKey]int)
	invocationChainCounts = make(map[string]int)
}

// analyzeSingleTrace is a helper to process one trace instance from TraceData.Data
// This function will do the heavy lifting for each individual trace.
func analyzeSingleTrace(traceInstance struct {
	TraceID           string `json:"traceID"`
	Duration          int64  `json:"duration"`
	PredictedDuration int64  `json:"predictedDuration"`
	Spans             []Span `json:"spans"` // Assuming Span struct is defined in common.go
	Processes         map[string]struct {
		ServiceName string `json:"serviceName"`
	} `json:"processes"`
}) {
	spanMap := make(map[string]Span)
	for _, span := range traceInstance.Spans {
		spanMap[span.SpanID] = span
	}

	// --- Logic to extract and count direct service calls (f_ij equivalent) ---
	processedCallsInTrace := make(map[ICCallKey]bool) // Use renamed key type
	for _, span := range traceInstance.Spans {
		if span.ParentService != "" && span.ParentService != span.ServiceName {
			caller := span.ParentService
			callee := span.ServiceName

			key := ICCallKey{Caller: caller, Callee: callee} // Use renamed key type
			// Increment for every occurrence of a direct call pair
			icCallCounts[key]++ // Use renamed map
			processedCallsInTrace[key] = true
		}
	}

	// --- Logic to extract and count Invocation Chains ---
	rootSpans := findRootSpans(traceInstance.Spans, spanMap)

	for _, root := range rootSpans {
		var currentChain InvocationChainType // Local variable for the current chain being built
		// Recursively build the chain from the root
		buildChainDFS(root, spanMap, &currentChain) // Pass *InvocationChainType

		if len(currentChain) > 0 {
			chainStr := fmt.Sprintf("%v", currentChain) // Convert []string to string for map key
			invocationChainCounts[chainStr]++
		}
	}
}

// extractInvocationChain is now the main entry point to process ALL traces in TraceData
func extractInvocationChain(allTraceData TraceData) { // Assuming TraceData struct is defined in common.go
	// Iterate through each individual trace instance within the Data slice
	for _, traceInstance := range allTraceData.Data {
		analyzeSingleTrace(traceInstance) // Process each trace one by one
	}
	// No return value here, as results are stored in global maps (invocationChainCounts, icCallCounts)
}

// Helper to find root spans in a trace
// Assumes Span struct is accessible
func findRootSpans(spans []Span, spanMap map[string]Span) []Span {
	isChild := make(map[string]bool)
	for _, span := range spans {
		for _, ref := range span.References {
			if ref.RefType == "CHILD_OF" {
				if _, ok := spanMap[ref.SpanID]; ok { // if parent is in this trace
					isChild[span.SpanID] = true
				}
			}
		}
	}

	var roots []Span
	for _, span := range spans {
		if !isChild[span.SpanID] {
			roots = append(roots, span)
		}
	}
	return roots
}

// buildChainDFS (simplified for a single primary chain)
// Accepts *InvocationChainType to match InvocationChainType as []string
func buildChainDFS(current Span, spanMap map[string]Span, chain *InvocationChainType) { // <--- 修正這裡的型別
	*chain = append(*chain, current.ServiceName)

	var children []Span
	for _, s := range spanMap {
		for _, ref := range s.References {
			if ref.RefType == "CHILD_OF" && ref.SpanID == current.SpanID {
				children = append(children, s)
				break
			}
		}
	}

	// Simplistic: just follow the first child found, assuming it's the main path.
	// For more robust chain extraction from complex traces, this logic needs to be more sophisticated.
	if len(children) > 0 {
		buildChainDFS(children[0], spanMap, chain)
	}
}

// *** for (6) and (7) *** //
// --- 輔助函式：根據公式 (7) 計算 C_d(ni) ---
// C_d(mi string) float64
// 計算服務 mi 的調用深度影響因子 C_d(ni)。
// 等於：以 mi 為呼叫者的直接呼叫總次數 / 所有直接呼叫的總次數。
func C_d(mi string) float64 {
	totalCallsFromMi := 0
	// 計算分子：以 mi 為呼叫者的直接呼叫總次數
	for key, count := range icCallCounts {
		if key.Caller == mi {
			totalCallsFromMi += count
		}
	}

	totalAllCalls := 0
	// 計算分母：所有直接呼叫的總次數
	// 我們需要檢查 icCallCounts 是否為空，避免除以零
	if len(icCallCounts) == 0 {
		return 0.0 // 如果沒有任何直接呼叫數據，返回 0
	}
	for _, count := range icCallCounts {
		totalAllCalls += count
	}

	if totalAllCalls == 0 {
		return 0.0 // 再次檢查，如果總數為零，返回 0
	}

	return float64(totalCallsFromMi) / float64(totalAllCalls)
}

// --- 輔助函式：計算 Num_IC(serviceName) ---
// Num_IC(serviceName string) int
// 計算在 invocationChainCounts 中包含 serviceName 的調用鏈的總出現次數。
func Num_IC(serviceName string) int {
	totalCount := 0
	for chainStr, count := range invocationChainCounts {
		// 將 chainStr 從 "[serviceA serviceB serviceC]" 轉換回 "serviceA serviceB serviceC"
		// 這樣我們可以用 strings.Contains 判斷 serviceName 是否在其中
		cleanedChainStr := strings.Trim(chainStr, "[]")
		chainServices := strings.Fields(cleanedChainStr) // 按空格分割以獲取單個服務名稱

		for _, s := range chainServices {
			if s == serviceName {
				totalCount += count
				break // 找到後就跳出內部迴圈，避免重複計數同一條鏈
			}
		}
	}
	return totalCount
}

// --- 輔助函式：計算 Num_IC(mj, mk) ---
// Num_IC_Pair(mj, mk string) int
// 計算同時包含 mj 和 mk 的調用鏈的總出現次數。
func Num_IC_Pair(mj, mk string) int {
	totalCount := 0
	for chainStr, count := range invocationChainCounts {
		cleanedChainStr := strings.Trim(chainStr, "[]")
		chainServices := strings.Fields(cleanedChainStr)

		mjFound := false
		mkFound := false

		for _, s := range chainServices {
			if s == mj {
				mjFound = true
			}
			if s == mk {
				mkFound = true
			}
		}

		if mjFound && mkFound {
			totalCount += count
		}
	}
	return totalCount
}

// --- 核心函式：根據公式 (6) 計算 DepIC(mj, mk) ---
// DepIC(mj, mk string) float64
// 計算服務 mj 和 mk 之間的 DepIC 值。
func DepIC(mj, mk string) float64 {
	// 獲取 C_d(mj) 和 C_d(mk)
	cd_mj := C_d(mj)
	cd_mk := C_d(mk)

	// 獲取 Num_IC(mj), Num_IC(mk) 和 Num_IC(mj, mk)
	num_ic_mj := Num_IC(mj)
	num_ic_mk := Num_IC(mk)
	num_ic_mj_mk := Num_IC_Pair(mj, mk)

	// 檢查分母是否為零，避免運行時錯誤
	term1 := 0.0
	if cd_mj != 0 && num_ic_mj != 0 {
		term1 = (1 / cd_mj) * (float64(num_ic_mj_mk) / float64(num_ic_mj))
	}

	term2 := 0.0
	if cd_mk != 0 && num_ic_mk != 0 {
		term2 = (1 / cd_mk) * (float64(num_ic_mj_mk) / float64(num_ic_mk))
	}

	return term1 + term2
}

// *** for (6) and (7) *** //

// RunDependency is the main entry point for the dependency analysis
// Assumes loadJSONFile and traceData are accessible (e.g., from common.go/main.go setup)
func RunDependency() {
	// Load your Jaeger trace JSON data
	// Assuming traceData is a global variable of type TraceData defined elsewhere (e.g., common.go or main.go)
	// And loadJSONFile is a function defined elsewhere (e.g., common.go)
	err := loadJSONFile("app.json", &traceData)
	if err != nil {
		log.Fatalf("Error loading trace data: %v", err)
	}

	// Now, call the main analysis function with the loaded TraceData
	extractInvocationChain(traceData)

	fmt.Println("--- Invocation Chain Counts ---")
	for chain, count := range invocationChainCounts {
		fmt.Printf("Chain: %s, Count: %d\n", chain, count)
	}

	fmt.Println("\n--- Direct Service Call Counts (f_ij equivalent) ---")
	for call, count := range icCallCounts { // Use renamed map
		fmt.Printf("Call: %s -> %s, Count: %d\n", call.Caller, call.Callee, count)
	}

	depIC_sp := DepIC("frontend", "checkoutservice")
	fmt.Printf("DepIC(\"frontend\", \"checkoutservice\"): %.4f\n", depIC_sp)
}
