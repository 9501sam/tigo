package main

import (
	"math/rand"
	"sync"
)

const (
	Iterations = 100
)

func randomSolutionForPS_GWCA() map[string]map[string]int {
	solution := make(map[string]map[string]int)
	for _, node := range nodes {
		solution[node] = make(map[string]int)
		for _, service := range services {
			solution[node][service] = 0
		}
	}

	instances := make(map[string]int)
	instances["cartservice"] = 6
	instances["checkoutservice"] = 8
	instances["currencyservice"] = 8
	instances["emailservice"] = 4
	instances["frontend"] = 7
	instances["paymentservice"] = 5
	instances["productcatalogservice"] = 6
	instances["recommendationservice"] = 9
	instances["redis-cart"] = 5
	instances["shippingservice"] = 7

	for _, service := range services {
		// Generate random total instances for this service (1 to 10, adjust range as needed)
		totalInstances := rand.Intn(10) + 1

		// Randomly distribute the instances across nodes
		for i := 0; i < totalInstances; i++ {
			selectedNode := nodes[rand.Intn(len(nodes))]
			solution[selectedNode][service]++
		}
	}

	return solution
}

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
