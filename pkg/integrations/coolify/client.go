package coolify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const apiPathPrefix = "/api/v1"

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
		return fmt.Sprintf("Coolify API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("Coolify API error %d: %s", e.StatusCode, e.Body)
}

// Application represents the fields the connector cares about from the
// Coolify applications API.
type Application struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	FQDN        string `json:"fqdn"`
	Description string `json:"description"`
	GitRepo     string `json:"git_repository"`
	GitBranch   string `json:"git_branch"`
}

// Service represents the fields the connector cares about from the
// Coolify services API.
type Service struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	FQDN        string `json:"fqdn"`
	Description string `json:"description"`
	ServerUUID  string `json:"server_uuid"`
}

// LifecycleResponse is the body returned by start/stop/restart endpoints.
// Coolify returns a small JSON object with a confirmation message.
type LifecycleResponse struct {
	Message string `json:"message"`
}

// DeployResponse is the body returned by GET /api/v1/deploy.
// It contains the queued deployment(s); we expose the first one.
type DeployResponse struct {
	Deployments []Deployment `json:"deployments"`
	Message     string       `json:"message"`
}

type Deployment struct {
	ResourceUUID   string `json:"resource_uuid"`
	DeploymentUUID string `json:"deployment_uuid"`
	Message        string `json:"message"`
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	baseURL, err := readBaseURL(integration)
	if err != nil {
		return nil, err
	}

	token, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("apiToken is required: %w", err)
	}

	return &Client{
		Token:   string(token),
		BaseURL: baseURL,
		http:    httpCtx,
	}, nil
}

// readBaseURL fetches and validates the Coolify base URL from the integration
// configuration. It strips any trailing slash and a trailing API path prefix
// (e.g. "/api/v1") so callers can append paths directly without producing a
// duplicated "/api/v1/api/v1" path.
func readBaseURL(integration core.IntegrationContext) (string, error) {
	raw, err := integration.GetConfig("baseUrl")
	if err != nil {
		return "", fmt.Errorf("baseUrl is required: %w", err)
	}

	value := strings.TrimSpace(string(raw))
	if value == "" {
		return "", fmt.Errorf("baseUrl is required")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid baseUrl: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid baseUrl: must include scheme and host (e.g. https://coolify.example.com)")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid baseUrl: unsupported scheme %q (expected http or https)", parsed.Scheme)
	}

	value = strings.TrimRight(value, "/")
	value = strings.TrimSuffix(value, apiPathPrefix)
	return strings.TrimRight(value, "/"), nil
}

func (c *Client) do(method, path string, query url.Values) (*http.Response, error) {
	target := c.BaseURL + apiPathPrefix + path
	if len(query) > 0 {
		target = target + "?" + query.Encode()
	}

	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(body)}

	var errPayload struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(body, &errPayload) == nil {
		switch {
		case errPayload.Message != "":
			apiErr.Message = errPayload.Message
		case errPayload.Error != "":
			apiErr.Message = errPayload.Error
		}
	}
	return apiErr
}

// Verify hits a lightweight endpoint to confirm credentials are valid.
// /api/v1/version returns the Coolify instance version when authenticated.
func (c *Client) Verify() error {
	resp, err := c.do(http.MethodGet, "/version", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

func (c *Client) ListApplications() ([]Application, error) {
	resp, err := c.do(http.MethodGet, "/applications", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var applications []Application
	if err := json.NewDecoder(resp.Body).Decode(&applications); err != nil {
		return nil, fmt.Errorf("decode list applications response: %w", err)
	}
	return applications, nil
}

func (c *Client) ListServices() ([]Service, error) {
	resp, err := c.do(http.MethodGet, "/services", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var services []Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("decode list services response: %w", err)
	}
	return services, nil
}

// LifecycleOperation is the operation requested on a Coolify resource.
type LifecycleOperation string

const (
	LifecycleStart   LifecycleOperation = "start"
	LifecycleStop    LifecycleOperation = "stop"
	LifecycleRestart LifecycleOperation = "restart"
)

// IsValid returns true when op is one of the known lifecycle operations.
func (op LifecycleOperation) IsValid() bool {
	switch op {
	case LifecycleStart, LifecycleStop, LifecycleRestart:
		return true
	default:
		return false
	}
}

func (c *Client) ControlApplication(uuid string, op LifecycleOperation) (*LifecycleResponse, error) {
	return c.lifecycle("/applications/"+uuid+"/"+string(op), op)
}

func (c *Client) ControlService(uuid string, op LifecycleOperation) (*LifecycleResponse, error) {
	return c.lifecycle("/services/"+uuid+"/"+string(op), op)
}

func (c *Client) lifecycle(path string, op LifecycleOperation) (*LifecycleResponse, error) {
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out LifecycleResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", op, err)
	}
	return &out, nil
}

// Deploy queues a deployment for the given application UUID. force=true
// triggers a fresh build instead of redeploying the existing image.
func (c *Client) Deploy(applicationUUID string, force bool) (*DeployResponse, error) {
	query := url.Values{}
	query.Set("uuid", applicationUUID)
	if force {
		query.Set("force", "true")
	}

	resp, err := c.do(http.MethodGet, "/deploy", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out DeployResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode deploy response: %w", err)
	}
	return &out, nil
}
