package hetzner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const defaultHetznerBaseURL = "https://api.hetzner.cloud/v1"

type Client struct {
	Token   string
	BaseURL string
	http    core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Hetzner API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("Hetzner API error %d: %s", e.StatusCode, e.Body)
}

type createServerRequest struct {
	Name             string   `json:"name"`
	ServerType       string   `json:"server_type"`
	Image            string   `json:"image"`
	Location         string   `json:"location,omitempty"`
	SSHKeys          []string `json:"ssh_keys,omitempty"`
	UserData         string   `json:"user_data,omitempty"`
	StartAfterCreate *bool    `json:"start_after_create,omitempty"`
}

type createServerResponse struct {
	Server *ServerResponse `json:"server"`
	Action *ActionResponse `json:"action"`
}

type ServerResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Created   string `json:"created"`
	PublicNet struct {
		IPv4 struct {
			IP string `json:"ip"`
		} `json:"ipv4"`
	} `json:"public_net"`
}

type ActionResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Command  string `json:"command"`
	Progress int    `json:"progress"`
	Started  string `json:"started"`
	Finished string `json:"finished"`
	Error    *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type getActionResponse struct {
	Action ActionResponse `json:"action"`
}

const (
	ActionStatusRunning = "running"
	ActionStatusSuccess = "success"
	ActionStatusError   = "error"
)

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	token, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("apiToken is required: %w", err)
	}
	return &Client{
		Token:   string(token),
		BaseURL: defaultHetznerBaseURL,
		http:    httpCtx,
	}, nil
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// decodeJSON decodes a JSON response body into the target struct.
// It uses json.Decoder.UseNumber() so that numeric IDs from the Hetzner API
// are preserved as strings (via mapstructure's WeaklyTypedInput).
func decodeJSON(r io.Reader, result any) error {
	var raw any
	dec := json.NewDecoder(r)
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return err
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           result,
		TagName:          "json",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(raw)
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	var errPayload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &errPayload) == nil && errPayload.Error.Message != "" {
		apiErr.Message = errPayload.Error.Message
		if resp.StatusCode == http.StatusUnprocessableEntity && strings.Contains(strings.ToLower(apiErr.Message), "unsupported location") {
			apiErr.Message = "the selected location is not available for this server type. Select a server type first; the Location dropdown then shows only locations that support it."
		}
	}
	return apiErr
}

func (c *Client) CreateServer(name, serverType, image, location string, sshKeys []string, userData string) (*ServerResponse, *ActionResponse, error) {
	req := createServerRequest{
		Name:       name,
		ServerType: serverType,
		Image:      image,
		Location:   location,
		SSHKeys:    sshKeys,
		UserData:   userData,
	}
	startAfter := true
	req.StartAfterCreate = &startAfter

	resp, err := c.do("POST", "/servers", req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, nil, c.parseError(resp)
	}

	var out createServerResponse
	if err := decodeJSON(resp.Body, &out); err != nil {
		return nil, nil, fmt.Errorf("decode create server response: %w", err)
	}
	if out.Server == nil || out.Action == nil {
		return nil, nil, fmt.Errorf("create server response missing server or action")
	}
	return out.Server, out.Action, nil
}

func (c *Client) GetAction(actionID string) (*ActionResponse, error) {
	resp, err := c.do("GET", "/actions/"+actionID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out getActionResponse
	if err := decodeJSON(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("decode get action response: %w", err)
	}
	return &out.Action, nil
}

func (c *Client) GetServer(serverID string) (*ServerResponse, error) {
	resp, err := c.do("GET", "/servers/"+serverID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		Server ServerResponse `json:"server"`
	}
	if err := decodeJSON(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("decode get server response: %w", err)
	}
	return &out.Server, nil
}

func (c *Client) DeleteServer(serverID string) (*ActionResponse, error) {
	resp, err := c.do("DELETE", "/servers/"+serverID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		Action ActionResponse `json:"action"`
	}
	if err := decodeJSON(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("decode delete server response: %w", err)
	}
	return &out.Action, nil
}

func (c *Client) ListServers() ([]ServerResponse, error) {
	resp, err := c.do("GET", "/servers?per_page=50", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		Servers []ServerResponse `json:"servers"`
	}
	if err := decodeJSON(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("decode list servers response: %w", err)
	}
	return out.Servers, nil
}

type ServerTypePrice struct {
	Location string `json:"location"`
}

type ServerTypeResponse struct {
	Name        string            `json:"name"`
	ID          int               `json:"id"`
	Description string            `json:"description"`
	Cores       int               `json:"cores"`
	Memory      float64           `json:"memory"`
	Disk        int               `json:"disk"`
	Prices      []ServerTypePrice `json:"prices"`
}

func (c *Client) ListServerTypes() ([]ServerTypeResponse, error) {
	resp, err := c.do("GET", "/server_types?per_page=50", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		ServerTypes []ServerTypeResponse `json:"server_types"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode list server types response: %w", err)
	}
	return out.ServerTypes, nil
}

// ServerTypeLocationNames returns the location names (e.g. fsn1, nbg1) where the given server type is available.
// Prices in the API list per-location pricing, so a price entry means the type is available there.
func (c *Client) ServerTypeLocationNames(serverTypeName string) ([]string, error) {
	types, err := c.ListServerTypes()
	if err != nil {
		return nil, err
	}
	for _, t := range types {
		if t.Name == serverTypeName {
			names := make([]string, 0, len(t.Prices))
			for _, p := range t.Prices {
				if p.Location != "" {
					names = append(names, p.Location)
				}
			}
			return names, nil
		}
	}
	return nil, fmt.Errorf("server type %q not found", serverTypeName)
}

// ServerTypeDisplayName returns a label for the server type including specs (e.g. "cpx11 — 2 vCPU, 2 GB RAM, 40 GB disk").
func (s *ServerTypeResponse) ServerTypeDisplayName() string {
	if s.Name == "" {
		return ""
	}
	var parts []string
	if s.Cores > 0 {
		parts = append(parts, fmt.Sprintf("%d vCPU", s.Cores))
	}
	if s.Memory > 0 {
		parts = append(parts, fmt.Sprintf("%.0f GB RAM", s.Memory))
	}
	if s.Disk > 0 {
		parts = append(parts, fmt.Sprintf("%d GB disk", s.Disk))
	}
	if len(parts) == 0 {
		return s.Name
	}
	return s.Name + " — " + strings.Join(parts, ", ")
}

type ImageResponse struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

func (c *Client) ListImages() ([]ImageResponse, error) {
	resp, err := c.do("GET", "/images?per_page=50&type=system", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		Images []ImageResponse `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode list images response: %w", err)
	}
	return out.Images, nil
}

type LocationResponse struct {
	Name        string `json:"name"`
	ID          int    `json:"id"`
	Description string `json:"description"`
	City        string `json:"city"`
	Country     string `json:"country"`
}

// LocationDisplayName returns a label for the location (e.g. "Nuremberg, DE (nbg1)").
func (l *LocationResponse) LocationDisplayName() string {
	if l.Name == "" {
		return ""
	}
	if l.City != "" && l.Country != "" {
		return fmt.Sprintf("%s, %s (%s)", l.City, l.Country, l.Name)
	}
	if l.City != "" {
		return fmt.Sprintf("%s (%s)", l.City, l.Name)
	}
	if l.Description != "" {
		return fmt.Sprintf("%s (%s)", l.Description, l.Name)
	}
	return l.Name
}

func (c *Client) ListLocations() ([]LocationResponse, error) {
	resp, err := c.do("GET", "/locations?per_page=50", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		Locations []LocationResponse `json:"locations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode list locations response: %w", err)
	}
	return out.Locations, nil
}

func (c *Client) Verify() error {
	resp, err := c.do("GET", "/servers?per_page=1", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// resolveServerID extracts the server ID from the configuration map,
// handling both string values and float64 values (which occur when
// template expressions resolve to JSON numbers).
func resolveServerID(config any) (string, error) {
	m, ok := config.(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid configuration type")
	}

	raw, ok := m["server"]
	if !ok {
		return "", fmt.Errorf("server is required")
	}

	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return "", fmt.Errorf("server is required")
		}
		return s, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	default:
		return "", fmt.Errorf("invalid server value: %v", raw)
	}
}
