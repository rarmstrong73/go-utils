package consul

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var port = 8500
var apiVersion = "v1"

// HealthNode represents the health information about a node in consul
type HealthNode struct {
	Node        string `json:"Node"`
	CheckID     string `json:"CheckID"`
	Name        string `json:"Name"`
	Status      string `json:"Status"`
	Notes       string `json:"Notes"`
	Output      string `json:"Output"`
	ServiceID   string `json:"ServiceID"`
	ServiceName string `json:"ServiceName"`
	CreateIndex int64  `json:"CreateIndex"`
	ModifyIndex int64  `json:"ModifyIndex"`
}

// GetHealthChecks returns the checks of a service
func GetHealthChecks(host, service string) (nodes []HealthNode, err error) {
	url := fmt.Sprintf("http://%s:%d/%s/health/checks/%s", host, port, apiVersion, service)
	response := httpGetResponse(url)
	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(responseBytes, &nodes)
	if err != nil {
		log.Fatal(err)
	}

	if len(nodes) == 0 {
		return []HealthNode{}, fmt.Errorf("Consul returned 0 checks.")
	}

	return nodes, nil
}

// ============================================================================
// ============================= HTTP UTILS ===================================
// ============================================================================

func httpGetResponse(url string) *http.Response {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	return response
}
