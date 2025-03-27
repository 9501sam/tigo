package main

// import "fmt"

// ConnectionStats holds latency and bandwidth between two nodes
type ConnectionStats struct {
	Latency   int // in milliseconds
	Bandwidth int // in Mbps
}

type LatencyBandwidthMap struct {
	Stats map[string]map[string]ConnectionStats // Map of from-node to to-node stats
}

// Constructor to initialize the map
func NewLatencyBandwidthMap() *LatencyBandwidthMap {
	return &LatencyBandwidthMap{
		Stats: make(map[string]map[string]ConnectionStats),
	}
}

// Method to set latency and bandwidth between two nodes
func (lbm *LatencyBandwidthMap) SetStats(from, to string, latency, bandwidth int) {
	if _, exists := lbm.Stats[from]; !exists {
		lbm.Stats[from] = make(map[string]ConnectionStats)
	}
	lbm.Stats[from][to] = ConnectionStats{Latency: latency, Bandwidth: bandwidth}
}

// Method to get latency and bandwidth between two nodes
func (lbm *LatencyBandwidthMap) GetStats(from, to string) (ConnectionStats, bool) {
	if toMap, exists := lbm.Stats[from]; exists {
		if stats, ok := toMap[to]; ok {
			return stats, true
		}
	}
	return ConnectionStats{}, false // Return empty stats and false if not found
}

// func main() {
// 	// Example usage
// 	lbm := NewLatencyBandwidthMap()
//
// 	// Set some example stats
// 	lbm.SetStats("vm1", "vm2", 10, 100)  // 10ms latency, 100 Mbps bandwidth
// 	lbm.SetStats("vm2", "vm3", 15, 50)   // 15ms latency, 50 Mbps bandwidth
// 	lbm.SetStats("vm3", "asus", 20, 200) // 20ms latency, 200 Mbps bandwidth
// 	lbm.SetStats("vm1", "vm1", 0, 0)     // Self-connection
//
// 	// Get and print some stats
// 	if stats, ok := lbm.GetStats("vm1", "vm2"); ok {
// 		fmt.Printf("vm1 -> vm2: Latency=%d ms, Bandwidth=%d Mbps\n", stats.Latency, stats.Bandwidth)
// 	}
//
// 	if stats, ok := lbm.GetStats("vm2", "vm3"); ok {
// 		fmt.Printf("vm2 -> vm3: Latency=%d ms, Bandwidth=%d Mbps\n", stats.Latency, stats.Bandwidth)
// 	}
//
// 	if _, ok := lbm.GetStats("vm1", "asus"); !ok {
// 		fmt.Println("No stats recorded from vm1 to asus")
// 	}
// }
