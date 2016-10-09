package etcd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var port = 2379
var apiVersion = "v2"

// Node represents an etcd node
type Node struct {
	Dir           bool   `json:"dir"`
	Nodes         []Node `json:"nodes"`
	Key           string `json:"key"`
	Value         string `json:"value"`
	ModifiedIndex int64  `json:"modifiedIndex"`
	CreatedIndex  int64  `json:"createdIndex"`
}

// Response is the response from a get request to etcd
type Response struct {
	Action string `json:"action"`
	Node   Node   `json:"node"`
}

// SetResponse is the response object returned by running a set
type SetResponse struct {
	Action   string `json:"action"`
	Node     Node   `json:"node"`
	PrevNode Node   `json:"prevNode"`
}

// Error represents an error in the request
type Error struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
	Cause     string `json:"cause"`
	Index     int64  `json:"index"`
}

// GetKey returns the node at the given path
func GetKey(host, path string) (Node, error) {
	url := fmt.Sprintf("http://%s:%d/%s/keys/%s", host, port, apiVersion, path)
	response := httpGetResponse(url)
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return Node{}, handleError(response.Body)
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var nodeResponse Response
	err = json.Unmarshal(responseBytes, &nodeResponse)
	if err != nil {
		log.Fatal(err)
	}

	return nodeResponse.Node, nil
}

// SetKey sets or updates the value at the given path
func SetKey(host, path, value string) (prevNode Node, err error) {
	url := fmt.Sprintf("http://%s:%d/%s/keys/%s", host, port, apiVersion, path)
	body := fmt.Sprintf("value=%s", value)

	response := httpPutResponse(url, []byte(body))
	defer response.Body.Close()

	if response.StatusCode != 200 && response.StatusCode != 201 {
		return Node{}, handleError(response.Body)
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var setResponse SetResponse
	err = json.Unmarshal(responseBytes, &setResponse)
	if err != nil {
		log.Fatal(err)
	}

	return setResponse.PrevNode, nil
}

// DeleteKey deletes the key at the given path
func DeleteKey(host, path string) error {
	url := fmt.Sprintf("http://%s:%d/%s/keys/%s", host, port, apiVersion, path)
	response := httpDeleteResponse(url)
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return handleError(response.Body)
	}

	return nil
}

// RecurseKeys returns a recursive listing of the keys at the given path
func RecurseKeys(host, path string) (Node, error) {
	return GetKey(host, fmt.Sprintf("%s?recursive=true", path))
}

func handleError(body io.ReadCloser) error {
	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		log.Fatal(err)
	}

	var errorResponse Error
	err = json.Unmarshal(bytes, &errorResponse)
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Errorf("%d: %s (%s)", errorResponse.ErrorCode, errorResponse.Message, errorResponse.Cause)
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

func httpPutResponse(url string, body []byte) *http.Response {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return response
}

func httpDeleteResponse(url string) *http.Response {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return response
}
