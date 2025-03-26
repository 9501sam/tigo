package main

import (
	"context"
	"fmt"
	"log"
	"os"

	// appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func main() {
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
