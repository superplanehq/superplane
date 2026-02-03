package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	Email   string
	Token   string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	baseURL, err := ctx.GetConfig("baseUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting baseUrl: %v", err)
	}

	email, err := ctx.GetConfig("email")
	if err != nil {
		return nil, fmt.Errorf("error getting email: %v", err)
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting apiToken: %v", err)
	}

	return &Client{
		Email:   string(email),
		Token:   string(apiToken),
		BaseURL: string(baseURL),
		http:    httpCtx,
	}, nil
}

func (c *Client) authHeader() string {
	credentials := fmt.Sprintf("%s:%s", c.Email, c.Token)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(credentials)))
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// User represents a Jira user from the /myself endpoint.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	EmailAddr   string `json:"emailAddress"`
}

// GetCurrentUser verifies credentials by fetching the authenticated user.
func (c *Client) GetCurrentUser() (*User, error) {
	url := fmt.Sprintf("%s/rest/api/3/myself", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(responseBody, &user); err != nil {
		return nil, fmt.Errorf("error parsing user response: %v", err)
	}

	return &user, nil
}

// Project represents a Jira project.
type Project struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ListProjects returns all projects accessible to the authenticated user.
func (c *Client) ListProjects() ([]Project, error) {
	url := fmt.Sprintf("%s/rest/api/3/project", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(responseBody, &projects); err != nil {
		return nil, fmt.Errorf("error parsing projects response: %v", err)
	}

	return projects, nil
}
