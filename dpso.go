package main

import (
	"fmt"
	"math/rand"
	"time"
)

type Particle struct {
	Solution     map[string]map[string]int
	Velocity     map[string]map[string]float64
	BestSolution map[string]map[string]int
	BestScore    float64
}

type DPSO struct {
	Particles    []Particle
	BestSolution map[string]map[string]int
	BestScore    float64
	NumParticles int
	MaxIter      int
}

var services = []string{
	"cartservice", "checkoutservice", "currencyservice", "emailservice",
	"frontend", "paymentservice", "productcatalogservice", "recommendationservice",
	"redis-cart", "shippingservice",
}

var traceData TraceData
var processTimeMap map[string]map[string]int64
var processTimeCloudMap map[string]map[string]int64

func Init() {
	loadJSONFile("resources_services.json", &serviceConstraints)
	loadJSONFile("resources_nodes.json", &nodeConstraints)

	loadJSONFile("path_durations.json", &traceData)
	loadJSONFile("processing_time_edge.json", &processTimeMap)
	loadJSONFile("processing_time_cloud.json", &processTimeCloudMap)

	// printJSON(serviceConstraints, "")
	// printJSON(nodeConstraints, "")
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

func (dpso *DPSO) Optimize() {
	w, c1, c2 := 0.5, 1.5, 1.5

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

func copySolution(dst, src map[string]map[string]int) {
	for node := range src {
		for service, value := range src[node] {
			dst[node][service] = value
		}
	}
}

func checkConstraints(solution map[string]map[string]int) bool {
	fmt.Println("enter checkConstraints()")
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
					fmt.Printf("Node %s exceeds CPU limit: %d/%d\n", node, totalCPU, nodeConstraint.CPU)
					return false
				}
				if totalMemory > nodeConstraint.Memory {
					fmt.Printf("Node %s exceeds Memory limit: %d/%d\n", node, totalMemory, nodeConstraint.Memory)
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

	var T = fitness(&traceData, solution, processTimeMap, processTimeCloudMap, probC)
	if T < 0 {
		fmt.Errorf("fitness() should not return negative value")
	}
	return T
}

func sigmoid(x float64) float64 {
	return 1 / (1 + 1/float64(1+rand.ExpFloat64()))
}

func RunDPSO() {
	Init()

	fmt.Println("enter NewDPSO()")
	dpso := NewDPSO(30, 1000)
	if dpso == nil {
		fmt.Println("asdf")
	}

	fmt.Println("enter Optimize()!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	dpso.Optimize()
}
