package sentry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	httpContext core.HTTPContext
	baseURL     string
	accessToken string
	orgSlug     string
}

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("sentry API returned status %d: %s", e.StatusCode, e.Body)
}

func NewClient(httpContext core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	baseURL, err := integration.GetConfig("baseUrl")
	if err != nil {
		return nil, fmt.Errorf("failed to get sentry base URL: %w", err)
	}

	accessToken, err := findSecret(integration, OAuthAccessTokenSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get sentry access token: %w", err)
	}

	if accessToken == "" {
		return nil, fmt.Errorf("Sentry access token is missing")
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode sentry metadata: %w", err)
	}

	if metadata.Organization == nil || metadata.Organization.Slug == "" {
		return nil, fmt.Errorf("Sentry organization is not connected")
	}

	return &Client{
		httpContext: httpContext,
		baseURL:     normalizeBaseURL(string(baseURL)),
		accessToken: accessToken,
		orgSlug:     metadata.Organization.Slug,
	}, nil
}

func NewAPIClient(httpContext core.HTTPContext, baseURL, accessToken string) *Client {
	return &Client{
		httpContext: httpContext,
		baseURL:     normalizeBaseURL(baseURL),
		accessToken: accessToken,
	}
}

type Organization struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type Project struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type Team struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type UpdateIssueRequest struct {
	Status     string `json:"status,omitempty"`
	AssignedTo string `json:"assignedTo,omitempty"`
}

func (c *Client) ListOrganizations() ([]Organization, error) {
	responseBody, err := c.doJSON(http.MethodGet, "/api/0/organizations/", nil)
	if err != nil {
		return nil, err
	}

	organizations := []Organization{}
	if err := json.Unmarshal(responseBody, &organizations); err != nil {
		return nil, err
	}

	return organizations, nil
}

func (c *Client) GetOrganization() (*Organization, error) {
	responseBody, err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/0/organizations/%s/", c.orgSlug), nil)
	if err != nil {
		return nil, err
	}

	organization := Organization{}
	if err := json.Unmarshal(responseBody, &organization); err != nil {
		return nil, err
	}

	return &organization, nil
}

func (c *Client) ListProjects() ([]ProjectSummary, error) {
	responseBody, err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/0/organizations/%s/projects/", c.orgSlug), nil)
	if err != nil {
		return nil, err
	}

	projects := []Project{}
	if err := json.Unmarshal(responseBody, &projects); err != nil {
		return nil, err
	}

	result := make([]ProjectSummary, 0, len(projects))
	for _, project := range projects {
		result = append(result, ProjectSummary{
			ID:   project.ID,
			Slug: project.Slug,
			Name: project.Name,
		})
	}

	return result, nil
}

func (c *Client) ListTeams() ([]TeamSummary, error) {
	responseBody, err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/0/organizations/%s/teams/", c.orgSlug), nil)
	if err != nil {
		return nil, err
	}

	teams := []Team{}
	if err := json.Unmarshal(responseBody, &teams); err != nil {
		return nil, err
	}

	result := make([]TeamSummary, 0, len(teams))
	for _, team := range teams {
		result = append(result, TeamSummary{
			ID:   team.ID,
			Slug: team.Slug,
			Name: team.Name,
		})
	}

	return result, nil
}

func (c *Client) UpdateIssue(issueID string, request UpdateIssueRequest) (map[string]any, error) {
	responseBody, err := c.doJSON(
		http.MethodPut,
		fmt.Sprintf("/api/0/organizations/%s/issues/%s/", c.orgSlug, issueID),
		request,
	)
	if err != nil {
		return nil, err
	}

	result := map[string]any{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) doJSON(method, path string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpContext.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &apiError{StatusCode: resp.StatusCode, Body: string(responseBody)}
	}

	return responseBody, nil
}
