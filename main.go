package main

import (
	"fmt"
	"math/rand"
	"time"
)

type Particle struct {
	Solution     [][]int
	Velocity     [][]float64
	BestSolution [][]int
	BestScore    float64
}

type DPSO struct {
	Particles    []Particle
	BestSolution [][]int
	BestScore    float64
	NumParticles int
	NumNodes     int
	NumServices  int
	MaxIter      int
}

func NewDPSO(numParticles, numNodes, numServices, maxIter int) *DPSO {
	rand.Seed(time.Now().UnixNano())
	particles := make([]Particle, numParticles)
	bestSolution := make([][]int, numNodes)
	for i := range bestSolution {
		bestSolution[i] = make([]int, numServices)
	}
	bestScore := -1.0

	for i := range particles {
		particles[i] = Particle{
			Solution:     randomSolution(numNodes, numServices),
			Velocity:     makeVelocity(numNodes, numServices),
			BestSolution: make([][]int, numNodes), // 確保分配外層 slice
			BestScore:    -1.0,
		}
		for n := range particles[i].BestSolution { // 分配內層 slice
			particles[i].BestSolution[n] = make([]int, numServices)
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
		NumNodes:     numNodes,
		NumServices:  numServices,
		MaxIter:      maxIter,
	}
}

func (dpso *DPSO) Optimize() {
	w, c1, c2 := 0.5, 1.5, 1.5

	for iter := 0; iter < dpso.MaxIter; iter++ {
		for i := range dpso.Particles {
			p := &dpso.Particles[i]
			for n := 0; n < dpso.NumNodes; n++ {
				for s := 0; s < dpso.NumServices; s++ {
					r1, r2 := rand.Float64(), rand.Float64()
					p.Velocity[n][s] = w*p.Velocity[n][s] + c1*r1*float64(p.BestSolution[n][s]-p.Solution[n][s]) + c2*r2*float64(dpso.BestSolution[n][s]-p.Solution[n][s])
					p.Solution[n][s] = int(sigmoid(p.Velocity[n][s])*9) + 1
				}
			}
			score := evaluate(p.Solution)
			if score > p.BestScore {
				p.BestScore = score
				copySolution(p.BestSolution, p.Solution)
			}
			if score > dpso.BestScore {
				dpso.BestScore = score
				copySolution(dpso.BestSolution, p.Solution)
			}
		}
		fmt.Printf("Iteration %d: Best Score = %f\n", iter, dpso.BestScore)
	}
}

func randomSolution(numNodes, numServices int) [][]int {
	solution := make([][]int, numNodes)
	for i := range solution {
		solution[i] = make([]int, numServices)
		for j := range solution[i] {
			solution[i][j] = rand.Intn(10) + 1
		}
	}
	return solution
}

func makeVelocity(numNodes, numServices int) [][]float64 {
	velocity := make([][]float64, numNodes)
	for i := range velocity {
		velocity[i] = make([]float64, numServices)
	}
	return velocity
}

func copySolution(dst, src [][]int) {
	for i := range src {
		copy(dst[i], src[i])
	}
}

func evaluate(solution [][]int) float64 {
	return 0.0 // Placeholder for actual evaluation function
}

func sigmoid(x float64) float64 {
	return 1 / (1 + 1/float64(1+rand.ExpFloat64()))
}

func main() {
	dpso := NewDPSO(30, 3, 11, 1000) // 3 nodes, 11 services, 50 iterations
	dpso.Optimize()

	fmt.Println("Best Solution:", dpso.BestSolution, "Score:", dpso.BestScore)
}
