package main

import (
	"sync"
)

func RunPS_GWCA() {
	InitPSO()
	InitGWO()
	var wg sync.WaitGroup
	wg.Add(3)
	// go pso.Optimize(&wg)

	wg.Wait()
}
