package cloudsmith

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	defaultBaseURL = "https://api.cloudsmith.io"
)

type Client struct {
	APIKey    string
	Workspace string
	BaseURL   string
	http      core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	if integration == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := integration.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("API key not configured: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return nil, fmt.Errorf("API key is required")
	}

	workspace, err := integration.GetConfig("workspace")
	if err != nil {
		return nil, fmt.Errorf("workspace not configured: %w", err)
	}

	ws := strings.TrimSpace(string(workspace))

	return &Client{
		APIKey:    key,
		Workspace: ws,
		BaseURL:   defaultBaseURL,
		http:      httpClient,
	}, nil
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, []byte, error) {
	finalURL := path
	if !strings.HasPrefix(path, "http") {
		finalURL = c.BaseURL + path
	}

	req, err := http.NewRequest(method, finalURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.APIKey)

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
		return nil, nil, fmt.Errorf("request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	return res, responseBody, nil
}

func (c *Client) ValidateCredentials() error {
	_, _, err := c.doRequest(http.MethodGet, "/user/self/", nil)
	return err
}

type Repository struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (c *Client) ListRepositories(namespace string) ([]Repository, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	path := fmt.Sprintf("/repos/%s/", namespace)
	_, responseBody, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var repositories []Repository
	if err := json.Unmarshal(responseBody, &repositories); err != nil {
		return nil, fmt.Errorf("failed to parse repositories response: %w", err)
	}

	return repositories, nil
}

type PackageInfo struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Format         string `json:"format"`
	Size           int64  `json:"size"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	CDNURL         string `json:"cdn_url"`
	Status         string `json:"status_str"`
}

func (c *Client) GetPackage(namespace, repo, identifier string) (*PackageInfo, error) {
	if namespace == "" || repo == "" || identifier == "" {
		return nil, fmt.Errorf("namespace, repo, and identifier are required")
	}

	path := fmt.Sprintf("/packages/%s/%s/%s/", namespace, repo, identifier)
	_, responseBody, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var pkg PackageInfo
	if err := json.Unmarshal(responseBody, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package response: %w", err)
	}

	return &pkg, nil
}

type WebhookTemplate struct {
	Event    string `json:"event"`
	Template string `json:"template"`
}

type CreateWebhookRequest struct {
	TargetURL string            `json:"target_url"`
	Events    []string          `json:"events"`
	IsActive  bool              `json:"is_active"`
	Templates []WebhookTemplate `json:"templates"`
}

type CreateWebhookResponse struct {
	SlugPerm string `json:"slug_perm"`
}

func (c *Client) CreateWebhook(namespace, repo, targetURL string, events []string) (string, error) {
	if namespace == "" || repo == "" || targetURL == "" {
		return "", fmt.Errorf("namespace, repo, and targetURL are required")
	}

	// Cloudsmith requires a templates entry per subscribed event. The Template field
	// holds a Handlebars template string for custom payload formatting; an empty string
	// uses the default JSON payload (request_body_format = 0).
	templates := make([]WebhookTemplate, 0, len(events))
	for _, event := range events {
		templates = append(templates, WebhookTemplate{Event: event, Template: ""})
	}

	payload, err := json.Marshal(CreateWebhookRequest{
		TargetURL: targetURL,
		Events:    events,
		IsActive:  true,
		Templates: templates,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal webhook request: %w", err)
	}

	path := fmt.Sprintf("/webhooks/%s/%s/", namespace, repo)
	_, responseBody, err := c.doRequest(http.MethodPost, path, strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}

	var response CreateWebhookResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse webhook response: %w", err)
	}

	return response.SlugPerm, nil
}

func (c *Client) DeleteWebhook(namespace, repo, slugPerm string) error {
	if namespace == "" || repo == "" || slugPerm == "" {
		return fmt.Errorf("namespace, repo, and slugPerm are required")
	}

	path := fmt.Sprintf("/webhooks/%s/%s/%s/", namespace, repo, slugPerm)
	_, _, err := c.doRequest(http.MethodDelete, path, nil)
	return err
}
