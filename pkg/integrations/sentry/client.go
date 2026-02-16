package sentry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	Token   string
	BaseURL string
	http    core.HTTPContext
	OrgSlug string
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	authToken, err := ctx.GetConfig("authToken")
	if err != nil {
		return nil, fmt.Errorf("error finding auth token: %v", err)
	}

	orgSlug, err := ctx.GetConfig("orgSlug")
	if err != nil {
		return nil, fmt.Errorf("error finding organization slug: %v", err)
	}

	baseURL, err := ctx.GetConfig("baseUrl")
	if err != nil || baseURL == nil {
		// Default to sentry.io
		baseURL = []byte("https://sentry.io/api/0")
	} else {
		// Ensure URL has /api/0 suffix
		if !bytes.HasSuffix(baseURL, []byte("/api/0")) {
			baseURL = []byte(fmt.Sprintf("%s/api/0", string(baseURL)))
		}
	}

	return &Client{
		Token:   string(authToken),
		BaseURL: string(baseURL),
		OrgSlug: string(orgSlug),
		http:    http,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

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

// CreateSentryApp creates a new internal Sentry integration (Sentry App)
func (c *Client) CreateSentryApp(name, webhookURL string, events []SentryAppEvent) (*SentryApp, error) {
	apiURL := fmt.Sprintf("%s/sentry-apps/", c.BaseURL)

	app := SentryAppCreateRequest{
		Name:         name,
		IsInternal:   true,
		Organization: c.OrgSlug,
		Scopes:       []string{"org:read", "event:read", "event:write", "event:admin"},
		WebhookURL:   webhookURL,
		Events:       events,
	}

	body, err := json.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var sentryApp SentryApp
	err = json.Unmarshal(responseBody, &sentryApp)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &sentryApp, nil
}

// DeleteSentryApp deletes a Sentry App by its slug
func (c *Client) DeleteSentryApp(slug string) error {
	apiURL := fmt.Sprintf("%s/sentry-apps/%s/", c.BaseURL, slug)
	_, err := c.execRequest(http.MethodDelete, apiURL, nil)
	return err
}

// UpdateIssue updates a Sentry issue (org-scoped per Sentry API)
func (c *Client) UpdateIssue(issueID string, update IssueUpdateRequest) (any, error) {
	apiURL := fmt.Sprintf("%s/organizations/%s/issues/%s/", c.BaseURL, c.OrgSlug, issueID)

	body, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// SentryApp represents a Sentry Internal Integration
type SentryApp struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	ClientSecret string `json:"clientSecret"`
	UUID         string `json:"uuid"`
}

type SentryAppCreateRequest struct {
	Name         string           `json:"name"`
	IsInternal   bool             `json:"isInternal"`
	Organization string           `json:"organization"`
	Scopes       []string         `json:"scopes"`
	WebhookURL   string           `json:"webhookUrl"`
	Events       []SentryAppEvent `json:"events"`
}

type SentryAppEvent struct {
	Type string `json:"type"`
}

// IssueUpdateRequest represents the request to update an issue
type IssueUpdateRequest struct {
	Status     *string `json:"status,omitempty"`
	AssignedTo *string `json:"assignedTo,omitempty"`
}
