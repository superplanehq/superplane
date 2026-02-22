package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"golang.org/x/oauth2/google"
)

const defaultComputeBaseURL = "https://compute.googleapis.com/compute/v1"

type Client struct {
	creds     *google.Credentials
	http      core.HTTPContext
	projectID string
	baseURL   string
}

func NewClient(httpClient core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	if integration == nil {
		return nil, fmt.Errorf("integration context is required")
	}

	creds, err := CredentialsFromIntegration(integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP credentials: %w", err)
	}

	meta := integration.GetMetadata()
	if meta == nil {
		return nil, fmt.Errorf("integration metadata missing")
	}
	var m Metadata
	if err := mapstructure.Decode(meta, &m); err != nil {
		return nil, fmt.Errorf("invalid integration metadata: %w", err)
	}
	projectID := strings.TrimSpace(m.ProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("integration metadata has no project ID")
	}

	return &Client{
		creds:     creds,
		http:      httpClient,
		projectID: projectID,
		baseURL:   defaultComputeBaseURL,
	}, nil
}

func (c *Client) ProjectID() string {
	return c.projectID
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) ExecRequest(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	token, err := c.creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, ParseGCPError(res.StatusCode, responseBody)
	}
	return responseBody, nil
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	path = strings.TrimPrefix(path, "/")
	url := strings.TrimSuffix(c.baseURL, "/") + "/" + path
	return c.ExecRequest(ctx, http.MethodGet, url, nil)
}

func (c *Client) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	return c.ExecRequest(ctx, http.MethodGet, fullURL, nil)
}

func (c *Client) Post(ctx context.Context, path string, body any) ([]byte, error) {
	path = strings.TrimPrefix(path, "/")
	url := strings.TrimSuffix(c.baseURL, "/") + "/" + path
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	return c.ExecRequest(ctx, http.MethodPost, url, bodyReader)
}
