package main

import (
	"context"
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

var renamed = false

func renameDeployments() {
	fmt.Println("enter renameDeployments()")
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

func UpDateDeploymentsByJSON(filename string) {
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
	if err := loadJSONFile(filename, &deploymentConfig); err != nil {
		fmt.Println("Error loading path_durations.json:", err)
		return
	}

	err = UpdateDeployments(clientset, deploymentConfig)
	if err != nil {
		log.Fatalf("Error updating deployments: %v", err)
	}

	fmt.Println("Successfully updated all deployments")
}

func UpdateDeployments(clientset *kubernetes.Clientset, deploymentConfig map[string]map[string]int) error {
	if !renamed {
		renameDeployments()
		renamed = true
	}

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
