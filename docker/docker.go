package docker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var port = 2375

// Bridge represents the bridge information
type Bridge struct {
	IPAMConfig          string `json:"IPAMConfig"`
	Links               string `json:"Links"`
	Aliases             string `json:"Aliases"`
	NetworkID           string `json:"NetworkID"`
	EndpointID          string `json:"EndpointID"`
	Gateway             string `json:"Gateway"`
	IPAddress           string `json:"IPAddress"`
	IPPrefixLen         int    `json:"IPPrefixLen"`
	IPv6Gateway         string `json:"IPv6Gateway"`
	GlobalIPv6Address   string `json:"GlobalIPv6Address"`
	GlobalIPv6PrefixLen int    `json:"GlobalIPv6PrefixLen"`
	MacAddress          string `json:"MacAddress"`
}

// Networks represents the network information
type Networks struct {
	Bridge Bridge `json:"bridge"`
}

// NetworkSettings represents the networks settings of a container
type NetworkSettings struct {
	Networks Networks `json:"Networks"`
}

// HostConfig represents the host config of a container
type HostConfig struct {
	NetworkMode string `json:"NetworkMode"`
}

// PortMap represents an individual port mapping
type PortMap struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

// Container represents a single container that comes back from the containers/json endpoint
type Container struct {
	ID              string            `json:"Id"`
	Names           []string          `json:"Names"`
	Image           string            `json:"Image"`
	ImageID         string            `json:"ImageID"`
	Command         string            `json:"Command"`
	Created         int64             `json:"Create"`
	Status          string            `json:"Status"`
	Ports           []PortMap         `json:"Ports"`
	Labels          map[string]string `json:"Labels"`
	SizeRw          int               `json:"SizeRw"`
	SizeRootFs      int               `json:"SizeRootFs"`
	HostConfig      HostConfig        `json:"HostConfig"`
	NetworkSettings NetworkSettings   `json:"NetworkSettings"`
}

// ListContainersFromHost returns the containers on the host
func ListContainersFromHost(host string) (containers []Container) {
	url := fmt.Sprintf("http://%s:%d/containers/json", host, port)
	response := httpGetResponse(url)
	defer response.Body.Close()

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(jsonBytes, &containers)
	if err != nil {
		log.Fatal(err)
	}

	return containers
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
