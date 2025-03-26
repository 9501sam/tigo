package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	// appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var nodes = []string{"vm1", "vm2", "vm3", "asus"}

const namespace = "online-boutique"

var targetDeployments = map[string]bool{
	"cartservice":           true,
	"checkoutservice":       true,
	"currencyservice":       true,
	"emailservice":          true,
	"frontend":              true,
	"paymentservice":        true,
	"productcatalogservice": true,
	"recommendationservice": true,
	"redis-cart":            true,
	"shippingservice":       true,
}

func renameDeployments() {
	// 嘗試讀取 kubeconfig
	kubeconfigPath := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to load kubeconfig: %v", err)
	}

	// 建立 Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	deploymentsClient := clientset.AppsV1().Deployments(namespace)
	deployList, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list deployments: %v", err)
	}

	for _, deploy := range deployList.Items {
		originalName := deploy.Name
		if !targetDeployments[originalName] {
			continue // 忽略不在指定清單內的 Deployment
		}
		for _, node := range nodes {
			newDeploy := deploy.DeepCopy()
			newDeploy.Name = fmt.Sprintf("%s-%s", originalName, node)
			newDeploy.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}
			newDeploy.ResourceVersion = ""
			newDeploy.UID = ""

			_, err := deploymentsClient.Create(context.TODO(), newDeploy, metav1.CreateOptions{})
			if err != nil {
				log.Printf("Failed to create deployment %s: %v", newDeploy.Name, err)
			} else {
				log.Printf("Created deployment %s", newDeploy.Name)
			}
		}

		// 刪除原本的 deployment
		if err := deploymentsClient.Delete(context.TODO(), originalName, metav1.DeleteOptions{}); err != nil {
			log.Printf("Failed to delete deployment %s: %v", originalName, err)
		} else {
			log.Printf("Deleted original deployment %s", originalName)
		}
	}
}

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
			"emailservice": 3,
			"frontend": 0,
			"paymentservice": 0,
			"productcatalogservice": 0,
			"recommendationservice": 0,
			"redis-cart": 0,
			"shippingservice": 1
		}
	}`

func main() {
	renameDeployments()

	// 嘗試讀取 kubeconfig
	kubeconfigPath := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to load kubeconfig: %v", err)
	}

	// 建立 Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	var deploymentConfig map[string]map[string]int
	err = json.Unmarshal([]byte(jsonStr), &deploymentConfig)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	err = UpdateDeployments(clientset, deploymentConfig)
	if err != nil {
		log.Fatalf("Error updating deployments: %v", err)
	}

	fmt.Println("Successfully updated all deployments")
}

func UpdateDeployments(clientset *kubernetes.Clientset, deploymentConfig map[string]map[string]int) error {
	deploymentsClient := clientset.AppsV1().Deployments(namespace)

	for node, services := range deploymentConfig {
		for service, replicas := range services {
			deployName := fmt.Sprintf("%s-%s", service, node)
			patch := fmt.Sprintf(`{"spec": {"replicas": %d}}`, replicas)

			_, err := deploymentsClient.Patch(context.TODO(), deployName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})

			if err != nil {
				log.Printf("Failed to update deployment %s: %v", deployName, err)
			} else {
				log.Printf("Updated deployment %s to %d replicas", deployName, replicas)
			}
		}
	}
	return nil
}
