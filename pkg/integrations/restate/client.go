package restate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	AdminURL   string
	IngressURL string
	AuthToken  string
	http       core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	adminURL, err := ctx.GetConfig("adminUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting adminUrl: %v", err)
	}

	ingressURL, err := ctx.GetConfig("ingressUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting ingressUrl: %v", err)
	}

	authToken := ""
	tokenBytes, err := ctx.GetConfig("authToken")
	if err == nil {
		authToken = string(tokenBytes)
	}

	return &Client{
		AdminURL:   strings.TrimRight(string(adminURL), "/"),
		IngressURL: strings.TrimRight(string(ingressURL), "/"),
		AuthToken:  authToken,
		http:       http,
	}, nil
}

func (c *Client) doRequest(method, url string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("error reading body: %v", err)
	}

	return responseBody, res.StatusCode, nil
}

func (c *Client) adminRequest(method, path string, body io.Reader) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.AdminURL, path)

	responseBody, statusCode, err := c.doRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", statusCode, string(responseBody))
	}

	return responseBody, nil
}

func (c *Client) ingressRequest(method, path string, body io.Reader) ([]byte, int, error) {
	url := fmt.Sprintf("%s%s", c.IngressURL, path)
	return c.doRequest(method, url, body)
}

// CheckHealth verifies the Restate server is reachable via the health endpoint.
func (c *Client) CheckHealth() error {
	_, err := c.adminRequest(http.MethodGet, "/health", nil)
	return err
}

// GetClusterHealth retrieves the cluster health status.
func (c *Client) GetClusterHealth() (map[string]any, error) {
	body, err := c.adminRequest(http.MethodGet, "/cluster-health", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return result, nil
}

// GetVersion retrieves the Restate server version.
func (c *Client) GetVersion() (map[string]any, error) {
	body, err := c.adminRequest(http.MethodGet, "/version", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return result, nil
}

// Deployment represents a Restate deployment.
type Deployment struct {
	ID       string `json:"id"`
	URI      string `json:"uri"`
	Services []any  `json:"services"`
	Protocol string `json:"protocol_type"`
}

// RegisterDeploymentRequest represents the request to register a deployment.
type RegisterDeploymentRequest struct {
	URI               string            `json:"uri"`
	AdditionalHeaders map[string]string `json:"additional_headers,omitempty"`
	Force             bool              `json:"force,omitempty"`
	DryRun            bool              `json:"dry_run,omitempty"`
	UseHTTP11         bool              `json:"use_http_11,omitempty"`
}

// RegisterDeployment registers a new deployment with Restate.
func (c *Client) RegisterDeployment(req RegisterDeploymentRequest) (map[string]any, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.adminRequest(http.MethodPost, "/deployments", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return result, nil
}

// RemoveDeployment removes a deployment from Restate.
func (c *Client) RemoveDeployment(deploymentID string, force bool) error {
	path := fmt.Sprintf("/deployments/%s", deploymentID)
	if force {
		path += "?force=true"
	}

	_, err := c.adminRequest(http.MethodDelete, path, nil)
	return err
}

// Service represents a Restate service.
type Service struct {
	Name                 string `json:"name"`
	Revision             int    `json:"revision"`
	Type                 string `json:"ty"`
	Deployment           string `json:"deployment_id"`
	Public               bool   `json:"public"`
	IdempotencyRetention string `json:"idempotency_retention,omitempty"`
}

// GetService retrieves details for a specific service.
func (c *Client) GetService(serviceName string) (map[string]any, error) {
	path := fmt.Sprintf("/services/%s", serviceName)

	responseBody, err := c.adminRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return result, nil
}

// ListServicesResponse represents the response from listing services.
type ListServicesResponse struct {
	Services []map[string]any `json:"services"`
}

// ListAllServices retrieves all registered services.
func (c *Client) ListAllServices() ([]map[string]any, error) {
	responseBody, err := c.adminRequest(http.MethodGet, "/services", nil)
	if err != nil {
		return nil, err
	}

	var result ListServicesResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return result.Services, nil
}

// InvokeHandlerResponse represents a synchronous invocation response.
type InvokeHandlerResponse struct {
	StatusCode int
	Body       []byte
}

// InvokeHandler invokes a handler synchronously and waits for the response.
func (c *Client) InvokeHandler(service, handler string, payload []byte, idempotencyKey string) (*InvokeHandlerResponse, error) {
	path := fmt.Sprintf("/%s/%s", service, handler)

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	url := fmt.Sprintf("%s%s", c.IngressURL, path)
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	return &InvokeHandlerResponse{
		StatusCode: res.StatusCode,
		Body:       responseBody,
	}, nil
}

// SendHandlerResponse represents an async invocation response.
type SendHandlerResponse struct {
	InvocationID string `json:"invocationId"`
	Status       string `json:"status"`
}

// SendHandler sends a fire-and-forget invocation.
func (c *Client) SendHandler(service, handler string, payload []byte, idempotencyKey string) (*SendHandlerResponse, error) {
	path := fmt.Sprintf("/%s/%s/send", service, handler)

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	url := fmt.Sprintf("%s%s", c.IngressURL, path)
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	var result SendHandlerResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &result, nil
}

// SendDelayedHandler sends a delayed fire-and-forget invocation.
func (c *Client) SendDelayedHandler(service, handler string, payload []byte, delay string, idempotencyKey string) (*SendHandlerResponse, error) {
	path := fmt.Sprintf("/%s/%s/send?delay=%s", service, handler, delay)

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	url := fmt.Sprintf("%s%s", c.IngressURL, path)
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	var result SendHandlerResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &result, nil
}

// CancelInvocation cancels a running invocation gracefully.
func (c *Client) CancelInvocation(invocationID string) error {
	path := fmt.Sprintf("/invocations/%s/cancel", invocationID)
	_, err := c.adminRequest(http.MethodPatch, path, nil)
	return err
}

// KillInvocation force-kills a running invocation without running compensation.
func (c *Client) KillInvocation(invocationID string) error {
	path := fmt.Sprintf("/invocations/%s/kill", invocationID)
	_, err := c.adminRequest(http.MethodPatch, path, nil)
	return err
}

// PurgeInvocation purges a completed invocation and its associated data.
func (c *Client) PurgeInvocation(invocationID string) error {
	path := fmt.Sprintf("/invocations/%s/purge", invocationID)
	_, err := c.adminRequest(http.MethodPatch, path, nil)
	return err
}
