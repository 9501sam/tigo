package main

import (
	// "fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	Iterations = 100
	Omega      = 0.45
	C1         = 0.1
)

type PSO struct {
	Particles    []Particle
	BestSolution map[string]map[string]int // gbest
	BestScore    float64
	ParetoFront  []Particle
	NumParticles int
	MaxIter      int
}

func NewPSO(numParticles, maxIter int) *PSO {
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

func (pso *PSO) Optimize(wg *sync.WaitGroup) {
	for i := 0; i < pso.MaxIter; i++ {
		//*** Communicate with Shared Memory ***///
		pso.ParetoFront = []Particle{}
		for _, p := range pso.Particles {
			pso.ParetoFront = updateParetoFront(pso.ParetoFront, p)
		}

		sharedMem.Lock()
		sharedMem.PSOFront = pso.ParetoFront
		sharedMem.Unlock()

		for {
			sharedMem.RLock()
			if sharedMem.Used {
				sharedMem.RUnlock()
				break
			}
			sharedMem.RUnlock()
			time.Sleep(time.Millisecond * 10)
		}

		sharedMem.Lock()
		if sharedMem.Transform == 1 {
			gwo := NewGWO(pso.NumParticles, pso.MaxIter)
			for j := 0; j < pso.NumParticles/2; j++ {
				// gwo.Particles[j] = GWOParticle{Particle: pso.Particles[j].Particle}
				gwo.Particles[j] = pso.Particles[j]
			}
			sharedMem.Transform = 0
			sharedMem.Unlock()
			gwo.Optimize(wg)
			return
		}
		sharedMem.Unlock()

		sharedMem.RLock()
		newFront := sharedMem.MergedFront
		sharedMem.RUnlock()

		if len(newFront) > 0 {
			worstIdx := 0
			worstScore := -math.Inf(1)
			for j, p := range pso.Particles {
				if p.BestScore > worstScore {
					worstScore = p.BestScore
					worstIdx = j
				}
			}
			randIdx := rand.Intn(len(newFront))
			pso.Particles[worstIdx].Solution = make(map[string]map[string]int)
			for _, pm := range nodes {
				pso.Particles[worstIdx].Solution[pm] = make(map[string]int)
			}
			copySolution(pso.Particles[worstIdx].Solution, newFront[randIdx].Solution)
			pso.Particles[worstIdx].BestScore = evaluate(pso.Particles[worstIdx].Solution)
		}

		//*** Original PSO Part ***///
		for j := range pso.Particles {
			transferOperation(&pso.Particles[j])
			pbestRows := selectRandomRows(int(C1 * float64(len(services))))
			copyOperation(&pso.Particles[j], pso.Particles[j].BestSolution, pbestRows)
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
}
