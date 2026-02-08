package circleci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://circleci.com/api/v2"

type Client struct {
	BaseURL  string
	APIToken string
	http     core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	if integration == nil {
		return nil, fmt.Errorf("no integration context")
	}

	token, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}
	if len(token) == 0 {
		return nil, fmt.Errorf("apiToken is required")
	}

	return &Client{
		BaseURL:  defaultBaseURL,
		APIToken: string(token),
		http:     httpCtx,
	}, nil
}

func (c *Client) exec(method, path string, body any, expectedStatus ...int) ([]byte, error) {
	full := strings.TrimRight(c.BaseURL, "/") + path

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, full, r)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Circle-Token", c.APIToken)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	ok := false
	if len(expectedStatus) == 0 {
		ok = res.StatusCode >= 200 && res.StatusCode < 300
	} else {
		for _, s := range expectedStatus {
			if res.StatusCode == s {
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("circleci request got %d: %s", res.StatusCode, string(b))
	}

	return b, nil
}

type CurrentUser struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

func (c *Client) GetCurrentUser() (*CurrentUser, error) {
	b, err := c.exec(http.MethodGet, "/me", nil, http.StatusOK)
	if err != nil {
		return nil, err
	}

	var u CurrentUser
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, fmt.Errorf("failed to parse /me response: %w", err)
	}
	return &u, nil
}

type Project struct {
	ID               string `json:"id"`
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	OrganizationName string `json:"organization_name"`
	OrganizationID   string `json:"organization_id"`
}

func encodeProjectSlug(slug string) string {
	// CircleCI allows reserved "/" characters in the slug.
	escaped := url.PathEscape(slug)
	return strings.ReplaceAll(escaped, "%2F", "/")
}

func (c *Client) GetProjectBySlug(projectSlug string) (*Project, error) {
	path := "/project/" + encodeProjectSlug(projectSlug)
	b, err := c.exec(http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}

	var p Project
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("failed to parse project response: %w", err)
	}
	return &p, nil
}

type TriggerPipelineRequest struct {
	Branch     string         `json:"branch,omitempty"`
	Tag        string         `json:"tag,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type PipelineCreation struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Number    int64  `json:"number"`
	CreatedAt string `json:"created_at"`
}

func (c *Client) TriggerPipeline(projectSlug string, req TriggerPipelineRequest) (*PipelineCreation, error) {
	path := "/project/" + encodeProjectSlug(projectSlug) + "/pipeline"
	b, err := c.exec(http.MethodPost, path, req, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	var out PipelineCreation
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("failed to parse trigger pipeline response: %w", err)
	}
	return &out, nil
}

type WebhookScope struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type CreateWebhookRequest struct {
	Name          string       `json:"name"`
	Events        []string     `json:"events"`
	URL           string       `json:"url"`
	VerifyTLS     bool         `json:"verify-tls"`
	SigningSecret string       `json:"signing-secret"`
	Scope         WebhookScope `json:"scope"`
}

type Webhook struct {
	ID string `json:"id"`
}

func (c *Client) CreateWebhook(req CreateWebhookRequest) (*Webhook, error) {
	b, err := c.exec(http.MethodPost, "/webhook", req, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	var out Webhook
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("failed to parse create webhook response: %w", err)
	}
	return &out, nil
}

func (c *Client) DeleteWebhook(webhookID string) error {
	webhookID = url.PathEscape(webhookID)
	_, err := c.exec(http.MethodDelete, "/webhook/"+webhookID, nil, http.StatusOK)
	return err
}
