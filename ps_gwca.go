package main

import (
	// "fmt"
	"math/rand"
	"time"
)

const (
	Iterations = 100
	Omega      = 0.45
	C1         = 0.1
)

type PSOParticle struct {
	Particle
	// Solution     map[string]map[string]int // [pm_i][ms_j] = number of containers of microservice i on node j
	// BestSolution map[string]map[string]int // pbest
	// BestScore    float64
}

type PSO struct {
	Particles    []PSOParticle
	BestSolution map[string]map[string]int // gbest
	BestScore    float64
	NumParticles int
	MaxIter      int
}

func NewPSO(numParticles, maxIter int) *PSO {
	rand.Seed(time.Now().UnixNano())
	particles := make([]PSOParticle, numParticles)
	bestSolution := make(map[string]map[string]int)
	for _, node := range nodes {
		bestSolution[node] = make(map[string]int)
		for _, service := range services {
			bestSolution[node][service] = 0

		}

	}
	bestScore := -1.0
	for i := range particles {
		particles[i] = PSOParticle{
			Particle: Particle{
				Solution:     randomSolution(),
				BestSolution: make(map[string]map[string]int),
				BestScore:    -1.0,
			},
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
	return &PSO{
		Particles:    particles,
		BestSolution: bestSolution,
		BestScore:    bestScore,
		NumParticles: numParticles,
		MaxIter:      maxIter,
	}
}

func InitPSO() {
	loadJSONFile("app.json", &traceData)
	loadJSONFile("resources_services.json", &serviceConstraints)
	loadJSONFile("resources_nodes.json", &nodeConstraints)

	loadJSONFile("processing_time_edge.json", &processTimeMap)
	loadJSONFile("processing_time_cloud.json", &processTimeCloudMap)
}

// func evaluate(solution map[string]map[string]int) float64 {
// 	// do not worry about what happened here
// }

// transferOperation moves containers randomly
func transferOperation(p *Particle) {
	rowsToTransfer := selectRandomRows(int(Omega * float64(len(services))))

	for _, msIdx := range rowsToTransfer {
		ms := services[msIdx] // get one ms to doing transfer
		// Collect total containers for the microservice
		totalContainers := 0
		for _, pm := range nodes {
			totalContainers += p.Solution[pm][ms]
		}
		// Clear current assignments
		for _, pm := range nodes {
			p.Solution[pm][ms] = 0
		}
		// Randomly redistribute containers
		pmIDs := make([]string, 0, len(p.Solution))
		for _, pm := range nodes {
			pmIDs = append(pmIDs, pm)
		}
		for totalContainers > 0 {
			newPM := pmIDs[rand.Intn(len(pmIDs))]
			p.Solution[newPM][ms]++
			totalContainers--
		}
	}
	p.BestScore = evaluate(p.Solution)
}

// copyOperation copies rows from a reference solution
func copyOperation(p *Particle, ref map[string]map[string]int, rows []int) {
	for _, msIdx := range rows {
		ms := services[msIdx]
		for pm := range p.Solution {
			p.Solution[pm][ms] = ref[pm][ms]
		}
	}
	p.BestScore = evaluate(p.Solution)
}

// selectRandomRows selects n random indices
func selectRandomRows(n int) []int {
	rows := rand.Perm(len(services))
	if n > len(services) {
		n = len(services)
	}
	return rows[:n]
}

func (pso *PSO) Optimize() {
	for i := 0; i < pso.MaxIter; i++ {
		// Update particles
		for j := range pso.Particles {
			transferOperation(&pso.Particles[j].Particle)
			pbestRows := selectRandomRows(int(C1 * float64(len(services))))
			copyOperation(&pso.Particles[j].Particle, pso.Particles[j].BestSolution, pbestRows)
			// Update pbest
			if score := evaluate(pso.Particles[j].Solution); score < pso.Particles[j].BestScore {
				pso.Particles[j].BestScore = score
				pso.Particles[j].BestSolution = make(map[string]map[string]int)

				// Initialize nested maps before copying
				for pm := range pso.Particles[j].Solution {
					pso.Particles[j].BestSolution[pm] = make(map[string]int)
				}

				copySolution(pso.Particles[j].BestSolution, pso.Particles[j].Solution)
			}

			// Update gbest
			if pso.Particles[j].BestScore < pso.BestScore {
				pso.BestScore = pso.Particles[j].BestScore
				pso.BestSolution = make(map[string]map[string]int)

				// Initialize nested maps before copying
				for pm := range pso.Particles[j].BestSolution {
					pso.BestSolution[pm] = make(map[string]int)
				}

				copySolution(pso.BestSolution, pso.Particles[j].BestSolution)
			}
		}
	}
	// Optimize finish
}

func RunPSO() {
	InitPSO()
	pso := NewPSO(30, 100)
	// go pso.Optimize()
	pso.Optimize()
}
