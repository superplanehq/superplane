package snyk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	BaseURL = "https://api.snyk.io"
	Version = "2024-06-10"
)

type Client struct {
	httpClient  core.HTTPContext
	integration core.IntegrationContext
	baseURL     string
	version     string
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	apiToken, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting API token: %v", err)
	}

	if len(apiToken) == 0 || string(apiToken) == "" {
		return nil, fmt.Errorf("apiToken is required")
	}

	return &Client{
		httpClient:  httpCtx,
		integration: integration,
		baseURL:     BaseURL,
		version:     Version,
	}, nil
}

type UserResponse struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Name              string `json:"name"`
			Username          string `json:"username"`
			Email             string `json:"email"`
			DefaultOrgContext string `json:"default_org_context"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetUser() (*UserResponse, error) {
	url := fmt.Sprintf("%s/rest/self?version=%s", c.baseURL, c.version)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	apiToken, err := c.integration.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", string(apiToken)))
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userResp UserResponse
	err = json.Unmarshal(body, &userResp)
	if err != nil {
		return nil, err
	}

	return &userResp, nil
}

type SnykProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListProjectsResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) ListProjects(orgID string) ([]SnykProject, error) {
	url := fmt.Sprintf("%s/rest/orgs/%s/projects?version=%s&limit=100", c.baseURL, orgID, c.version)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	apiToken, err := c.integration.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", string(apiToken)))
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var projectsResp ListProjectsResponse
	if err := json.Unmarshal(body, &projectsResp); err != nil {
		return nil, err
	}

	projects := make([]SnykProject, 0, len(projectsResp.Data))
	for _, p := range projectsResp.Data {
		projects = append(projects, SnykProject{
			ID:   p.ID,
			Name: p.Attributes.Name,
		})
	}

	return projects, nil
}

type IgnoreIssueRequest struct {
	Reason             string `json:"reason"`
	ReasonType         string `json:"reasonType,omitempty"`
	DisregardIfFixable bool   `json:"disregardIfFixable"`
	Expires            string `json:"expires,omitempty"`
}

type IgnoreIssueResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (c *Client) IgnoreIssue(orgID, projectID, issueID string, req IgnoreIssueRequest) (*IgnoreIssueResponse, error) {
	url := fmt.Sprintf("%s/v1/org/%s/project/%s/ignore/%s", c.baseURL, orgID, projectID, issueID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	apiToken, err := c.integration.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("token %s", string(apiToken)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Snyk's ignore API might not return JSON, so we handle both cases
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &IgnoreIssueResponse{
			Success: true,
			Message: fmt.Sprintf("Issue %s ignored successfully", issueID),
		}, nil
	}

	return &IgnoreIssueResponse{
		Success: false,
		Message: fmt.Sprintf("Failed to ignore issue %s: %s", issueID, string(body)),
	}, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
}

func (c *Client) RegisterWebhook(orgID, url, secret string) (string, error) {
	registerURL := fmt.Sprintf("%s/v1/org/%s/webhooks", c.baseURL, orgID)

	requestBody := map[string]any{
		"url":    url,
		"secret": secret,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest("POST", registerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	apiToken, err := c.integration.GetConfig("apiToken")
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("token %s", string(apiToken)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("webhook registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse webhook registration response: %w", err)
	}

	if id, ok := response["id"]; ok {
		if idStr, isString := id.(string); isString {
			return idStr, nil
		}
		return "", fmt.Errorf("webhook ID in response is not a string")
	}

	return "", fmt.Errorf("webhook ID not found in response")
}

func (c *Client) DeleteWebhook(orgID, webhookID string) error {
	deleteURL := fmt.Sprintf("%s/v1/org/%s/webhooks/%s", c.baseURL, orgID, webhookID)

	httpReq, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return err
	}

	apiToken, err := c.integration.GetConfig("apiToken")
	if err != nil {
		return err
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("token %s", string(apiToken)))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("webhook deletion failed with status %d: %s", resp.StatusCode, string(body))
}
