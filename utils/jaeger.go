package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const jaegerBaseURL = "http://localhost:16686/api/services"

type OperationsResponse struct {
	Data []string `json:"data"`
}

func getOperations(serviceName string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/operations", jaegerBaseURL, serviceName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching operations: %v", err)
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
