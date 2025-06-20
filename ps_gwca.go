package main

import (
	// "fmt"
	"sync"
)

const (
	Iterations = 100
)

func RunPS_GWCA() {
	InitPSO()
	InitGWO()

	var abDone sync.WaitGroup

	// Channels for signaling
	aDone := make(chan struct{}, 1)
	bDone := make(chan struct{}, 1)
	done := make(chan struct{}, 3)  // Buffered to avoid blocking
	nextIter := make(chan struct{}) // Signal to start next iteration

	var wg sync.WaitGroup
	wg.Add(3)
	pso := NewPSO(300, Iterations)
	gwo := NewGWO(300, Iterations)
	factory := NewFactory(Iterations)
	go func() {
		defer wg.Done()
		pso.Optimize(&abDone, aDone, done, nextIter)
	}()
	go func() {
		defer wg.Done()
		gwo.Optimize(&abDone, bDone, done, nextIter)
	}()
	go func() {
		defer wg.Done()
		factory.Run(&abDone, aDone, bDone, done, nextIter)
	}()

	// Control iterations
	for i := 0; i < iternum; i++ {
		abDone.Add(2) // Add 2 at the beginning of each iteration for PSO and GWO
		// Wait for all three goroutines to signal completion of this iteration
		for j := 0; j < 3; j++ {
			<-done
		}
		// Signal all goroutines to start the next iteration
		for j := 0; j < 3; j++ {
			nextIter <- struct{}{}
		}
	}

	// Close channels to clean up
	close(aDone)
	close(bDone)
	close(done)
	close(nextIter)

	wg.Wait()
	printJSON(sharedMem.MergedFront[len(sharedMem.MergedFront)-1].BestSolution, "ps_gwca_solution.json")
}
