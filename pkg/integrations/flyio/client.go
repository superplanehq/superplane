package flyio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.machines.dev"

type Client struct {
	Token   string
	http    core.HTTPContext
	BaseURL string
}

type FlyIOError struct {
	Message string `json:"error"`
}

type FlyIOAPIError struct {
	StatusCode int
	Message    string
	Body       []byte
}

func (e *FlyIOAPIError) Error() string {
	return fmt.Sprintf("Fly.io API error (status %d): %s", e.StatusCode, e.Message)
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error finding API token: %v", err)
	}

	return &Client{
		Token:   string(apiToken),
		http:    http,
		BaseURL: baseURL,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	statusCode, responseBody, err := c.execRequestRaw(method, url, body)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, newFlyIOAPIError(statusCode, responseBody)
	}

	return responseBody, nil
}

func (c *Client) execRequestRaw(method, url string, body io.Reader) (int, []byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	res, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("error reading body: %v", err)
	}

	return res.StatusCode, responseBody, nil
}

func newFlyIOAPIError(statusCode int, responseBody []byte) *FlyIOAPIError {
	apiError := &FlyIOAPIError{
		StatusCode: statusCode,
		Body:       responseBody,
	}

	var payload FlyIOError
	if err := json.Unmarshal(responseBody, &payload); err == nil && payload.Message != "" {
		apiError.Message = payload.Message
	} else {
		apiError.Message = string(responseBody)
	}

	return apiError
}

// App represents a Fly.io application
type App struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Status       string       `json:"status,omitempty"`
	MachineCount int          `json:"machine_count,omitempty"`
	VolumeCount  int          `json:"volume_count,omitempty"`
	Network      string       `json:"network,omitempty"`
	Organization *AppOrg      `json:"organization,omitempty"`
}

type AppOrg struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Machine represents a Fly.io Machine
type Machine struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	State      string         `json:"state"`
	Region     string         `json:"region"`
	InstanceID string         `json:"instance_id"`
	PrivateIP  string         `json:"private_ip"`
	Config     *MachineConfig `json:"config,omitempty"`
	ImageRef   *ImageRef      `json:"image_ref,omitempty"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

type MachineConfig struct {
	Image       string            `json:"image"`
	Env         map[string]string `json:"env,omitempty"`
	AutoDestroy bool              `json:"auto_destroy,omitempty"`
	Guest       *GuestConfig      `json:"guest,omitempty"`
	Init        *InitConfig       `json:"init,omitempty"`
	Restart     *RestartPolicy    `json:"restart,omitempty"`
	Services    []Service         `json:"services,omitempty"`
}

type GuestConfig struct {
	CPUKind  string `json:"cpu_kind,omitempty"`
	CPUs     int    `json:"cpus,omitempty"`
	MemoryMB int    `json:"memory_mb,omitempty"`
}

type InitConfig struct {
	Exec       []string `json:"exec,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty"`
	Cmd        []string `json:"cmd,omitempty"`
	TTY        bool     `json:"tty,omitempty"`
}

type RestartPolicy struct {
	Policy string `json:"policy,omitempty"`
}

type Service struct {
	Protocol     string `json:"protocol"`
	InternalPort int    `json:"internal_port"`
	Ports        []Port `json:"ports,omitempty"`
}

type Port struct {
	Port     int      `json:"port"`
	Handlers []string `json:"handlers,omitempty"`
}

type ImageRef struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest,omitempty"`
}

// ListApps retrieves all apps for an organization
func (c *Client) ListApps(orgSlug string) ([]App, error) {
	url := fmt.Sprintf("%s/v1/apps?org_slug=%s", c.BaseURL, orgSlug)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		TotalApps int   `json:"total_apps"`
		Apps      []App `json:"apps"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Apps, nil
}

// GetApp retrieves details about a specific app
func (c *Client) GetApp(appName string) (*App, error) {
	url := fmt.Sprintf("%s/v1/apps/%s", c.BaseURL, appName)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var app App
	err = json.Unmarshal(responseBody, &app)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &app, nil
}

// ListMachines retrieves all machines for an app
func (c *Client) ListMachines(appName string) ([]Machine, error) {
	url := fmt.Sprintf("%s/v1/apps/%s/machines", c.BaseURL, appName)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var machines []Machine
	err = json.Unmarshal(responseBody, &machines)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return machines, nil
}

// GetMachine retrieves a specific machine
func (c *Client) GetMachine(appName, machineID string) (*Machine, error) {
	url := fmt.Sprintf("%s/v1/apps/%s/machines/%s", c.BaseURL, appName, machineID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var machine Machine
	err = json.Unmarshal(responseBody, &machine)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &machine, nil
}

// CreateMachineRequest is the payload for creating a new Machine
type CreateMachineRequest struct {
	Name                    string         `json:"name,omitempty"`
	Region                  string         `json:"region,omitempty"`
	Config                  *MachineConfig `json:"config"`
	SkipLaunch              bool           `json:"skip_launch,omitempty"`
	SkipServiceRegistration bool           `json:"skip_service_registration,omitempty"`
	LeaseTTL                int            `json:"lease_ttl,omitempty"`
}

// CreateMachine creates a new Machine in an app
func (c *Client) CreateMachine(appName string, req CreateMachineRequest) (*Machine, error) {
	url := fmt.Sprintf("%s/v1/apps/%s/machines", c.BaseURL, appName)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var machine Machine
	err = json.Unmarshal(responseBody, &machine)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &machine, nil
}

// UpdateMachineRequest is the payload for updating a Machine
type UpdateMachineRequest struct {
	Config *MachineConfig `json:"config"`
}

// UpdateMachine updates an existing Machine
func (c *Client) UpdateMachine(appName, machineID string, req UpdateMachineRequest) (*Machine, error) {
	url := fmt.Sprintf("%s/v1/apps/%s/machines/%s", c.BaseURL, appName, machineID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var machine Machine
	err = json.Unmarshal(responseBody, &machine)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &machine, nil
}

// StartMachine starts a stopped Machine
func (c *Client) StartMachine(appName, machineID string) error {
	url := fmt.Sprintf("%s/v1/apps/%s/machines/%s/start", c.BaseURL, appName, machineID)
	_, err := c.execRequest(http.MethodPost, url, nil)
	return err
}

// StopMachineRequest is the optional payload for stopping a Machine
type StopMachineRequest struct {
	Signal  string `json:"signal,omitempty"`
	Timeout string `json:"timeout,omitempty"`
}

// StopMachine stops a running Machine
func (c *Client) StopMachine(appName, machineID string, req *StopMachineRequest) error {
	url := fmt.Sprintf("%s/v1/apps/%s/machines/%s/stop", c.BaseURL, appName, machineID)

	var body io.Reader
	if req != nil {
		bodyBytes, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("error marshaling request: %v", err)
		}
		body = bytes.NewReader(bodyBytes)
	}

	_, err := c.execRequest(http.MethodPost, url, body)
	return err
}

// DeleteMachine permanently deletes a Machine
func (c *Client) DeleteMachine(appName, machineID string, force bool) error {
	url := fmt.Sprintf("%s/v1/apps/%s/machines/%s", c.BaseURL, appName, machineID)
	if force {
		url += "?force=true"
	}

	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// SuspendMachine suspends a Machine (takes a snapshot)
func (c *Client) SuspendMachine(appName, machineID string) error {
	url := fmt.Sprintf("%s/v1/apps/%s/machines/%s/suspend", c.BaseURL, appName, machineID)
	_, err := c.execRequest(http.MethodPost, url, nil)
	return err
}

// WaitForState waits for a Machine to reach a specified state
func (c *Client) WaitForState(appName, machineID, state string, timeoutSeconds int) error {
	url := fmt.Sprintf("%s/v1/apps/%s/machines/%s/wait?state=%s", c.BaseURL, appName, machineID, state)
	if timeoutSeconds > 0 {
		url += fmt.Sprintf("&timeout=%d", timeoutSeconds)
	}

	_, err := c.execRequest(http.MethodGet, url, nil)
	return err
}
