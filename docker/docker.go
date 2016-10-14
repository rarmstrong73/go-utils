package docker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
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

// Image represents information about a docker image
type Image struct {
	RepoTags    []string          `json:"RepoTags"`
	RepoDigests []string          `json:"RepoDigests"`
	ID          string            `json:"Id"`
	Created     int32             `json:"Created"`
	Size        int32             `json:"Size"`
	VirtualSize int32             `json:"VirtualSize"`
	Labels      map[string]string `json:"Labels"`
}

// ListContainers returns the containers on the host
func ListContainers(host string, all bool) (containers []Container) {
	queryStringParams := map[string]string{
		"all": strconv.FormatBool(all),
	}
	return getContainers(fmt.Sprintf("http://%s:%d/containers/json", host, port), queryStringParams)
}

func getContainers(url string, queryStringParams map[string]string) (containers []Container) {
	response := httpGetResponse(url, queryStringParams)
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

// RemoveContainer deletes the given container from the given host
func RemoveContainer(host, nameOrID string, deleteVolumes, force bool) error {
	url := fmt.Sprintf("http://%s:%d/containers/%s", host, port, nameOrID)
	queryStringParams := map[string]string{
		"v":     strconv.FormatBool(deleteVolumes),
		"force": strconv.FormatBool(force),
	}
	response := httpDeleteResponse(url, queryStringParams)
	defer response.Body.Close()

	if response.StatusCode == 400 {
		return fmt.Errorf("One of the supplied paramaters was bad %v", queryStringParams)
	} else if response.StatusCode == 404 {
		return fmt.Errorf("%s didn't exist on %s's filesystem.\n", nameOrID, host)
	} else if response.StatusCode == 409 {
		return fmt.Errorf("There was a conflict trying to remove %s from %s's filesystem.\n", nameOrID, host)
	} else if response.StatusCode == 500 {
		return fmt.Errorf("There was a server error trying to remove %s from %s.\n", nameOrID, host)
	}

	log.Printf("%s successfully removed from %s's filesystem.\n", nameOrID, host)
	return nil
}

// ListImages returns the images on the host
func ListImages(host string, all bool) (images []Image) {
	queryStringParams := map[string]string{
		"all": strconv.FormatBool(all),
	}

	response := httpGetResponse(fmt.Sprintf("http://%s:%d/images/json", host, port), queryStringParams)
	defer response.Body.Close()

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(string(jsonBytes))

	err = json.Unmarshal(jsonBytes, &images)
	if err != nil {
		log.Fatal(err)
	}

	return images
}

// CreateImage creates an image either by pulling it from the registry or by importing it
func CreateImage(host, fromImage, fromSrc, repo, tag string) error {
	url := fmt.Sprintf("http://%s:%d/images/create", host, port)
	queryStringParams := map[string]string{}

	if fromImage != "" {
		queryStringParams["fromImage"] = fromImage
	}

	if fromSrc != "" {
		queryStringParams["fromSrc"] = fromSrc
	}

	if repo != "" {
		queryStringParams["repo"] = repo
	}

	if tag != "" {
		queryStringParams["tag"] = tag
	}

	response := httpPostRequest(url, queryStringParams)
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("Failed to start container")
	}

	return nil
}

// RemoveImage will remove the image from the hosts filesystem
func RemoveImage(host, image string, force, noPrune bool) error {
	url := fmt.Sprintf("http://%s:%d/images/%s", host, port, image)
	queryStringParams := map[string]string{
		"force":   strconv.FormatBool(force),
		"noprune": strconv.FormatBool(noPrune),
	}
	response := httpDeleteResponse(url, queryStringParams)
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return fmt.Errorf("%s didn't exist on %s's filesystem", image, host)
	} else if response.StatusCode == 409 {
		return fmt.Errorf("There was a conflict trying to remove %s from %s's filesystem", image, host)
	} else if response.StatusCode == 500 {
		return fmt.Errorf("There was an error trying to remove %s from %s", image, host)
	}

	log.Printf("%s successfully removed from %s's filesystem", image, host)
	return nil
}

// ============================================================================
// ============================= HTTP UTILS ===================================
// ============================================================================

func httpGetResponse(url string, queryStringParams map[string]string) *http.Response {
	return doHTTPResponse(http.MethodGet, url, queryStringParams)
}

func httpPostRequest(url string, queryStringParams map[string]string) *http.Response {
	return doHTTPResponse(http.MethodPost, url, queryStringParams)
}

func httpDeleteResponse(url string, queryStringParams map[string]string) *http.Response {
	return doHTTPResponse(http.MethodDelete, url, queryStringParams)
}

func doHTTPResponse(method, url string, queryStringParams map[string]string) *http.Response {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
	}

	queryString := request.URL.Query()
	for key, value := range queryStringParams {
		queryString.Add(key, value)
	}
	request.URL.RawQuery = queryString.Encode()

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	return response
}
