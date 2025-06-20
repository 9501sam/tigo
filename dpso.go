package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

type DPSO struct {
	Particles    []Particle
	BestSolution map[string]map[string]int
	BestScore    float64
	NumParticles int
	MaxIter      int
}

func Init() {
	loadJSONFile("app.json", &traceData)
	loadJSONFile("resources_services.json", &serviceConstraints)
	loadJSONFile("resources_nodes.json", &nodeConstraints)

	loadJSONFile("processing_time_edge.json", &processTimeMap)
	loadJSONFile("processing_time_cloud.json", &processTimeCloudMap)

	callCounts = CountServiceCalls(traceData)
}

func NewDPSO(numParticles, maxIter int) *DPSO {
	rand.Seed(time.Now().UnixNano())
	particles := make([]Particle, numParticles)
	bestSolution := make(map[string]map[string]int)
	for _, node := range nodes {
		bestSolution[node] = make(map[string]int)
		for _, service := range services {
			bestSolution[node][service] = 0
		}
	}
	bestScore := -1.0

	for i := range particles {
		particles[i] = Particle{
			Solution:     randomSolution(),
			Velocity:     makeVelocity(),
			BestSolution: make(map[string]map[string]int),
			BestScore:    -1.0,
		}
		// Initialize BestSolution maps
		for _, node := range nodes {
			particles[i].BestSolution[node] = make(map[string]int)
		}
		copySolution(particles[i].BestSolution, particles[i].Solution)
		score := evaluate(particles[i].Solution)
		particles[i].BestScore = score
		if score > bestScore {
			bestScore = score
			copySolution(bestSolution, particles[i].Solution)
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
			for _, node := range nodes {
				for _, service := range services {
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
				copySolution(p.BestSolution, p.Solution)
			}
			if score < dpso.BestScore {
				dpso.BestScore = score
				copySolution(dpso.BestSolution, p.Solution)
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
			printJSON(dpso.BestSolution, "")
		}
		fmt.Println("-----------------------------------------")
	}
}

func randomSolution() map[string]map[string]int {
	solution := make(map[string]map[string]int)
	for _, node := range nodes {
		solution[node] = make(map[string]int)
	}

	for _, service := range services {
		selectedNode := nodes[rand.Intn(4)]
		solution[selectedNode][service] = 1
	}
	return solution
}

func makeVelocity() map[string]map[string]float64 {
	velocity := make(map[string]map[string]float64)
	for _, node := range nodes {
		velocity[node] = make(map[string]float64)
		for _, service := range services {
			velocity[node][service] = 0.0
		}
	}
	return velocity
}

func checkConstraints(solution map[string]map[string]int) bool {
	// fmt.Println("enter checkConstraints()")
	// printJSON(solution, "")

	for _, node := range nodes {
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
	// printJSON(probC, "")
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

	var T = fitness(&traceData, solution, processTimeMap, processTimeCloudMap, probC, callCounts)
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

	printJSON(dpso.BestSolution, "dpso_solution.json")
	elapsed := time.Since(start) // 計算花費時間
	fmt.Printf("execution time(dpso): %s\n\n", elapsed)
	// UpDateDeploymentsByJSON("dpso_solution.json")
}
