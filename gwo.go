package main

import (
	// "encoding/json"
	"fmt"
	"math"
	"math/rand"
	// "os"
	"time"
)

const (
	// Iterations    = 100
	PhysicalNodes = 100
	Microservices = 10 // Matches len(services)
	// Omega         = 0.45
	// C1 = 0.1
)

// GWOParticle represents a GWO solution
type GWOParticle struct {
	Particle
	// Solution     map[string]map[string]int // [pm_i][ms_j] = number of containers
	// BestSolution map[string]map[string]int // Personal best
	// BestScore    float64
}

func InitGWO() {
	loadJSONFile("app.json", &traceData)
	loadJSONFile("resources_services.json", &serviceConstraints)
	loadJSONFile("resources_nodes.json", &nodeConstraints)

	loadJSONFile("processing_time_edge.json", &processTimeMap)
	loadJSONFile("processing_time_cloud.json", &processTimeCloudMap)
}

// GWO represents the GWO algorithm state
type GWO struct {
	Particles    []GWOParticle
	Alpha        GWOParticle // Best solution
	Beta         GWOParticle // Second best
	Delta        GWOParticle // Third best
	ParetoFront  []GWOParticle
	NumParticles int
	MaxIter      int
}

func NewGWO(numParticles, maxIter int) *GWO {
	rand.Seed(time.Now().UnixNano())
	particles := make([]GWOParticle, numParticles)

	// Initialize nodes if not already set
	if len(nodes) == 0 {
		nodes = make([]string, PhysicalNodes)
		for i := 0; i < PhysicalNodes; i++ {
			nodes[i] = fmt.Sprintf("pm%d", i+1)
		}
	}

	// Initialize particles
	for i := range particles {
		solution := randomSolution()
		bestSolution := make(map[string]map[string]int)
		for _, node := range nodes {
			bestSolution[node] = make(map[string]int)
			for _, service := range services {
				bestSolution[node][service] = solution[node][service]
			}
		}
		score := evaluate(solution)
		particles[i] = GWOParticle{
			Particle: Particle{
				Solution:     solution,
				BestSolution: bestSolution,
				BestScore:    score,
			},
		}
	}

	// Initialize alpha, beta, delta
	var alpha, beta, delta GWOParticle
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
		ParetoFront:  []GWOParticle{alpha}, // Initial Pareto front
		NumParticles: numParticles,
		MaxIter:      maxIter,
	}
}
func (gwo *GWO) Optimize() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < gwo.MaxIter; i++ {
		// Update a (linearly decreases from 0.8 to 0.2)
		a := 0.8 - float64(i)/float64(gwo.MaxIter)*(0.8-0.2)

		// Update particles
		for j := range gwo.Particles {
			if rand.Float64() < a {
				// Transfer operation for exploration
				transferOperation(&gwo.Particles[j].Particle)
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
				copyOperation(&gwo.Particles[j].Particle, leader.Solution, rows)
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
	}
	fmt.Printf("Final GWO Pareto front size: %d, Alpha Score: %.2f\n", len(gwo.ParetoFront), gwo.Alpha.BestScore)
}

func RunGWO() {
	InitGWO()
	gwo := NewGWO(30, 100)
	gwo.Optimize()
}
