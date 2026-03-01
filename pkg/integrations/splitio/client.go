package splitio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const BaseURL = "https://api.split.io/internal/api/v2"

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Environment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Split struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type SplitListResponse struct {
	Objects []Split `json:"objects"`
	Offset  int     `json:"offset"`
	Limit   int     `json:"limit"`
	Total   int     `json:"totalCount"`
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
		return nil, fmt.Errorf("API key is required")
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
	req.Header.Set("Authorization", "Bearer "+c.Token)

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

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	const limit = 200
	var all []Workspace
	for offset := 0; ; offset += limit {
		path := fmt.Sprintf("/workspaces?limit=%d&offset=%d", limit, offset)
		responseBody, err := c.execRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var response struct {
			Objects []Workspace `json:"objects"`
			Total   int         `json:"totalCount"`
		}
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing workspaces response: %w", err)
		}

		all = append(all, response.Objects...)
		if len(response.Objects) == 0 || len(all) >= response.Total {
			break
		}
	}

	return all, nil
}

func (c *Client) ListEnvironments(workspaceID string) ([]Environment, error) {
	path := fmt.Sprintf("/environments/ws/%s", workspaceID)
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var environments []Environment
	if err := json.Unmarshal(responseBody, &environments); err != nil {
		return nil, fmt.Errorf("error parsing environments response: %w", err)
	}

	return environments, nil
}

func (c *Client) ListSplits(workspaceID string) ([]Split, error) {
	const limit = 200
	var all []Split
	for offset := 0; ; offset += limit {
		path := fmt.Sprintf("/splits/ws/%s?limit=%d&offset=%d", workspaceID, limit, offset)
		responseBody, err := c.execRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var response SplitListResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing splits response: %w", err)
		}

		all = append(all, response.Objects...)
		if len(response.Objects) == 0 || len(all) >= response.Total {
			break
		}
	}

	return all, nil
}

func (c *Client) GetSplitDefinition(workspaceID, splitName, environmentID string) (map[string]any, error) {
	path := fmt.Sprintf("/splits/ws/%s/%s/environments/%s", workspaceID, splitName, environmentID)
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing split definition response: %w", err)
	}

	return result, nil
}
