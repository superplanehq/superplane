package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.bitbucket.org/2.0"

type Client struct {
	AuthType string
	Email    string
	Token    string
	HTTP     core.HTTPContext
}

type RepositoryResponse struct {
	Values []Repository `json:"values"`
	Next   string       `json:"next"`
}

type Repository struct {
	UUID     string         `json:"uuid" mapstructure:"uuid"`
	Name     string         `json:"name" mapstructure:"name"`
	FullName string         `json:"full_name" mapstructure:"full_name"`
	Slug     string         `json:"slug" mapstructure:"slug"`
	Links    RepositoryLink `json:"links" mapstructure:"links"`
}

type RepositoryLink struct {
	HTML struct {
		Href string `json:"href" mapstructure:"href"`
	} `json:"html" mapstructure:"html"`
}

func NewClient(authType string, httpContext core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	switch authType {
	case AuthTypeAPIToken:
		token, err := integration.GetConfig("token")
		if err != nil {
			return nil, fmt.Errorf("error getting token config: %w", err)
		}

		email, err := integration.GetConfig("email")
		if err != nil {
			return nil, fmt.Errorf("error getting email config: %w", err)
		}

		return &Client{
			AuthType: AuthTypeAPIToken,
			Email:    string(email),
			Token:    string(token),
			HTTP:     httpContext,
		}, nil

	case AuthTypeWorkspaceAccessToken:
		token, err := integration.GetConfig("token")
		if err != nil {
			return nil, fmt.Errorf("error getting token config: %w", err)
		}
		return &Client{
			AuthType: AuthTypeWorkspaceAccessToken,
			Token:    string(token),
			HTTP:     httpContext,
		}, nil
	}

	return nil, fmt.Errorf("unknown auth type %s", authType)
}

func (c *Client) setAuthHeaders(req *http.Request) {
	if c.AuthType == AuthTypeAPIToken {
		req.SetBasicAuth(c.Email, c.Token)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func (c *Client) doJSONRequest(method, url string, payload any, expectedStatusCodes ...int) ([]byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		requestBody, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(requestBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if !statusCodeAllowed(resp.StatusCode, expectedStatusCodes) {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

func statusCodeAllowed(statusCode int, expectedStatusCodes []int) bool {
	for _, expectedStatusCode := range expectedStatusCodes {
		if statusCode == expectedStatusCode {
			return true
		}
	}

	return false
}

type Workspace struct {
	UUID string `json:"uuid" mapstructure:"uuid"`
	Name string `json:"name" mapstructure:"name"`
	Slug string `json:"slug" mapstructure:"slug"`
}

func (c *Client) GetWorkspace(workspaceSlug string) (*Workspace, error) {
	url := fmt.Sprintf("%s/workspaces/%s", baseURL, workspaceSlug)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var workspace Workspace
	err = json.Unmarshal(body, &workspace)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &workspace, nil
}

func (c *Client) ListRepositories(workspace string) ([]Repository, error) {
	url := fmt.Sprintf("%s/repositories/%s?pagelen=100", baseURL, workspace)
	repositories := []Repository{}

	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		c.setAuthHeaders(req)
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error executing request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}

		var repoResponse RepositoryResponse
		err = json.Unmarshal(body, &repoResponse)
		if err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}

		repositories = append(repositories, repoResponse.Values...)
		url = repoResponse.Next
	}

	return repositories, nil
}

func (c *Client) GetIssue(workspace, repositorySlug string, issueNumber int) (map[string]any, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d", baseURL, workspace, repositorySlug, issueNumber)

	responseBody, err := c.doJSONRequest(http.MethodGet, url, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}

	issue := map[string]any{}
	if err := json.Unmarshal(responseBody, &issue); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return issue, nil
}

func (c *Client) CreateIssue(workspace, repositorySlug string, issue map[string]any) (map[string]any, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/issues", baseURL, workspace, repositorySlug)

	responseBody, err := c.doJSONRequest(http.MethodPost, url, issue, http.StatusCreated, http.StatusOK)
	if err != nil {
		return nil, err
	}

	createdIssue := map[string]any{}
	if err := json.Unmarshal(responseBody, &createdIssue); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return createdIssue, nil
}

func (c *Client) UpdateIssue(
	workspace,
	repositorySlug string,
	issueNumber int,
	issue map[string]any,
) (map[string]any, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d", baseURL, workspace, repositorySlug, issueNumber)

	responseBody, err := c.doJSONRequest(http.MethodPut, url, issue, http.StatusOK)
	if err != nil {
		return nil, err
	}

	updatedIssue := map[string]any{}
	if err := json.Unmarshal(responseBody, &updatedIssue); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return updatedIssue, nil
}

func (c *Client) CreateIssueComment(
	workspace,
	repositorySlug string,
	issueNumber int,
	commentBody string,
) (map[string]any, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d/comments", baseURL, workspace, repositorySlug, issueNumber)

	request := map[string]any{
		"content": map[string]any{
			"raw": commentBody,
		},
	}

	responseBody, err := c.doJSONRequest(http.MethodPost, url, request, http.StatusCreated, http.StatusOK)
	if err != nil {
		return nil, err
	}

	comment := map[string]any{}
	if err := json.Unmarshal(responseBody, &comment); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return comment, nil
}

type BitbucketHookRequest struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Secret      string   `json:"secret,omitempty"`
	Events      []string `json:"events"`
}

type BitbucketHookResponse struct {
	UUID   string `json:"uuid"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

func (c *Client) CreateWebhook(workspace, repoSlug, webhookURL, secret string, events []string) (*BitbucketHookResponse, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/hooks", baseURL, workspace, repoSlug)

	hookReq := BitbucketHookRequest{
		Description: "SuperPlane",
		URL:         webhookURL,
		Active:      true,
		Secret:      secret,
		Events:      events,
	}

	body, err := json.Marshal(hookReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling webhook request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	var hookResp BitbucketHookResponse
	err = json.Unmarshal(respBody, &hookResp)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &hookResp, nil
}

func (c *Client) DeleteWebhook(workspace, repoSlug, webhookUID string) error {
	url := fmt.Sprintf("%s/repositories/%s/%s/hooks/%s", baseURL, workspace, repoSlug, webhookUID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
