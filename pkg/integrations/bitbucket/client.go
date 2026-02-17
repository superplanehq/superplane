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
