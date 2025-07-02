package algorithms

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"optimizer/analyzer"
	"optimizer/common"
	"os"
	"strconv"
	"time"
)

type DPSO struct {
	Particles    []common.Particle
	BestSolution map[string]map[string]int
	BestScore    float64
	NumParticles int
	MaxIter      int
}

var heatmap map[common.CallKey]float64
var traceData common.TraceData
var serviceConstraints common.ResourceConstraints
var nodeConstraints common.NodeConstraints

var processTimeMap map[string]map[string]int64
var processTimeCloudMap map[string]map[string]int64

func Init() {
	common.LoadJSONFile("app.json", &traceData)
	common.LoadJSONFile("resources_services.json", &serviceConstraints)
	common.LoadJSONFile("resources_nodes.json", &nodeConstraints)

	common.LoadJSONFile("processing_time_edge.json", &processTimeMap)
	common.LoadJSONFile("processing_time_cloud.json", &processTimeCloudMap)

	// callCounts = CountServiceCalls(traceData)
	heatmap, _ = analyzer.LoadDepICsFromCSV("depICs.csv")
}

func NewDPSO(numParticles, maxIter int) *DPSO {
	rand.Seed(time.Now().UnixNano())
	particles := make([]common.Particle, numParticles)
	bestSolution := make(map[string]map[string]int)
	for _, node := range common.Nodes {
		bestSolution[node] = make(map[string]int)
		for _, service := range common.Services {
			bestSolution[node][service] = 0
		}
	}
	bestScore := -1.0

	for i := range particles {
		particles[i] = common.Particle{
			Solution:     randomSolution(),
			Velocity:     makeVelocity(),
			BestSolution: make(map[string]map[string]int),
			BestScore:    -1.0,
		}
		// Initialize BestSolution maps
		for _, node := range common.Nodes {
			particles[i].BestSolution[node] = make(map[string]int)
		}
		common.CopySolution(particles[i].BestSolution, particles[i].Solution)
		score := evaluate(particles[i].Solution)
		particles[i].BestScore = score
		if score > bestScore {
			bestScore = score
			common.CopySolution(bestSolution, particles[i].Solution)
		}
	}

	return &DPSO{
		Particles:    particles,
		BestSolution: bestSolution,
		BestScore:    bestScore,
		NumParticles: numParticles,
		MaxIter:      maxIter,
	}
}

// writeCSVHeader checks if the CSV file exists and writes the header if it doesn't.
func writeCSVHeader(filePath string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer f.Close()
		writer := csv.NewWriter(f)
		defer writer.Flush()
		return writer.Write([]string{"Iteration", "BestScore"})
	}
	return nil
}

func (dpso *DPSO) Optimize() {
	w, c1, c2 := 0.5, 1.5, 1.5
	csvFileName := "dpso_optimization_results.csv" // Define your CSV file name

	if err := writeCSVHeader(csvFileName); err != nil {
		fmt.Printf("Error writing CSV header: %v\n", err)
		return // Or handle error as appropriate
	}
	// Open the CSV file in append mode. If it doesn't exist, it will be created.
	f, err := os.OpenFile(csvFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening CSV file: %v\n", err)
		return // Or handle error as appropriate
	}
	defer f.Close() // Ensure the file is closed when the function exits

	writer := csv.NewWriter(f)
	defer writer.Flush() // Ensure all buffered data is written to the file

	for iter := 0; iter < dpso.MaxIter; iter++ {
		fmt.Printf("\nIteration %d start!!!\n", iter)
		for i := range dpso.Particles {
			p := &dpso.Particles[i]
			for _, node := range common.Nodes {
				for _, service := range common.Services {
					r1, r2 := rand.Float64(), rand.Float64()
					p.Velocity[node][service] = w*p.Velocity[node][service] +
						c1*r1*float64(p.BestSolution[node][service]-p.Solution[node][service]) +
						c2*r2*float64(dpso.BestSolution[node][service]-p.Solution[node][service])
					threshold := sigmoid(p.Velocity[node][service])

					// fmt.Printf("threshold = %f\n", threshold)

					p.Solution[node][service] = 0
					if rand.Float64() < threshold {
						p.Solution[node][service]++
					}
					if rand.Float64() < threshold {
						p.Solution[node][service]++
					}
					if rand.Float64() < threshold {
						p.Solution[node][service]++
					}
				}
			}

			// small is better (faster)
			score := evaluate(p.Solution)
			if score < p.BestScore {
				p.BestScore = score
				common.CopySolution(p.BestSolution, p.Solution)
			}
			if score < dpso.BestScore {
				dpso.BestScore = score
				common.CopySolution(dpso.BestSolution, p.Solution)
			}
		}
		fmt.Printf("Iteration %d: Best Score = %f, BestSolution till this iteration be like: \n", iter, dpso.BestScore)
		// TODO: save iter, dpso.BestScore to a csv file
		record := []string{strconv.Itoa(iter), strconv.FormatFloat(dpso.BestScore, 'f', -1, 64)}
		if err := writer.Write(record); err != nil {
			fmt.Printf("Error writing record to CSV: %v\n", err)
			// Decide how to handle this error: continue, break, return
		}
		writer.Flush() // Flush after each write to ensure data is written immediately
		if iter == dpso.MaxIter-1 {
			common.PrintJSON(dpso.BestSolution, "")
		}
		fmt.Println("-----------------------------------------")
	}
}

func randomSolution() map[string]map[string]int {
	solution := make(map[string]map[string]int)
	for _, node := range common.Nodes {
		solution[node] = make(map[string]int)
	}

	for _, service := range common.Services {
		selectedNode := common.Nodes[rand.Intn(4)]
		solution[selectedNode][service] = 1
	}
	return solution
}

func makeVelocity() map[string]map[string]float64 {
	velocity := make(map[string]map[string]float64)
	for _, node := range common.Nodes {
		velocity[node] = make(map[string]float64)
		for _, service := range common.Services {
			velocity[node][service] = 0.0
		}
	}
	return velocity
}

func checkConstraints(solution map[string]map[string]int) bool {
	// fmt.Println("enter checkConstraints()")
	// common.PrintJSON(solution, "")
	if solution["asus"]["frontend"] > 0 {
		return false
	}

	for _, node := range common.Nodes {
		for _, services := range solution {
			totalCPU := 0
			totalMemory := 0

			// 計算該 VM 的資源使用量
			for service, replicas := range services {
				if constraint, exists := serviceConstraints[service]; exists {
					totalCPU += constraint.CPU * replicas
					totalMemory += constraint.Memory * replicas
				} else {
					fmt.Printf("Warning: Service %s not found in constraints\n", service)
				}
			}

			// 取得該 VM 的資源限制
			if nodeConstraint, exists := nodeConstraints[node]; exists {
				if totalCPU > nodeConstraint.CPU {
					// fmt.Printf("Node %s exceeds CPU limit: %d/%d\n", node, totalCPU, nodeConstraint.CPU)
					return false
				}
				if totalMemory > nodeConstraint.Memory {
					// fmt.Printf("Node %s exceeds Memory limit: %d/%d\n", node, totalMemory, nodeConstraint.Memory)
					return false
				}
			} else {
				fmt.Printf("Warning: Node %s not found in constraints\n", node)
			}
		}
	}
	return true
}

func evaluate(solution map[string]map[string]int) float64 {
	probC := CalculateProbability(solution, "asus")

	// fmt.Printf("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	// common.PrintJSON(probC, "")
	if !checkConstraints(solution) {
		return 999999999 // big number as penalty (means very slow)
	}
	// TODO: we should use fittness()
	// 1. traceData: traces.json
	// 2. deploymentConfig: solution
	// 3. processTimeMap: process_time_edge.json
	// 4. processTimeCloudMap: process_time_cloud.json
	// 5. probC

	// return 0.0

	// this is for call count heatmap
	// var T = fitness(&traceData, solution, processTimeMap, processTimeCloudMap, probC, callCounts)
	// if T < 0 {
	// 	fmt.Errorf("fitness() should not return negative value")
	// }

	// this is for depIC heatmap

	var T = fitness(&traceData, solution, processTimeMap, processTimeCloudMap, probC, heatmap)
	if T < 0 {
		fmt.Errorf("fitness() should not return negative value")
	}
	return T
}

func sigmoid(x float64) float64 {
	return 1 / (1 + 1/float64(1+rand.ExpFloat64()))
}

func RunDPSO() {
	start := time.Now()
	Init()

	fmt.Println("enter NewDPSO()")
	dpso := NewDPSO(30, 100)
	if dpso == nil {
		fmt.Println("asdf")
	}

	fmt.Println("enter Optimize()!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	dpso.Optimize()

	common.PrintJSON(dpso.BestSolution, "dpso_dep_solution_version2.json")
	elapsed := time.Since(start) // 計算花費時間
	fmt.Printf("execution time(dpso): %s\n\n", elapsed)
	// UpDateDeploymentsByJSON("dpso_solution.json")
}
