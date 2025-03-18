package main

import (
	"context"
	"fmt"
	"log"
	"os"

	// appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var nodes = []string{"vm1", "vm2", "vm3", "asus"}
var services = []string{
	"adservice", "cartservice", "checkoutservice", "currencyservice", "emailservice",
	"frontend", "paymentservice", "productcatalogservice", "recommendationservice",
	"redis-cart", "shippingservice",
}

var config *rest.Config
var err error

func main() {

	// 判斷是否在 Kubernetes 內部運行
	if _, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST"); exists {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Failed to create in-cluster config: %v", err)
		}
	} else {
		kubeconfigPath := clientcmd.RecommendedHomeFile
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Fatalf("Failed to load kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// 假設這是來自 DPSO 的最佳解
	solution := [][]int{
		{1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0}, // vm1
		{0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0}, // vm2
		{0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1}, // vm3
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // asus
	}

	// 逐一更新 Kubernetes 部署
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
