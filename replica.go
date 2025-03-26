package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var jsonStr = `{
		"vm1": {
			"cartservice": 1,
			"checkoutservice": 1,
			"currencyservice": 1,
			"emailservice": 0,
			"frontend": 0,
			"paymentservice": 0,
			"productcatalogservice": 0,
			"recommendationservice": 0,
			"redis-cart": 0,
			"shippingservice": 0
		},
		"vm2": {
			"cartservice": 0,
			"checkoutservice": 0,
			"currencyservice": 0,
			"emailservice": 1,
			"frontend": 1,
			"paymentservice": 1,
			"productcatalogservice": 0,
			"recommendationservice": 0,
			"redis-cart": 0,
			"shippingservice": 0
		},
		"vm3": {
			"cartservice": 0,
			"checkoutservice": 0,
			"currencyservice": 0,
			"emailservice": 0,
			"frontend": 0,
			"paymentservice": 0,
			"productcatalogservice": 1,
			"recommendationservice": 1,
			"redis-cart": 1,
			"shippingservice": 1
		},
		"asus": {
			"cartservice": 0,
			"checkoutservice": 0,
			"currencyservice": 0,
			"emailservice": 0,
			"frontend": 0,
			"paymentservice": 0,
			"productcatalogservice": 0,
			"recommendationservice": 0,
			"redis-cart": 0,
			"shippingservice": 0
		}
	}`

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
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	var deploymentConfig map[string]map[string]int
	if err := json.Unmarshal([]byte(jsonStr), &deploymentConfig); err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	if err := updateDeployments(deploymentConfig, clientset); err != nil {
		log.Fatalf("Error updating deployments: %v", err)
	}

	fmt.Println("Successfully updated all deployments")
}

func getAllDeployments(config map[string]map[string]int) map[string]bool {
	deployments := make(map[string]bool)
	for _, services := range config {
		for deployment := range services {
			deployments[deployment] = true
		}
	}
	return deployments
}

func calculateDeploymentConfig(deployment string, config map[string]map[string]int) (int, *corev1.Affinity, []corev1.TopologySpreadConstraint) {
	totalReplicas := 0
	var nodeSelectorTerms []corev1.NodeSelectorTerm

	for node, services := range config {
		if replicas, exists := services[deployment]; exists && replicas > 0 {
			totalReplicas += replicas
			nodeSelectorTerms = append(nodeSelectorTerms, corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "deployment-node",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{node},
					},
				},
			})
		}
	}

	nodeAffinity := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: nodeSelectorTerms,
			},
		},
	}

	return totalReplicas, nodeAffinity, nil
}

func updateDeployments(config map[string]map[string]int, clientset *kubernetes.Clientset) error {
	for deployment := range getAllDeployments(config) {
		totalReplicas, nodeAffinity, topologySpreadConstraints := calculateDeploymentConfig(deployment, config)
		if err := applyDeployment(deployment, totalReplicas, nodeAffinity, topologySpreadConstraints, clientset); err != nil {
			return fmt.Errorf("failed to apply deployment for %s: %w", deployment, err)
		}
	}
	return nil
}

func applyDeployment(deployment string, replicas int, nodeAffinity *corev1.Affinity, topologySpreadConstraints []corev1.TopologySpreadConstraint, clientset *kubernetes.Clientset) error {
	// 先嘗試取得現有的 Deployment，看看是否有指定 image
	deploymentsClient := clientset.AppsV1().Deployments("online-boutique")
	existingDeployment, err := deploymentsClient.Get(context.TODO(), deployment, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing deployment: %w", err)
	}

	// 取得容器映像檔
	image := existingDeployment.Spec.Template.Spec.Containers[0].Image

	// 建立新的 Deployment 配置
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment,
			Namespace: "online-boutique",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deployment},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": deployment},
				},
				Spec: corev1.PodSpec{
					Affinity:                  nodeAffinity,
					TopologySpreadConstraints: topologySpreadConstraints,
					Containers: []corev1.Container{
						{
							Name:  deployment,
							Image: image, // 使用現有的 image
						},
					},
				},
			},
		},
	}

	// 更新或創建 Deployment
	if existingDeployment != nil {
		dep.ResourceVersion = existingDeployment.ResourceVersion // 保留資源版本以便更新
		_, err = deploymentsClient.Update(context.TODO(), dep, metav1.UpdateOptions{})
	} else {
		_, err = deploymentsClient.Create(context.TODO(), dep, metav1.CreateOptions{})
	}
	return err
}

func int32Ptr(i int32) *int32 { return &i }
