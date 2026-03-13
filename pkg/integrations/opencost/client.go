package opencost

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const MaxResponseSize = 1 * 1024 * 1024 // 1MB

type Client struct {
	baseURL     string
	authType    string
	username    string
	password    string
	bearerToken string
	http        core.HTTPContext
}

type AllocationResponse struct {
	Code    int                          `json:"code"`
	Data    []map[string]AllocationEntry `json:"data"`
	Message string                       `json:"message,omitempty"`
}

type AllocationEntry struct {
	Name              string           `json:"name"`
	Properties        AllocationProps  `json:"properties"`
	Window            AllocationWindow `json:"window"`
	Start             string           `json:"start"`
	End               string           `json:"end"`
	CPUCost           float64          `json:"cpuCost"`
	GPUCost           float64          `json:"gpuCost"`
	RAMCost           float64          `json:"ramCost"`
	PVCost            float64          `json:"pvCost"`
	NetworkCost       float64          `json:"networkCost"`
	TotalCost         float64          `json:"totalCost"`
	CPUEfficiency     float64          `json:"cpuEfficiency"`
	RAMEfficiency     float64          `json:"ramEfficiency"`
	TotalEfficiency   float64          `json:"totalEfficiency"`
	ExternalCost      float64          `json:"externalCost,omitempty"`
	RawAllocationOnly map[string]any   `json:"rawAllocationOnly,omitempty"`
}

type AllocationProps struct {
	Cluster        string `json:"cluster,omitempty"`
	Node           string `json:"node,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	ControllerKind string `json:"controllerKind,omitempty"`
	Controller     string `json:"controller,omitempty"`
	Pod            string `json:"pod,omitempty"`
	Container      string `json:"container,omitempty"`
}

type AllocationWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func NewClient(httpContext core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	baseURL, err := requiredConfig(integration, "baseURL")
	if err != nil {
		return nil, err
	}

	authType, err := requiredConfig(integration, "authType")
	if err != nil {
		return nil, err
	}

	client := &Client{
		baseURL:  normalizeBaseURL(baseURL),
		authType: authType,
		http:     httpContext,
	}

	switch authType {
	case AuthTypeNone:
		return client, nil
	case AuthTypeBasic:
		username, err := requiredConfig(integration, "username")
		if err != nil {
			return nil, fmt.Errorf("username is required when authType is basic")
		}
		password, err := requiredConfig(integration, "password")
		if err != nil {
			return nil, fmt.Errorf("password is required when authType is basic")
		}

		client.username = username
		client.password = password
		return client, nil
	case AuthTypeBearer:
		bearerToken, err := requiredConfig(integration, "bearerToken")
		if err != nil {
			return nil, fmt.Errorf("bearerToken is required when authType is bearer")
		}

		client.bearerToken = bearerToken
		return client, nil
	default:
		return nil, fmt.Errorf("invalid authType %q", authType)
	}
}

func requiredConfig(ctx core.IntegrationContext, name string) (string, error) {
	value, err := ctx.GetConfig(name)
	if err != nil {
		return "", fmt.Errorf("%s is required", name)
	}

	s := string(value)
	if s == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return s, nil
}

func normalizeBaseURL(baseURL string) string {
	if baseURL == "/" {
		return baseURL
	}

	for len(baseURL) > 0 && strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL[:len(baseURL)-1]
	}

	return baseURL
}

func (c *Client) GetAllocation(window, aggregate string) ([]map[string]AllocationEntry, error) {
	params := url.Values{}
	params.Set("window", window)
	params.Set("aggregate", aggregate)

	apiPath := "/allocation/compute?" + params.Encode()

	body, err := c.execRequest(http.MethodGet, apiPath)
	if err != nil {
		return nil, err
	}

	response := AllocationResponse{}
	if err := decodeResponse(body, &response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, formatOpenCostError(response.Code, response.Message)
	}

	return response.Data, nil
}

func (c *Client) GetAllocationWithStep(window, aggregate, step string) ([]map[string]AllocationEntry, error) {
	params := url.Values{}
	params.Set("window", window)
	params.Set("aggregate", aggregate)
	if step != "" {
		params.Set("step", step)
	}

	apiPath := "/allocation/compute?" + params.Encode()

	body, err := c.execRequest(http.MethodGet, apiPath)
	if err != nil {
		return nil, err
	}

	response := AllocationResponse{}
	if err := decodeResponse(body, &response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, formatOpenCostError(response.Code, response.Message)
	}

	return response.Data, nil
}

func (c *Client) execRequest(method string, path string) ([]byte, error) {
	apiURL := c.baseURL
	if strings.HasPrefix(path, "/") {
		apiURL += path
	} else {
		apiURL += "/" + path
	}

	req, err := http.NewRequest(method, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if err := c.setAuth(req); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	limitedReader := io.LimitReader(res.Body, MaxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) > MaxResponseSize {
		return nil, fmt.Errorf("response too large: exceeds maximum size of %d bytes", MaxResponseSize)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", res.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) setAuth(req *http.Request) error {
	switch c.authType {
	case AuthTypeNone:
		return nil
	case AuthTypeBasic:
		req.SetBasicAuth(c.username, c.password)
		return nil
	case AuthTypeBearer:
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		return nil
	default:
		return fmt.Errorf("invalid authType %q", c.authType)
	}
}

func decodeResponse[T any](body []byte, out *T) error {
	if len(body) == 0 {
		return fmt.Errorf("empty response body")
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to decode response JSON: %w", err)
	}

	return nil
}

func formatOpenCostError(code int, message string) error {
	if message == "" {
		return fmt.Errorf("OpenCost API returned non-success status code %d", code)
	}

	return fmt.Errorf("OpenCost API error (code %d): %s", code, message)
}
