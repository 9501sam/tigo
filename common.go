package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
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

var services = []string{
	"cartservice", "checkoutservice", "currencyservice", "emailservice",
	"frontend", "paymentservice", "productcatalogservice", "recommendationservice",
	"redis-cart", "shippingservice",
}

var traceData TraceData
var processTimeMap map[string]map[string]int64
var processTimeCloudMap map[string]map[string]int64

// var callCounts map[CallKey]int

type Constraints struct {
	CPU    int `json:"cpu"`
	Memory int `json:"memory"`
}

type ResourceConstraints map[string]Constraints
type NodeConstraints map[string]Constraints

var serviceConstraints ResourceConstraints
var nodeConstraints NodeConstraints

func copySolution(dst, src map[string]map[string]int) {
	for node := range src {
		for service, value := range src[node] {
			dst[node][service] = value
		}
	}
}

type Particle struct {
	Solution     map[string]map[string]int
	Velocity     map[string]map[string]float64
	BestSolution map[string]map[string]int
	BestScore    float64
}

// SharedMemory holds Pareto fronts and synchronization data
type SharedMemory struct {
	sync.RWMutex
	PSOFront    []Particle
	GWOFront    []Particle
	MergedFront []Particle
	Used        bool
	Transform   int // 0: no transform, 1: PSO to GWO, 2: GWO to PSO
}

var sharedMem SharedMemory

func sumServiceInstances(filename string) {
	// // Read the JSON file
	// data, err := ioutil.ReadFile(filename)
	// if err != nil {
	// 	return nil, fmt.Errorf("error reading file: %v", err)
	// }

	// // Parse JSON into a map
	// var nodeServices map[string]map[string]int
	// err = json.Unmarshal(data, &nodeServices)
	// if err != nil {
	// 	return nil, fmt.Errorf("error parsing JSON: %v", err)
	// }
	var nodeServices map[string]map[string]int
	loadJSONFile(filename, &nodeServices)

	// Initialize the result map to store service totals
	serviceTotals := make(map[string]int)

	// Sum instances for each service across all nodes
	for nodeName, services := range nodeServices {
		fmt.Printf("Processing node: %s\n", nodeName)
		for serviceName, instances := range services {
			serviceTotals[serviceName] += instances
		}
	}
	printJSON(serviceTotals, "")
}

func randomSolutionForPS_GWCA() map[string]map[string]int {
	solution := make(map[string]map[string]int)
	for _, node := range nodes {
		solution[node] = make(map[string]int)
		for _, service := range services {
			solution[node][service] = 0
		}
	}

	instances := make(map[string]int)
	instances["cartservice"] = 6
	instances["checkoutservice"] = 8
	instances["currencyservice"] = 8
	instances["emailservice"] = 4
	instances["frontend"] = 7
	instances["paymentservice"] = 5
	instances["productcatalogservice"] = 6
	instances["recommendationservice"] = 9
	instances["redis-cart"] = 5
	instances["shippingservice"] = 7

	for _, service := range services {
		// Generate random total instances for this service (1 to 10, adjust range as needed)
		totalInstances := rand.Intn(10) + 1

		// Randomly distribute the instances across nodes
		for i := 0; i < totalInstances; i++ {
			selectedNode := nodes[rand.Intn(len(nodes))]
			solution[selectedNode][service]++
		}
	}

	return solution
}
