package main

import (
	"sync"
)

func RunPS_GWCA() {
	InitPSO()
	InitGWO()
	var wg sync.WaitGroup
	wg.Add(3)

	pso := NewPSO(300, Iterations)
	gwo := NewGWO(300, Iterations)
	factory := NewFactory(Iterations)
	go factory.Run(&wg)
	go pso.Optimize(&wg)
	go gwo.Optimize(&wg)

	wg.Wait()
}
