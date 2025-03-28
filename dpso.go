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

var nodes = []string{"vm1", "vm2", "vm3", "asus"}
var services = []string{
	"adservice", "cartservice", "checkoutservice", "currencyservice", "emailservice",
	"frontend", "paymentservice", "productcatalogservice", "recommendationservice",
	"redis-cart", "shippingservice",
}

func Init() {
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
		for i := range dpso.Particles {
			p := &dpso.Particles[i]
			for _, node := range nodes {
				for _, service := range services {
					r1, r2 := rand.Float64(), rand.Float64()
					p.Velocity[node][service] = w*p.Velocity[node][service] +
						c1*r1*float64(p.BestSolution[node][service]-p.Solution[node][service]) +
						c2*r2*float64(dpso.BestSolution[node][service]-p.Solution[node][service])
					p.Solution[node][service] = int(sigmoid(p.Velocity[node][service])*9) + 1
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

func randomSolution() map[string]map[string]int {
	solution := make(map[string]map[string]int)
	for _, node := range nodes {
		solution[node] = make(map[string]int)
		for _, service := range services {
			solution[node][service] = rand.Intn(3) + 1
		}
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

func evaluate(solution map[string]map[string]int) float64 {
	return 0 // Implement your evaluation logic here
}

func sigmoid(x float64) float64 {
	return 1 / (1 + 1/float64(1+rand.ExpFloat64()))
}

func main() {
	Init()
	dpso := NewDPSO(3, 60)
	dpso.Optimize()
}
