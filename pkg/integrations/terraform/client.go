package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

type Client struct {
	HTTPClient *http.Client
	Token      string
	BaseURL    string
}

func NewClient(configuration map[string]any) (*Client, error) {
	var config Configuration
	err := mapstructure.Decode(configuration, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	if config.APIToken == "" {
		return nil, fmt.Errorf("apiToken is required")
	}

	address := config.Address
	if address == "" {
		address = "https://app.terraform.io"
	}

	return &Client{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Token:      config.APIToken,
		BaseURL:    address,
	}, nil
}

func (c *Client) newRequest(method, path string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	baseURL := strings.TrimRight(c.BaseURL, "/")
	reqPath := path
	if !strings.HasPrefix(reqPath, "/") {
		reqPath = "/" + reqPath
	}
	reqURL := baseURL + reqPath

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	return req, nil
}

func (c *Client) readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bad status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) ResolveWorkspaceID(identifier string) (string, error) {
	if strings.HasPrefix(identifier, "ws-") {
		return identifier, nil
	}
	parts := strings.Split(identifier, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid workspace identifier format. Expected 'ws-xxx' or 'org_name/workspace_name'")
	}

	orgName := parts[0]
	wsName := parts[1]

	path := fmt.Sprintf("/api/v2/organizations/%s/workspaces/%s", url.PathEscape(orgName), url.PathEscape(wsName))
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create resolve workspace request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to lookup workspace %s: %w", identifier, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to lookup workspace %s: expected 200 OK, got %d", identifier, resp.StatusCode)
	}

	var payload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode workspace response: %w", err)
	}

	if payload.Data.ID == "" {
		return "", fmt.Errorf("workspace ID missing from response")
	}

	return payload.Data.ID, nil
}
