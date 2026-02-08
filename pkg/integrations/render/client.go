package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultRenderBaseURL = "https://api.render.com/v1"

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type workspaceWithCursor struct {
	Cursor string `json:"cursor"`
	// Render docs call this a workspace, but the API response uses the legacy "owner" key.
	Workspace Workspace `json:"owner"`
}

type Service struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type serviceWithCursor struct {
	Cursor  string  `json:"cursor"`
	Service Service `json:"service"`
}

type Webhook struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"ownerId"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Enabled     bool     `json:"enabled"`
	EventFilter []string `json:"eventFilter"`
	Secret      string   `json:"secret"`
}

type webhookWithCursor struct {
	Cursor  string  `json:"cursor"`
	Webhook Webhook `json:"webhook"`
}

type CreateWebhookRequest struct {
	WorkspaceID string   `json:"ownerId"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Enabled     bool     `json:"enabled"`
	EventFilter []string `json:"eventFilter"`
}

type UpdateWebhookRequest struct {
	Name        string   `json:"name,omitempty"`
	URL         string   `json:"url,omitempty"`
	Enabled     bool     `json:"enabled"`
	EventFilter []string `json:"eventFilter,omitempty"`
}

type deployRequest struct {
	ClearCache string `json:"clearCache"`
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	trimmedAPIKey := strings.TrimSpace(string(apiKey))
	if trimmedAPIKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	return &Client{
		APIKey:  trimmedAPIKey,
		BaseURL: defaultRenderBaseURL,
		http:    httpClient,
	}, nil
}

func (c *Client) Verify() error {
	query := url.Values{}
	query.Set("limit", "1")
	_, _, err := c.execRequestWithResponse(http.MethodGet, "/services", query, nil)
	return err
}

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	query := url.Values{}
	query.Set("limit", "100")

	_, body, err := c.execRequestWithResponse(http.MethodGet, "/owners", query, nil)
	if err != nil {
		return nil, err
	}

	return parseWorkspaces(body)
}

func (c *Client) ListServices(workspaceID string) ([]Service, error) {
	query := url.Values{}
	query.Set("limit", "100")
	if strings.TrimSpace(workspaceID) != "" {
		query.Set("ownerId", strings.TrimSpace(workspaceID))
	}

	_, body, err := c.execRequestWithResponse(http.MethodGet, "/services", query, nil)
	if err != nil {
		return nil, err
	}

	return parseServices(body)
}

func (c *Client) ListWebhooks(workspaceID string) ([]Webhook, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspaceID is required")
	}

	query := url.Values{}
	query.Set("ownerId", workspaceID)
	query.Set("limit", "100")

	_, body, err := c.execRequestWithResponse(http.MethodGet, "/webhooks", query, nil)
	if err != nil {
		return nil, err
	}

	return parseWebhooks(body)
}

func (c *Client) GetWebhook(webhookID string) (*Webhook, error) {
	if webhookID == "" {
		return nil, fmt.Errorf("webhookID is required")
	}

	_, body, err := c.execRequestWithResponse(http.MethodGet, "/webhooks/"+url.PathEscape(webhookID), nil, nil)
	if err != nil {
		return nil, err
	}

	return parseWebhook(body)
}

func (c *Client) CreateWebhook(request CreateWebhookRequest) (*Webhook, error) {
	if request.WorkspaceID == "" {
		return nil, fmt.Errorf("workspaceID is required")
	}
	if request.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	if request.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	_, body, err := c.execRequestWithResponse(http.MethodPost, "/webhooks", nil, request)
	if err != nil {
		return nil, err
	}

	return parseWebhook(body)
}

func (c *Client) UpdateWebhook(webhookID string, request UpdateWebhookRequest) (*Webhook, error) {
	if webhookID == "" {
		return nil, fmt.Errorf("webhookID is required")
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodPatch,
		"/webhooks/"+url.PathEscape(webhookID),
		nil,
		request,
	)
	if err != nil {
		return nil, err
	}

	return parseWebhook(body)
}

func (c *Client) DeleteWebhook(webhookID string) error {
	if webhookID == "" {
		return fmt.Errorf("webhookID is required")
	}

	_, _, err := c.execRequestWithResponse(http.MethodDelete, "/webhooks/"+url.PathEscape(webhookID), nil, nil)
	return err
}

func (c *Client) TriggerDeploy(serviceID string, clearCache bool) (map[string]any, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("serviceID is required")
	}

	clearCacheValue := "do_not_clear"
	if clearCache {
		clearCacheValue = "clear"
	}

	_, body, err := c.execRequestWithResponse(
		http.MethodPost,
		"/services/"+url.PathEscape(serviceID)+"/deploys",
		nil,
		deployRequest{ClearCache: clearCacheValue},
	)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deploy response: %w", err)
	}

	if deployValue, ok := payload["deploy"]; ok {
		deployMap, ok := deployValue.(map[string]any)
		if ok {
			return deployMap, nil
		}
	}

	return payload, nil
}

func parseWorkspaces(body []byte) ([]Workspace, error) {
	withCursor := []workspaceWithCursor{}
	if err := json.Unmarshal(body, &withCursor); err == nil {
		workspaces := make([]Workspace, 0, len(withCursor))
		for _, item := range withCursor {
			if item.Workspace.ID == "" {
				continue
			}
			workspaces = append(workspaces, item.Workspace)
		}

		if len(withCursor) > 0 {
			return workspaces, nil
		}
	}

	plainWorkspaces := []Workspace{}
	if err := json.Unmarshal(body, &plainWorkspaces); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workspaces response: %w", err)
	}

	return plainWorkspaces, nil
}

func parseServices(body []byte) ([]Service, error) {
	withCursor := []serviceWithCursor{}
	if err := json.Unmarshal(body, &withCursor); err == nil {
		services := make([]Service, 0, len(withCursor))
		for _, item := range withCursor {
			if item.Service.ID == "" {
				continue
			}
			services = append(services, item.Service)
		}

		if len(withCursor) > 0 {
			return services, nil
		}
	}

	plainServices := []Service{}
	if err := json.Unmarshal(body, &plainServices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal services response: %w", err)
	}

	return plainServices, nil
}

func parseWebhooks(body []byte) ([]Webhook, error) {
	withCursor := []webhookWithCursor{}
	if err := json.Unmarshal(body, &withCursor); err == nil {
		webhooks := make([]Webhook, 0, len(withCursor))
		for _, item := range withCursor {
			if item.Webhook.ID == "" {
				continue
			}
			webhooks = append(webhooks, item.Webhook)
		}

		if len(withCursor) > 0 {
			return webhooks, nil
		}
	}

	plainWebhooks := []Webhook{}
	if err := json.Unmarshal(body, &plainWebhooks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhooks response: %w", err)
	}

	return plainWebhooks, nil
}

func parseWebhook(body []byte) (*Webhook, error) {
	webhook := Webhook{}
	if err := json.Unmarshal(body, &webhook); err == nil && webhook.ID != "" {
		return &webhook, nil
	}

	wrapper := struct {
		Webhook Webhook `json:"webhook"`
	}{}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook response: %w", err)
	}

	if wrapper.Webhook.ID == "" {
		return nil, fmt.Errorf("webhook id is missing in response")
	}

	return &wrapper.Webhook, nil
}

func (c *Client) execRequestWithResponse(
	method string,
	path string,
	query url.Values,
	payload any,
) (*http.Response, []byte, error) {
	endpoint := c.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		encodedBody, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(encodedBody)
	}

	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return res, responseBody, nil
}
