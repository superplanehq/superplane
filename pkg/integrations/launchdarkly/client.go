package launchdarkly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const BaseURL = "https://app.launchdarkly.com"

// Project represents a LaunchDarkly project.
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ProjectListResponse is the API response for listing projects.
type ProjectListResponse struct {
	Items      []Project `json:"items"`
	TotalCount int       `json:"totalCount"`
}

// FeatureFlag represents a LaunchDarkly feature flag.
type FeatureFlag struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Kind         string `json:"kind"`
	CreationDate int64  `json:"creationDate"`
	Archived     bool   `json:"archived"`
	Temporary    bool   `json:"temporary"`
}

// FeatureFlagListResponse is the API response for listing feature flags.
type FeatureFlagListResponse struct {
	Items      []FeatureFlag `json:"items"`
	TotalCount int           `json:"totalCount"`
}

// Environment represents a LaunchDarkly environment within a project.
type Environment struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// EnvironmentListResponse is the API response for listing environments.
type EnvironmentListResponse struct {
	Items      []Environment `json:"items"`
	TotalCount int           `json:"totalCount"`
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type Client struct {
	Token   string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error getting API key: %w", err)
	}

	token := strings.TrimSpace(string(apiKey))
	if token == "" {
		return nil, fmt.Errorf("api key is required")
	}

	return &Client{
		Token:   token,
		BaseURL: BaseURL,
		http:    http,
	}, nil
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.Token)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return responseBody, nil
}

// ListProjects returns all projects in the LaunchDarkly account.
func (c *Client) ListProjects() ([]Project, error) {
	const limit = 200
	var all []Project
	for offset := 0; ; offset += limit {
		path := fmt.Sprintf("/api/v2/projects?limit=%d&offset=%d", limit, offset)
		responseBody, err := c.execRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var response ProjectListResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing projects response: %w", err)
		}

		all = append(all, response.Items...)
		if len(response.Items) == 0 || len(all) >= response.TotalCount {
			break
		}
	}

	return all, nil
}

// GetFeatureFlag returns a feature flag by project key and flag key.
func (c *Client) GetFeatureFlag(projectKey, flagKey string) (map[string]any, error) {
	path := fmt.Sprintf("/api/v2/flags/%s/%s", projectKey, flagKey)
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing feature flag response: %w", err)
	}

	return result, nil
}

// ListFeatureFlags returns all feature flags in a LaunchDarkly project.
func (c *Client) ListFeatureFlags(projectKey string) ([]FeatureFlag, error) {
	const limit = 200
	var all []FeatureFlag
	for offset := 0; ; offset += limit {
		path := fmt.Sprintf("/api/v2/flags/%s?limit=%d&offset=%d", projectKey, limit, offset)
		responseBody, err := c.execRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var response FeatureFlagListResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing feature flags response: %w", err)
		}

		all = append(all, response.Items...)
		if len(response.Items) == 0 || len(all) >= response.TotalCount {
			break
		}
	}

	return all, nil
}

// ListEnvironments returns all environments in a LaunchDarkly project.
func (c *Client) ListEnvironments(projectKey string) ([]Environment, error) {
	const limit = 200
	var all []Environment
	for offset := 0; ; offset += limit {
		path := fmt.Sprintf("/api/v2/projects/%s/environments?limit=%d&offset=%d", projectKey, limit, offset)
		responseBody, err := c.execRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var response EnvironmentListResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing environments response: %w", err)
		}

		all = append(all, response.Items...)
		if len(response.Items) == 0 || len(all) >= response.TotalCount {
			break
		}
	}

	return all, nil
}

// DeleteFeatureFlag deletes a feature flag by project key and flag key.
func (c *Client) DeleteFeatureFlag(projectKey, flagKey string) error {
	path := fmt.Sprintf("/api/v2/flags/%s/%s", projectKey, flagKey)
	_, err := c.execRequest(http.MethodDelete, path, nil)
	return err
}

// WebhookStatement is a policy statement that filters which resource/action combinations
// the webhook responds to.
type WebhookStatement struct {
	Effect    string   `json:"effect"`
	Resources []string `json:"resources,omitempty"`
	Actions   []string `json:"actions,omitempty"`
}

// CreateWebhookRequest is the request body for creating a LaunchDarkly webhook.
type CreateWebhookRequest struct {
	URL        string             `json:"url"`
	Sign       bool               `json:"sign"`
	On         bool               `json:"on"`
	Name       string             `json:"name,omitempty"`
	Statements []WebhookStatement `json:"statements,omitempty"`
}

// LDWebhook is the response from creating a webhook. The _id field is the webhook ID
// needed later for deletion.
type LDWebhook struct {
	ID     string `json:"_id"`
	Secret string `json:"secret"`
}

// CreateWebhook creates a new signed webhook in LaunchDarkly. LaunchDarkly auto-generates
// the signing secret if one is not provided in the request.
func (c *Client) CreateWebhook(req CreateWebhookRequest) (*LDWebhook, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error encoding request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, "/api/v2/webhooks", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var result LDWebhook
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing webhook response: %w", err)
	}

	return &result, nil
}

// DeleteWebhook deletes a webhook from LaunchDarkly by its ID.
func (c *Client) DeleteWebhook(id string) error {
	_, err := c.execRequest(http.MethodDelete, "/api/v2/webhooks/"+id, nil)
	return err
}
