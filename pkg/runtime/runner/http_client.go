package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type httpClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func newHTTPClient(cfg Config) *httpClient {
	return &httpClient{
		baseURL:   strings.TrimRight(cfg.HTTPURL, "/"),
		authToken: cfg.AuthToken,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *httpClient) SetupTrigger(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	return c.post(ctx, fmt.Sprintf("/v1/triggers/%s/setup", url.PathEscape(name)), req)
}

func (c *httpClient) SetupComponent(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	return c.post(ctx, fmt.Sprintf("/v1/components/%s/setup", url.PathEscape(name)), req)
}

func (c *httpClient) ExecuteComponent(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	return c.post(ctx, fmt.Sprintf("/v1/components/%s/execute", url.PathEscape(name)), req)
}

func (c *httpClient) SyncIntegration(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	return c.post(ctx, fmt.Sprintf("/v1/integrations/%s/sync", url.PathEscape(name)), req)
}

func (c *httpClient) CleanupIntegration(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	return c.post(ctx, fmt.Sprintf("/v1/integrations/%s/cleanup", url.PathEscape(name)), req)
}

func (c *httpClient) ListCapabilities(ctx context.Context) ([]Capability, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/capabilities", nil)
	if err != nil {
		return nil, err
	}

	if c.authToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("runtime runner request failed: status %d", response.StatusCode)
	}

	var payload struct {
		Capabilities []Capability `json:"capabilities"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return payload.Capabilities, nil
}

func (c *httpClient) post(ctx context.Context, path string, req OperationRequest) (*OperationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal runtime runner request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	response, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("runtime runner request failed: status %d", response.StatusCode)
	}

	var payload OperationResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode runtime runner response: %w", err)
	}

	return &payload, nil
}
