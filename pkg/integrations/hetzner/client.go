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
	Name             string                 `json:"name"`
	ServerType       string                 `json:"server_type"`
	Image            string                 `json:"image"`
	Location         string                 `json:"location,omitempty"`
	SSHKeys          []string               `json:"ssh_keys,omitempty"`
	Firewalls        []createServerFirewall `json:"firewalls,omitempty"`
	UserData         string                 `json:"user_data,omitempty"`
	StartAfterCreate *bool                  `json:"start_after_create,omitempty"`
}

type createServerFirewall struct {
	Firewall string `json:"firewall"`
}

type createServerResponse struct {
	Server *ServerResponse `json:"server"`
	Action *ActionResponse `json:"action"`
}

type createSnapshotRequest struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type createSnapshotResponse struct {
	Image  *ImageResponse  `json:"image"`
	Action *ActionResponse `json:"action"`
}

const (
	LoadBalancerAlgorithmTypeRoundRobin       = "round_robin"
	LoadBalancerAlgorithmTypeLeastConnections = "least_connections"
)

type LoadBalancerAlgorithmType struct {
	Type string `json:"type"`
}

type createLoadBalancerRequest struct {
	Name             string                    `json:"name"`
	LoadBalancerType string                    `json:"load_balancer_type"`
	Location         string                    `json:"location"`
	Algorithm        LoadBalancerAlgorithmType `json:"algorithm"`
}

type createLoadBalancerResponse struct {
	LoadBalancer *ServerResponse `json:"load_balancer"`
	Action       *ActionResponse `json:"action"`
}

type ServerResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Created string `json:"created"`
	Image   struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	} `json:"image"`
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

	return decodeValue(raw, result)
}

// decodeValue decodes an already-parsed JSON value into the target struct,
// with the same weak typing decodeJSON applies to whole response bodies.
func decodeValue(value any, result any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           result,
		TagName:          "json",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(value)
}

// listPerPage is the largest page Hetzner serves. The API defaults to 25 items
// and caps per_page at 50, so every listing has to follow next_page to the end.
const listPerPage = 50

type paginationMeta struct {
	Pagination struct {
		NextPage *int `json:"next_page"`
	} `json:"pagination"`
}

// listAll returns every page of a Hetzner collection endpoint. collection is the
// key the items arrive under (for example "servers" for /servers), and the
// walk follows meta.pagination.next_page until the API reports no further page.
func listAll[T any](c *Client, path, collection string) ([]T, error) {
	all := []T{}
	page := 1

	for {
		resp, err := c.do("GET", fmt.Sprintf("%s?per_page=%d&page=%d", path, listPerPage, page), nil)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, c.parseError(resp)
		}

		var body map[string]any
		if err := decodeJSON(resp.Body, &body); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode list %s response: %w", collection, err)
		}
		resp.Body.Close()

		var items []T
		if err := decodeValue(body[collection], &items); err != nil {
			return nil, fmt.Errorf("decode list %s response: %w", collection, err)
		}
		all = append(all, items...)

		var meta paginationMeta
		if err := decodeValue(body["meta"], &meta); err != nil {
			return nil, fmt.Errorf("decode list %s pagination: %w", collection, err)
		}

		// next_page is null on the last page. The <= guard keeps a malformed
		// response from looping forever.
		if meta.Pagination.NextPage == nil || *meta.Pagination.NextPage <= page {
			return all, nil
		}
		page = *meta.Pagination.NextPage
	}
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

// server actions

func (c *Client) CreateServer(name, serverType, image, location string, sshKeys []string, firewall, userData string) (*ServerResponse, *ActionResponse, error) {
	req := createServerRequest{
		Name:       name,
		ServerType: serverType,
		Image:      image,
		Location:   location,
		SSHKeys:    sshKeys,
		UserData:   userData,
	}
	if strings.TrimSpace(firewall) != "" {
		req.Firewalls = []createServerFirewall{
			{Firewall: firewall},
		}
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

func (c *Client) CreateServerSnapshot(serverID, description string) (*ImageResponse, *ActionResponse, error) {
	req := createSnapshotRequest{
		Type:        "snapshot",
		Description: description,
	}

	resp, err := c.do("POST", "/servers/"+serverID+"/actions/create_image", req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, nil, c.parseError(resp)
	}

	var out createSnapshotResponse
	if err := decodeJSON(resp.Body, &out); err != nil {
		return nil, nil, fmt.Errorf("decode create snapshot response: %w", err)
	}
	if out.Image == nil || out.Action == nil {
		return nil, nil, fmt.Errorf("create snapshot response missing image or action")
	}
	return out.Image, out.Action, nil
}

func (c *Client) ListServers() ([]ServerResponse, error) {
	return listAll[ServerResponse](c, "/servers", "servers")
}

// load balancer actions

func (c *Client) ListLoadBalancers() ([]ServerResponse, error) {
	return listAll[ServerResponse](c, "/load_balancers", "load_balancers")
}

func (c *Client) CreateLoadBalancer(name, loadBalancerType, location, algorithm string) (*ServerResponse, *ActionResponse, error) {
	req := createLoadBalancerRequest{
		Name:             name,
		LoadBalancerType: loadBalancerType,
		Location:         location,
		Algorithm: LoadBalancerAlgorithmType{
			Type: algorithm,
		},
	}

	resp, err := c.do("POST", "/load_balancers", req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, nil, c.parseError(resp)
	}

	var out createLoadBalancerResponse
	if err := decodeJSON(resp.Body, &out); err != nil {
		return nil, nil, fmt.Errorf("decode create load balancer response: %w", err)
	}
	if out.LoadBalancer == nil || out.Action == nil {
		return nil, nil, fmt.Errorf("create load balancer missing load_balancer or action")
	}
	return out.LoadBalancer, out.Action, nil
}

func (c *Client) DeleteLoadBalancer(loadBalancerID string) error {
	resp, err := c.do("DELETE", "/load_balancers/"+loadBalancerID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// server info actions

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
	return listAll[ServerTypeResponse](c, "/server_types", "server_types")
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
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	ID          int    `json:"id"`
}

func (c *Client) ListImages() ([]ImageResponse, error) {
	images, err := listAll[ImageResponse](c, "/images", "images")
	if err != nil {
		return nil, err
	}

	// A page walk can see the same image twice when the underlying list changes
	// between requests, so only the first occurrence of an ID is kept.
	unique := make([]ImageResponse, 0, len(images))
	seen := map[int]struct{}{}
	for _, image := range images {
		if _, ok := seen[image.ID]; ok {
			continue
		}
		seen[image.ID] = struct{}{}
		unique = append(unique, image)
	}

	return unique, nil
}

func (c *Client) GetImage(imageID string) (*ImageResponse, error) {
	resp, err := c.do("GET", "/images/"+imageID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		Image ImageResponse `json:"image"`
	}
	if err := decodeJSON(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("decode get image response: %w", err)
	}
	return &out.Image, nil
}

func (c *Client) DeleteImage(imageID string) error {
	resp, err := c.do("DELETE", "/images/"+imageID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

type LocationResponse struct {
	Name        string `json:"name"`
	ID          int    `json:"id"`
	Description string `json:"description"`
	City        string `json:"city"`
	Country     string `json:"country"`
}

type FirewallResponse struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
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
	return listAll[LocationResponse](c, "/locations", "locations")
}

func (c *Client) ListFirewalls() ([]FirewallResponse, error) {
	return listAll[FirewallResponse](c, "/firewalls", "firewalls")
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

// load balancer actions

type LoadBalancerTypeResponse struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (c *Client) ListLoadBalancerTypes() ([]LoadBalancerTypeResponse, error) {
	return listAll[LoadBalancerTypeResponse](c, "/load_balancer_types", "load_balancer_types")
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

// resolveLoadBalancerID extracts the load balancer ID from the configuration map,
// handling both string values and float64 values (which occur when
// template expressions resolve to JSON numbers).
func resolveLoadBalancerID(config any) (string, error) {
	m, ok := config.(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid configuration type")
	}

	raw, ok := m["loadBalancer"]
	if !ok {
		return "", fmt.Errorf("loadBalancer is required")
	}

	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return "", fmt.Errorf("loadBalancer is required")
		}
		return s, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	default:
		return "", fmt.Errorf("invalid loadBalancer value: %v", raw)
	}
}

// resolveImageID extracts the image ID from configuration, handling
// both string and numeric values.
func resolveImageID(config any, fieldName string) (string, error) {
	m, ok := config.(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid configuration type")
	}

	raw, ok := m[fieldName]
	if !ok {
		return "", fmt.Errorf("%s is required", fieldName)
	}

	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return "", fmt.Errorf("%s is required", fieldName)
		}
		return s, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	default:
		return "", fmt.Errorf("invalid %s value: %v", fieldName, raw)
	}
}
