package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	// appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	jsonStr := `{
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
			"recommendationservice": 3,
			"redis-cart": 0,
			"shippingservice": 1
		}
	}`

	var config map[string]map[string]int
	err := json.Unmarshal([]byte(jsonStr), &config)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	err = updateDeployments(config)
	if err != nil {
		log.Fatalf("Error updating deployments: %v", err)
	}

	fmt.Println("Successfully updated all deployments")
}

func updateDeployments(config map[string]map[string]int) error {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	configRest, err := kubeconfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(configRest)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %v", err)
	}

	ctx := context.Background()
	namespace := "online-boutique"

	type deploymentInfo struct {
		node     string
		replicas int32
	}
	serviceDeployments := make(map[string][]deploymentInfo)

	// Collect deployment requirements
	for node, services := range config {
		for service, replicas := range services {
			if replicas > 0 {
				serviceDeployments[service] = append(serviceDeployments[service], deploymentInfo{
					node:     node,
					replicas: int32(replicas),
				})
			}
		}
	}

	for service, deployments := range serviceDeployments {
		// Get original deployment as template
		original, err := clientset.AppsV1().Deployments(namespace).Get(ctx, service, metav1.GetOptions{})
		if err != nil {
			log.Printf("Warning: Original deployment %s not found: %v", service, err)
			continue
		}

		// Handle each node-specific deployment
		for _, depInfo := range deployments {
			deploymentName := service
			if len(deployments) > 1 {
				deploymentName = fmt.Sprintf("%s-%s", service, depInfo.node)
			}

			// Check if deployment exists
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
			if err != nil && strings.Contains(err.Error(), "not found") {
				// Create new deployment
				deployment = original.DeepCopy()
				// Clear metadata fields that shouldn't be set on creation
				deployment.ObjectMeta = metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: namespace,
				}
				deployment.Spec.Replicas = &depInfo.replicas
				if len(deployments) > 1 {
					// deployment.Spec.Selector = &metav1.LabelSelector{
					// 	MatchLabels: map[string]string{
					// 		"app": deploymentName,
					// 	},
					// }
					// deployment.Spec.Template.Labels = map[string]string{
					// 	"app": deploymentName,
					// }
				}
				if deployment.Spec.Template.Spec.NodeSelector == nil {
					deployment.Spec.Template.Spec.NodeSelector = make(map[string]string)
				}
				deployment.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = depInfo.node

				_, err = clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
				if err != nil {
					log.Printf("Failed to create deployment %s: %v", deploymentName, err)
					continue
				}
			} else if err == nil {
				// Update existing deployment
				deployment.Spec.Replicas = &depInfo.replicas
				if len(deployments) > 1 {
					deployment.Spec.Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": deploymentName,
						},
					}
					deployment.Spec.Template.Labels = map[string]string{
						"app": deploymentName,
					}
				}
				if deployment.Spec.Template.Spec.NodeSelector == nil {
					deployment.Spec.Template.Spec.NodeSelector = make(map[string]string)
				}
				deployment.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = depInfo.node

				_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
				if err != nil {
					log.Printf("Failed to update deployment %s: %v", deploymentName, err)
					continue
				}
			} else {
				log.Printf("Error checking deployment %s: %v", deploymentName, err)
				continue
			}
			fmt.Printf("Updated %s with %d replicas on %s\n", deploymentName, depInfo.replicas, depInfo.node)
		}

		// Clean up original deployment only if we created multiple node-specific ones
		if len(deployments) > 1 {
			err = clientset.AppsV1().Deployments(namespace).Delete(ctx, service, metav1.DeleteOptions{})
			if err != nil && !strings.Contains(err.Error(), "not found") {
				log.Printf("Failed to delete original deployment %s: %v", service, err)
			}
		}
	}

	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-o", "wide")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to verify deployments: %v", err)
	} else {
		fmt.Printf("\nCurrent pods:\n%s\n", string(output))
	}

	return nil
}
