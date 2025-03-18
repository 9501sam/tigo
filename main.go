package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	// appsv1 "k8s.io/api/apps/v1"
	"encoding/json"
	// "fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
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

var nodes = []string{"vm1", "vm2", "vm3", "asus"}
var services = []string{
	"adservice", "cartservice", "checkoutservice", "currencyservice", "emailservice",
	"frontend", "paymentservice", "productcatalogservice", "recommendationservice",
	"redis-cart", "shippingservice",
}

var config *rest.Config
var err error
var clientset *kubernetes.Clientset

func Init() {
	var config *rest.Config
	var err error

	if _, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST"); exists {
		config, err = rest.InClusterConfig()
	} else {
		kubeconfigPath := clientcmd.RecommendedHomeFile
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	if err != nil {
		log.Fatalf("Failed to create Kubernetes config: %v", err)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
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
				for s := 0; s < dpso.NumServices; s++ { // 抓出一個粒子
					r1, r2 := rand.Float64(), rand.Float64() // TODO: r1, r2 為 []
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
			solution[i][j] = rand.Intn(3) + 1
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
	for s, serviceName := range services {
		totalReplicas := 0
		nodeReplicas := map[string]int{}

		for n, nodeName := range nodes {
			replicas := solution[n][s]
			nodeReplicas[nodeName] = replicas
			totalReplicas += replicas
		}

		if totalReplicas == 0 {
			fmt.Printf("Skipping service %s (no replicas needed)\n", serviceName)
			continue
		}

		fmt.Printf("Updating service %s with total %d replicas\n", serviceName, totalReplicas)

		// 獲取現有 Deployment
		deployment, err := clientset.AppsV1().Deployments("online-boutique").Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Failed to get deployment %s: %v", serviceName, err)
			continue
		}

		// 更新 replicas 數量
		replicasInt32 := int32(totalReplicas)
		deployment.Spec.Replicas = &replicasInt32

		// 設定 Node Affinity
		nodeAffinity := &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   getAssignedNodes(nodeReplicas),
							},
						},
					},
				},
			},
		}

		deployment.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: nodeAffinity,
		}

		// 更新 Deployment
		_, err = clientset.AppsV1().Deployments("online-boutique").Update(context.TODO(), deployment, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to update deployment %s: %v", serviceName, err)
			continue
		}
		fmt.Printf("Successfully updated service %s\n", serviceName)
	}
	// return 0.0 // Placeholder for actual evaluation function
	time.Sleep(1 * time.Minute)
	return float64(getResponseTime())
}

func sigmoid(x float64) float64 {
	return 1 / (1 + 1/float64(1+rand.ExpFloat64()))
}

func main() {
	Init()
	dpso := NewDPSO(3, len(nodes), len(services), 60)
	dpso.Optimize()

	// 假設這是來自 DPSO 的最佳解 for test
	// solution := [][]int{
	// 	{2, 1, 0, 1, 0, 2, 1, 0, 0, 1, 0}, // vm1
	// 	{1, 1, 1, 0, 2, 0, 1, 2, 1, 0, 1}, // vm2
	// 	{0, 0, 2, 1, 0, 1, 0, 1, 0, 1, 2}, // vm3
	// 	{0, 1, 0, 2, 1, 0, 1, 0, 1, 2, 0}, // asus
	// }
	// evaluate(solution)
	// getResponseTime()

	// fmt.Println("Best Solution:", dpso.BestSolution, "Score:", dpso.BestScore)
}

// 取得有 Replica 的節點列表
func getAssignedNodes(nodeReplicas map[string]int) []string {
	var nodes []string
	for node, replicas := range nodeReplicas {
		if replicas > 0 {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

/// *** Jaeger Part *** ///
const jaegerBaseURL = "http://localhost:16686/api"

// 定義 JSON 結構
type ServicesResponse struct {
	Data []string `json:"data"`
}

type OperationsResponse struct {
	Data []string `json:"data"`
}

type TraceResponse struct {
	Data []struct {
		Spans []struct {
			SpanID        string `json:"spanID"`
			OperationName string `json:"operationName"`
			Duration      int64  `json:"duration"` // 微秒 (µs)
			References    []struct {
				RefType string `json:"refType"`
				SpanID  string `json:"spanID"`
			} `json:"references"`
		} `json:"spans"`
	} `json:"data"`
}

// 取得所有 service
func getServices() ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/services", jaegerBaseURL))
	if err != nil {
		return nil, fmt.Errorf("error fetching services: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var result ServicesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return result.Data, nil
}

// 取得某個 service 的所有 operations
func getOperations(service string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/services/%s/operations", jaegerBaseURL, service))
	if err != nil {
		return nil, fmt.Errorf("error fetching operations for %s: %v", service, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var result OperationsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return result.Data, nil
}

// 計算某個 API operation 的 self duration
func getOperationSelfDuration(service, operation string) (int64, error) {
	url := fmt.Sprintf("%s/traces?service=%s&operation=%s&limit=10", jaegerBaseURL, service, operation)
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("error fetching traces: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response: %v", err)
	}

	var result TraceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("error parsing JSON: %v", err)
	}

	// 計算 self duration
	var totalSelfDuration int64
	var count int64

	for _, trace := range result.Data {
		spanMap := make(map[string]int64)
		childMap := make(map[string][]string)

		for _, span := range trace.Spans {
			spanMap[span.SpanID] = span.Duration
			for _, ref := range span.References {
				if ref.RefType == "CHILD_OF" {
					childMap[ref.SpanID] = append(childMap[ref.SpanID], span.SpanID)
				}
			}
		}

		for _, span := range trace.Spans {
			if span.OperationName == operation {
				childDuration := int64(0)
				for _, childID := range childMap[span.SpanID] {
					childDuration += spanMap[childID]
				}
				selfDuration := span.Duration - childDuration
				totalSelfDuration += selfDuration
				count++
			}
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("no traces found for %s/%s", service, operation)
	}

	return totalSelfDuration / count, nil
}

func getResponseTime() int64 {
	services, err := getServices()
	if err != nil {
		fmt.Printf("Error getting services: %v\n", err)
		return 0
	}

	score := int64(0)

	for _, service := range services {
		operations, err := getOperations(service)
		if err != nil {
			fmt.Printf("Error getting operations for %s: %v\n", service, err)
			continue
		}

		for _, operation := range operations {
			selfDuration, err := getOperationSelfDuration(service, operation)
			if err != nil {
				fmt.Printf("Error getting self duration for %s/%s: %v\n", service, operation, err)
				continue
			}
			fmt.Printf("Self Duration for %s/%s: %d µs\n", service, operation, selfDuration)
			score = score + selfDuration
		}
	}

	return score
}
