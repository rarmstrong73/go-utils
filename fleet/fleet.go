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
func ListUnits(host string) (units []Unit, err error) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units", host, port, apiVersion)
	response, err := httpGetResponse(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var fleetResponse UnitsResponse
	err = json.Unmarshal(jsonBytes, &fleetResponse)
	if err != nil {
		return nil, err
	}

	units = append(units, fleetResponse.Units...)
	nextPageToken := fleetResponse.NextPageToken

	for nextPageToken != "" {
		nextPageURL := fmt.Sprintf("%s?nextPageToken=%s", url, nextPageToken)
		resp, err := httpGetResponse(nextPageURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		jsonContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var nextPageFleetResponse UnitsResponse
		err = json.Unmarshal(jsonContent, &nextPageFleetResponse)

		units = append(units, nextPageFleetResponse.Units...)
		nextPageToken = nextPageFleetResponse.NextPageToken
	}

	return units, err
}

// ListUnitsByName returns the template and any known units with the given name
func ListUnitsByName(host, name string) (template Unit, units []Unit, err error) {
	allUnits, err := ListUnits(host)
	if err != nil {
		return Unit{}, nil, err
	}
	for _, unit := range allUnits {
		if strings.HasPrefix(unit.Name, fmt.Sprintf("%s@", name)) {
			if strings.Contains(unit.Name, "@.") {
				template = unit
			} else {
				units = append(units, unit)
			}
		}
	}
	return template, units, err
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

	response, err := httpPutResponse(url, bodyBytes)
	if err != nil {
		return err
	}
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

// ModifyDesiredState modifies the desired state of the given unit
func (unit Unit) ModifyDesiredState(host, desiredState string) error {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, unit.Name)

	body := map[string]string{
		"desiredState": desiredState,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Fatal(err)
	}

	response, err := httpPutResponse(url, bodyBytes)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == 400 {
		return handleError(response.Body)
	}

	if response.StatusCode != 204 {
		return handleError(response.Body)
	}

	return nil
}

// ModifyDesiredState modifies the desired state of the given unit
func (unitState UnitState) ModifyDesiredState(host, desiredState string) error {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, unitState.Name)

	body := map[string]string{
		"desiredState": desiredState,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Fatal(err)
	}

	response, err := httpPutResponse(url, bodyBytes)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == 400 {
		return handleError(response.Body)
	}

	if response.StatusCode != 204 {
		return handleError(response.Body)
	}

	return nil
}

// Destroy destroys the unit
func (unit Unit) Destroy(host string) error {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, unit.Name)
	response, err := httpDeleteResponse(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 204 {
		return handleError(response.Body)
	}
	return nil
}

// Destroy destroys the unit
func (unitState UnitState) Destroy(host string) error {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, unitState.Name)
	response, err := httpDeleteResponse(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 204 {
		return handleError(response.Body)
	}
	return nil
}

// ListUnitStates returns all unit states in the host's cluster
func ListUnitStates(host string) (unitStates []UnitState, err error) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/state", host, port, apiVersion)
	response, err := httpGetResponse(url)
	if err != nil {
		return nil, err
	}
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
		resp, err := httpGetResponse(nextPageURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		jsonContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var nextPageFleetStateResponse UnitStateResponse
		err = json.Unmarshal(jsonContent, &nextPageFleetStateResponse)
		if err != nil {
			return nil, err
		}

		unitStates = append(unitStates, nextPageFleetStateResponse.States...)
		nextPageToken = nextPageFleetStateResponse.NextPageToken
	}

	return unitStates, err
}

// ListUnitStatesByName returns a list of unit states with the given name
func ListUnitStatesByName(host, name string) (unitStates []UnitState, err error) {
	allUnitStates, err := ListUnitStates(host)
	if err != nil {
		return nil, err
	}
	for _, unitState := range allUnitStates {
		if strings.HasPrefix(unitState.Name, fmt.Sprintf("%s@", name)) {
			unitStates = append(unitStates, unitState)
		}
	}
	return unitStates, err
}

// GetUnitStatesByMachineID returns the unit states with the given machineID
func GetUnitStatesByMachineID(host, machineID string) (unitStates []UnitState, err error) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/state?machineID=%s", host, port, apiVersion, machineID)
	response, err := httpGetResponse(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var unitStateResponse UnitStateResponse
	err = json.Unmarshal(responseBytes, &unitStateResponse)
	if err != nil {
		return nil, err
	}

	return unitStateResponse.States, err
}

// GetUnitStatesByUnitName returns the unit states with the given unit name
func GetUnitStatesByUnitName(host, unitName string) (unitStates []UnitState, err error) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/state?unitName=%s", host, port, apiVersion, unitName)
	response, err := httpGetResponse(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var unitStateResponse UnitStateResponse
	err = json.Unmarshal(responseBytes, &unitStateResponse)
	if err != nil {
		return nil, err
	}

	return unitStateResponse.States, err
}

// ListMachines returns all machines in the host's cluster
func ListMachines(host string) (machines []Machine, err error) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/machines", host, port, apiVersion)
	response, err := httpGetResponse(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var fleetMachinesResponse MachinesResponse
	err = json.Unmarshal(jsonBytes, &fleetMachinesResponse)
	if err != nil {
		return nil, err
	}

	machines = append(machines, fleetMachinesResponse.Machines...)
	nextPageToken := fleetMachinesResponse.NextPageToken

	for nextPageToken != "" {
		nextPageURL := fmt.Sprintf("%s?nextPageToken=%s", url, nextPageToken)
		resp, err := httpGetResponse(nextPageURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		jsonContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var nextPageFleetMachinesResponse MachinesResponse
		err = json.Unmarshal(jsonContent, &nextPageFleetMachinesResponse)
		if err != nil {
			return nil, err
		}

		machines = append(machines, nextPageFleetMachinesResponse.Machines...)
		nextPageToken = nextPageFleetMachinesResponse.NextPageToken
	}
	return machines, err
}

// GetStateOfFleet returns all units, states, and machines in the host's cluster
func GetStateOfFleet(host string) (units []Unit, unitStates []UnitState, machines []Machine, err error) {
	units, err = ListUnits(host)
	if err != nil {
		return nil, nil, nil, err
	}
	unitStates, err = ListUnitStates(host)
	if err != nil {
		return nil, nil, nil, err
	}
	machines, err = ListMachines(host)
	if err != nil {
		return nil, nil, nil, err
	}
	return units, unitStates, machines, err
}

// GetUnit returns the single requested unit
func GetUnit(host, name string) (unit Unit, err error) {
	url := fmt.Sprintf("http://%s:%d/fleet/%s/units/%s", host, port, apiVersion, name)
	response, err := httpGetResponse(url)
	if err != nil {
		return Unit{}, err
	}
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return Unit{}, handleError(response.Body)
	}

	if response.StatusCode != 200 {
		return Unit{}, handleError(response.Body)
	}

	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return Unit{}, err
	}
	err = json.Unmarshal(jsonBytes, &unit)
	if err != nil {
		return Unit{}, err
	}
	return unit, err
}

func handleError(body io.ReadCloser) error {
	errorBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	var errorResponse ErrorResponse
	err = json.Unmarshal(errorBytes, &errorResponse)
	if err != nil {
		return err
	}

	return fmt.Errorf("%d: %s", errorResponse.Error.Code, errorResponse.Error.Message)
}

// ============================================================================
// ============================= HTTP UTILS ===================================
// ============================================================================

func httpGetResponse(url string) (*http.Response, error) {
	response, err := http.Get(url)
	return response, err
}

func httpPutResponse(url string, body []byte) (*http.Response, error) {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))

	request.Header.Add("Content-Type", "application/json")

	response, err := client.Do(request)
	return response, err
}

func httpDeleteResponse(url string) (*http.Response, error) {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.Do(request)
	return response, err
}
