package fleet

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

var port = 49153
var apiVersion = "v1"

// Acceptable fleet states
const (
	Launched = "launched"
	Loaded   = "loaded"
	Inactive = "inactive"
)

// Option represents a single option in a fleet unit
type Option struct {
	Name    string `json:"name"`
	Section string `json:"section"`
	Value   string `json:"value"`
}

// Unit represents a fleet unit.
type Unit struct {
	CurrentState string   `json:"currentState"`
	DesiredState string   `json:"desiredState"`
	Name         string   `json:"name"`
	Options      []Option `json:"options"`
}

// UnitState represents a unit state.
type UnitState struct {
	Hash               string `json:"hash"`
	MachineID          string `json:"machineID"`
	Name               string `json:"name"`
	SystemdActiveState string `json:"systemdActiveState"`
	SystemdLoadState   string `json:"systemdLoadState"`
	SystemdSubState    string `json:"systemdSubState"`
}

// Machine represents information about a fleet machine.
type Machine struct {
	ID        string            `json:"id"`
	PrimaryIP string            `json:"primaryIP"`
	Metadata  map[string]string `json:"metadata"`
}

// UnitsResponse represents the response from the units endpoint.
type UnitsResponse struct {
	NextPageToken string `json:"nextPageToken"`
	Units         []Unit `json:"units"`
}

// UnitStateResponse represents the response from the state endpoint.
type UnitStateResponse struct {
	NextPageToken string      `json:"nextPageToken"`
	States        []UnitState `json:"states"`
}

// MachinesResponse represents the response from the machines endpoint
type MachinesResponse struct {
	NextPageToken string    `json:"nextPageToken"`
	Machines      []Machine `json:"machines"`
}

// Error the code and message of the fleet error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse the response structure of a fleet error
type ErrorResponse struct {
	Error Error `json:"error"`
}

// ListUnits returns all fleet units in the host's cluster
func ListUnits(host string) (units []Unit) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units", host, port, apiVersion)
	response := httpGetResponse(url)
	defer response.Body.Close()

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var fleetResponse UnitsResponse
	err = json.Unmarshal(jsonBytes, &fleetResponse)
	if err != nil {
		log.Fatal(err)
	}

	units = append(units, fleetResponse.Units...)
	nextPageToken := fleetResponse.NextPageToken

	for nextPageToken != "" {
		nextPageURL := fmt.Sprintf("%s?nextPageToken=%s", url, nextPageToken)
		resp := httpGetResponse(nextPageURL)
		defer resp.Body.Close()

		jsonContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var nextPageFleetResponse UnitsResponse
		err = json.Unmarshal(jsonContent, &nextPageFleetResponse)

		units = append(units, nextPageFleetResponse.Units...)
		nextPageToken = nextPageFleetResponse.NextPageToken
	}

	return units
}

// ListUnitsByName returns the template and any known units with the given name
func ListUnitsByName(host, name string) (template Unit, units []Unit) {
	allUnits := ListUnits(host)
	for _, unit := range allUnits {
		if strings.HasPrefix(unit.Name, fmt.Sprintf("%s@", name)) {
			if strings.Contains(unit.Name, "@.") {
				template = unit
			} else {
				units = append(units, unit)
			}
		}
	}
	return template, units
}

// CreateUnit creates a unit with the given name, desired state, and options
func CreateUnit(host, name, desiredState string, options []Option) error {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, name)
	body := map[string]interface{}{
		"desiredState": desiredState,
		"options":      options,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Fatal(err)
	}

	response := httpPutResponse(url, bodyBytes)
	defer response.Body.Close()

	if response.StatusCode == 400 {
		return handleError(response.Body)
	}

	if response.StatusCode == 409 {
		return handleError(response.Body)
	}

	if response.StatusCode != 201 {
		return handleError(response.Body)
	}

	return nil
}

// ModifyUnitDesiredState modifies the desired state of the given unit
func ModifyUnitDesiredState(host, name, desiredState string) error {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, name)

	body := map[string]string{
		"desiredState": desiredState,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Fatal(err)
	}

	response := httpPutResponse(url, bodyBytes)
	defer response.Body.Close()

	if response.StatusCode == 400 {
		return handleError(response.Body)
	}

	if response.StatusCode != 204 {
		return handleError(response.Body)
	}

	return nil
}

// DestroyUnit destroys the unit
func DestroyUnit(host, name string) error {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, name)
	response := httpDeleteResponse(url)
	defer response.Body.Close()
	if response.StatusCode != 204 {
		return handleError(response.Body)
	}
	return nil
}

// ListUnitStates returns all unit states in the host's cluster
func ListUnitStates(host string) (unitStates []UnitState) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/state", host, port, apiVersion)
	response := httpGetResponse(url)
	defer response.Body.Close()

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var fleetStateResponse UnitStateResponse
	err = json.Unmarshal(jsonBytes, &fleetStateResponse)
	if err != nil {
		log.Fatal(err)
	}

	unitStates = append(unitStates, fleetStateResponse.States...)
	nextPageToken := fleetStateResponse.NextPageToken

	for nextPageToken != "" {
		nextPageURL := fmt.Sprintf("%s?nextPageToken=%s", url, nextPageToken)
		resp := httpGetResponse(nextPageURL)
		defer resp.Body.Close()

		jsonContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var nextPageFleetStateResponse UnitStateResponse
		err = json.Unmarshal(jsonContent, &nextPageFleetStateResponse)
		if err != nil {
			log.Fatal(err)
		}

		unitStates = append(unitStates, nextPageFleetStateResponse.States...)
		nextPageToken = nextPageFleetStateResponse.NextPageToken
	}

	return unitStates
}

// ListUnitStatesByName returns a list of unit states with the given name
func ListUnitStatesByName(host, name string) (unitStates []UnitState) {
	allUnitStates := ListUnitStates(host)
	for _, unitState := range allUnitStates {
		if strings.HasPrefix(unitState.Name, fmt.Sprintf("%s@", name)) {
			unitStates = append(unitStates, unitState)
		}
	}
	return unitStates
}

// GetUnitStatesByMachineID returns the unit states with the given machineID
func GetUnitStatesByMachineID(host, machineID string) (unitStates []UnitState) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/state?machineID=%s", host, port, apiVersion, machineID)
	response := httpGetResponse(url)
	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var unitStateResponse UnitStateResponse
	err = json.Unmarshal(responseBytes, &unitStateResponse)
	if err != nil {
		log.Fatal(err)
	}

	return unitStateResponse.States
}

// GetUnitStatesByUnitName returns the unit states with the given unit name
func GetUnitStatesByUnitName(host, unitName string) (unitStates []UnitState) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/state?unitName=%s", host, port, apiVersion, unitName)
	response := httpGetResponse(url)
	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var unitStateResponse UnitStateResponse
	err = json.Unmarshal(responseBytes, &unitStateResponse)
	if err != nil {
		log.Fatal(err)
	}

	return unitStateResponse.States
}

// ListMachines returns all machines in the host's cluster
func ListMachines(host string) (machines []Machine) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/machines", host, port, apiVersion)
	response := httpGetResponse(url)
	defer response.Body.Close()

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var fleetMachinesResponse MachinesResponse
	err = json.Unmarshal(jsonBytes, &fleetMachinesResponse)
	if err != nil {
		log.Fatal(err)
	}

	machines = append(machines, fleetMachinesResponse.Machines...)
	nextPageToken := fleetMachinesResponse.NextPageToken

	for nextPageToken != "" {
		nextPageURL := fmt.Sprintf("%s?nextPageToken=%s", url, nextPageToken)
		resp := httpGetResponse(nextPageURL)
		defer resp.Body.Close()

		jsonContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var nextPageFleetMachinesResponse MachinesResponse
		err = json.Unmarshal(jsonContent, &nextPageFleetMachinesResponse)
		if err != nil {
			log.Fatal(err)
		}

		machines = append(machines, nextPageFleetMachinesResponse.Machines...)
		nextPageToken = nextPageFleetMachinesResponse.NextPageToken
	}
	return machines
}

// GetStateOfFleet returns all units, states, and machines in the host's cluster
func GetStateOfFleet(host string) (units []Unit, unitStates []UnitState, machines []Machine) {
	return ListUnits(host), ListUnitStates(host), ListMachines(host)
}

// GetUnit returns the single requested unit
func GetUnit(host, name string) (unit Unit, err error) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, name)
	response := httpGetResponse(url)
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return Unit{}, handleError(response.Body)
	}

	if response.StatusCode != 200 {
		return Unit{}, handleError(response.Body)
	}

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(jsonBytes, &unit)
	return unit, nil
}

func handleError(body io.ReadCloser) error {
	errorBytes, err := ioutil.ReadAll(body)
	if err != nil {
		log.Fatal(err)
	}

	var errorResponse ErrorResponse
	err = json.Unmarshal(errorBytes, &errorResponse)
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Errorf("%d: %s", errorResponse.Error.Code, errorResponse.Error.Message)
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

	request.Header.Add("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	return response
}

func httpDeleteResponse(url string) *http.Response {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return response
}
