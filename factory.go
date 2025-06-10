package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type Factory struct {
	MaxIter int
}

func NewFactory(maxIter int) *Factory {
	return &Factory{
		MaxIter: maxIter,
	}
}

func (f *Factory) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	count := 0
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < f.MaxIter; i++ {
		sharedMem.Lock()
		psoFront := sharedMem.PSOFront
		gwoFront := sharedMem.GWOFront
		sharedMem.Unlock()

		// Merge and reorder Pareto fronts
		mergedFront := []Particle{}
		for _, p := range psoFront {
			mergedFront = updateParetoFront(mergedFront, p)
		}
		for _, p := range gwoFront {
			mergedFront = updateParetoFront(mergedFront, p)
		}

		// Evaluate algorithm performance
		if sharedMem.Transform == 0 {
			if len(mergedFront) > 0 {
				worstIdx := 0
				worstScore := -math.Inf(1)
				for j, p := range mergedFront {
					if p.BestScore > worstScore {
						worstScore = p.BestScore
						worstIdx = j
					}
				}
				fmt.Printf("worseIdx = %d\n", worstIdx)
				// Simplified: Assume worst particle indicates source algorithm
				// In practice, track source (PSO/GWO) during merge
				if rand.Float64() < 0.5 { // Placeholder for PSO/GWO identification
					count++
				} else {
					count--
				}
			}
		}

		// Check for transformation
		if count > f.MaxIter/2 {
			sharedMem.Lock()
			sharedMem.Transform = 1 // PSO to GWO
			sharedMem.Unlock()
		} else if count < -f.MaxIter/2 {
			sharedMem.Lock()
			sharedMem.Transform = 2 // GWO to PSO
			sharedMem.Unlock()
		}

		// Update shared memory with merged Pareto front
		sharedMem.Lock()
		sharedMem.MergedFront = mergedFront
		sharedMem.Used = true
		sharedMem.Unlock()

		time.Sleep(time.Millisecond * 10)
	}
}

func updateParetoFront(front []Particle, candidate Particle) []Particle {
	// Simplified Pareto dominance check
	for i := 0; i < len(front); i++ {
		if dominates(front[i], candidate) {
			return front
		}
	}
	return append(front, candidate)
}

func dominates(p1, p2 Particle) bool {
	// Placeholder: Implement actual multi-objective dominance check
	return p1.BestScore <= p2.BestScore
}
