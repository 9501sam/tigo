package utils

import (
	"fmt"
	"testing"
)

func TestJaeger(t *testing.T) {
	// 你可以列出所有 service，然後遍歷
	services := []string{"productcatalogservice", "frontend", "checkoutservice",
		"recommendationservice", "emailservice", "paymentservice", "currencyservice", "jaeger-all-in-one"}

	for _, service := range services {
		operations, err := getOperations(service)
		if err != nil {
			fmt.Printf("Error getting operations for %s: %v\n", service, err)
			continue
		}

		fmt.Printf("Operations for %s:\n", service)
		for _, op := range operations {
			fmt.Println(" -", op)
		}
	}
}
