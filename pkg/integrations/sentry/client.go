package sentry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://sentry.io"

type Client struct {
	BaseURL string
	Token   string
	http    core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	token, err := ctx.GetConfig("authToken")
	if err != nil {
		return nil, fmt.Errorf("sentry auth token not found: %w", err)
	}
	baseURL := defaultBaseURL
	if u, err := ctx.GetConfig("baseURL"); err == nil && len(u) > 0 {
		baseURL = string(u)
	}

	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		BaseURL: baseURL,
		Token:   strings.TrimSpace(string(token)),
		http:    http,
	}, nil
}

func (c *Client) do(method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}
	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("sentry API %s %s: %d %s", method, path, res.StatusCode, string(resBody))
	}
	return resBody, nil
}

func (c *Client) ValidateToken() error {
	_, err := c.do(http.MethodGet, "/api/0/organizations/", nil)
	return err
}

type UpdateIssueRequest struct {
	Status        string         `json:"status,omitempty"`
	StatusDetails map[string]any `json:"statusDetails,omitempty"`
	AssignedTo    string         `json:"assignedTo,omitempty"`
	HasSeen       *bool          `json:"hasSeen,omitempty"`
	IsBookmarked  *bool          `json:"isBookmarked,omitempty"`
	IsSubscribed  *bool          `json:"isSubscribed,omitempty"`
	IsPublic      *bool          `json:"isPublic,omitempty"`
}

func (c *Client) UpdateIssue(org, issueID string, req UpdateIssueRequest) (map[string]any, error) {
	if org == "" || issueID == "" {
		return nil, fmt.Errorf("organization and issue id are required")
	}
	path := fmt.Sprintf("/api/0/organizations/%s/issues/%s/", org, issueID)
	// Sentry API: https://docs.sentry.io/api/events/update-an-issue/
	body, err := c.do(http.MethodPut, path, req)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return out, nil
}
