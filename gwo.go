package main

import (
	// "encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	// "os"
	"time"
)

const (
	PhysicalNodes = 100
	Microservices = 10 // Matches len(services)
	// Omega         = 0.45
	// C1 = 0.1
)

func InitGWO() {
	loadJSONFile("app.json", &traceData)
	loadJSONFile("resources_services.json", &serviceConstraints)
	loadJSONFile("resources_nodes.json", &nodeConstraints)

	loadJSONFile("processing_time_edge.json", &processTimeMap)
	loadJSONFile("processing_time_cloud.json", &processTimeCloudMap)
}

// GWO represents the GWO algorithm state
type GWO struct {
	Particles    []Particle
	Alpha        Particle // Best solution
	Beta         Particle // Second best
	Delta        Particle // Third best
	ParetoFront  []Particle
	NumParticles int
	MaxIter      int
}

func NewGWO(numParticles, maxIter int) *GWO {
	rand.Seed(time.Now().UnixNano())
	particles := make([]Particle, numParticles)

	// Initialize nodes if not already set
	if len(nodes) == 0 {
		nodes = make([]string, PhysicalNodes)
		for i := 0; i < PhysicalNodes; i++ {
			nodes[i] = fmt.Sprintf("pm%d", i+1)
		}
	}

	// Initialize particles
	for i := range particles {
		solution := randomSolutionForPS_GWCA()
		bestSolution := make(map[string]map[string]int)
		for _, node := range nodes {
			bestSolution[node] = make(map[string]int)
			for _, service := range services {
				bestSolution[node][service] = solution[node][service]
			}
		}
		score := evaluate(solution)
		particles[i] = Particle{
			Solution:     solution,
			BestSolution: bestSolution,
			BestScore:    score,
		}
	}

	// Initialize alpha, beta, delta
	var alpha, beta, delta Particle
	alpha.BestScore = math.Inf(1)
	beta.BestScore = math.Inf(1)
	delta.BestScore = math.Inf(1)
	for _, p := range particles {
		if p.BestScore < alpha.BestScore {
			delta = beta
			beta = alpha
			alpha = p
		} else if p.BestScore < beta.BestScore {
			delta = beta
			beta = p
		} else if p.BestScore < delta.BestScore {
			delta = p
		}
	}

	return &GWO{
		Particles:    particles,
		Alpha:        alpha,
		Beta:         beta,
		Delta:        delta,
		ParetoFront:  []Particle{alpha}, // Initial Pareto front
		NumParticles: numParticles,
		MaxIter:      maxIter,
	}
}
func (gwo *GWO) Optimize(abDone *sync.WaitGroup, bDone chan<- struct{}, done chan<- struct{}, nextIter chan struct{}) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < gwo.MaxIter; i++ {
		// // Update a (linearly decreases from 0.8 to 0.2)
		a := 0.8 - float64(i)/float64(gwo.MaxIter)*(0.8-0.2)

		//*** Communicate with Shared Memory ***///
		gwo.ParetoFront = []Particle{}
		for _, p := range gwo.Particles {
			gwo.ParetoFront = updateParetoFront(gwo.ParetoFront, p)
		}

		sharedMem.Lock()
		sharedMem.GWOFront = gwo.ParetoFront
		sharedMem.Unlock()

		sharedMem.Lock()
		if sharedMem.Transform == 2 {
			pso := NewPSO(gwo.NumParticles, gwo.MaxIter-i)
			for j := 0; j < gwo.NumParticles/2; j++ {
				// pso.Particles[j] = PSOParticle{Particle: gwo.Particles[j].Particle} // TODO
				pso.Particles[j] = gwo.Particles[j]
			}
			sharedMem.Transform = 0
			sharedMem.Unlock()
			pso.Optimize(abDone, bDone, done, nextIter)
			return
		}
		sharedMem.Unlock()

		sharedMem.RLock()
		newFront := sharedMem.MergedFront
		sharedMem.RUnlock()

		if len(newFront) > 0 {
			worstIdx := 0
			worstScore := -math.Inf(1)
			for j, p := range gwo.Particles {
				if p.BestScore > worstScore {
					worstScore = p.BestScore
					worstIdx = j
				}
			}
			randIdx := rand.Intn(len(newFront))
			gwo.Particles[worstIdx].Solution = make(map[string]map[string]int)
			for _, pm := range nodes {
				gwo.Particles[worstIdx].Solution[pm] = make(map[string]int)
			}
			copySolution(gwo.Particles[worstIdx].Solution, newFront[randIdx].Solution)
			gwo.Particles[worstIdx].BestScore = evaluate(gwo.Particles[worstIdx].Solution)
		}
		bDone <- struct{}{} // Signal that critical section B is done
		abDone.Done()       // Signal that B is done for C to proceed

		//*** Original GWO Part ***///
		for j := range gwo.Particles {
			if rand.Float64() < a {
				// Transfer operation for exploration
				transferOperation(&gwo.Particles[j])
			} else if len(gwo.ParetoFront) > 0 {
				// Copy operation from alpha, beta, or delta
				leader := gwo.Alpha
				switch rand.Intn(3) {
				case 1:
					leader = gwo.Beta
				case 2:
					leader = gwo.Delta
				}
				rows := selectRandomRows(1) // Single row as per paper
				copyOperation(&gwo.Particles[j], leader.Solution, rows)
			}

			// Update personal best
			if score := evaluate(gwo.Particles[j].Solution); score < gwo.Particles[j].BestScore {
				gwo.Particles[j].BestScore = score
				gwo.Particles[j].BestSolution = make(map[string]map[string]int)
				for _, pm := range nodes {
					gwo.Particles[j].BestSolution[pm] = make(map[string]int)
				}
				copySolution(gwo.Particles[j].BestSolution, gwo.Particles[j].Solution)
			}

			// Update alpha, beta, delta
			if gwo.Particles[j].BestScore < gwo.Alpha.BestScore {
				gwo.Delta = gwo.Beta
				gwo.Beta = gwo.Alpha
				gwo.Alpha = gwo.Particles[j]
			} else if gwo.Particles[j].BestScore < gwo.Beta.BestScore {
				gwo.Delta = gwo.Beta
				gwo.Beta = gwo.Particles[j]
			} else if gwo.Particles[j].BestScore < gwo.Delta.BestScore {
				gwo.Delta = gwo.Particles[j]
			}
		}
		fmt.Println("gwo")
		// Signal completion of this iteration
		done <- struct{}{}
		// Wait for the next iteration signal
		<-nextIter
	}
	fmt.Printf("Final GWO Pareto front size: %d, Alpha Score: %.2f\n", len(gwo.ParetoFront), gwo.Alpha.BestScore)
	printJSON(gwo.Alpha.BestSolution, "")
}
