package buildkite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const BaseURL = "https://api.buildkite.com/v2"

type Client struct {
	http     core.HTTPContext
	apiToken string
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	return &Client{
		http:     httpCtx,
		apiToken: string(apiToken),
	}, nil
}

func (c *Client) makeRequest(method, endpoint string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, BaseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	return c.http.Do(req)
}

type Pipeline struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	WebURL string `json:"web_url"`
}

func (c *Client) ListPipelines(orgSlug string) ([]Pipeline, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/pipelines", orgSlug), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var pipelines []Pipeline
	err = json.NewDecoder(resp.Body).Decode(&pipelines)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return pipelines, nil
}

type CreateBuildRequest struct {
	Commit   string            `json:"commit"`
	Branch   string            `json:"branch"`
	Message  string            `json:"message,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Metadata map[string]string `json:"meta_data,omitempty"`
}

type Build struct {
	ID         string `json:"id"`
	Number     int    `json:"number"`
	State      string `json:"state"`
	WebURL     string `json:"web_url"`
	Commit     string `json:"commit"`
	Branch     string `json:"branch"`
	Message    string `json:"message"`
	Blocked    bool   `json:"blocked"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
}

func (c *Client) CreateBuild(orgSlug, pipelineSlug string, req CreateBuildRequest) (*Build, error) {
	resp, err := c.makeRequest("POST", fmt.Sprintf("/organizations/%s/pipelines/%s/builds", orgSlug, pipelineSlug), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var build Build
	err = json.NewDecoder(resp.Body).Decode(&build)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &build, nil
}

func (c *Client) GetBuild(orgSlug, pipelineSlug string, buildNumber int) (*Build, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/pipelines/%s/builds/%d", orgSlug, pipelineSlug, buildNumber), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var build Build
	err = json.NewDecoder(resp.Body).Decode(&build)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &build, nil
}

type AccessToken struct {
	UUID        string   `json:"uuid"`
	Scopes      []string `json:"scopes"`
	Description string   `json:"description"`
	CreatedAt   string   `json:"created_at"`
	User        struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
}

func (c *Client) ValidateToken() (*AccessToken, error) {
	resp, err := c.makeRequest("GET", "/access-token", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API token validation failed with status %d", resp.StatusCode)
	}

	var token AccessToken
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode access token response: %w", err)
	}

	return &token, nil
}
