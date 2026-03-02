package loki

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

const MaxResponseSize = 1 * 1024 * 1024 // 1MB

type Client struct {
	baseURL     string
	authType    string
	username    string
	password    string
	bearerToken string
	tenantID    string
	http        core.HTTPContext
}

type PushRequest struct {
	Streams []Stream `json:"streams"`
}

type Stream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type QueryResponse struct {
	Status string    `json:"status"`
	Data   QueryData `json:"data"`
}

type QueryData struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

func NewClient(httpContext core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	baseURL, err := requiredConfig(integration, "baseURL")
	if err != nil {
		return nil, err
	}

	authType, err := requiredConfig(integration, "authType")
	if err != nil {
		return nil, err
	}

	client := &Client{
		baseURL:  normalizeBaseURL(baseURL),
		authType: authType,
		tenantID: optionalConfig(integration, "tenantID"),
		http:     httpContext,
	}

	switch authType {
	case AuthTypeNone:
		return client, nil
	case AuthTypeBasic:
		username, err := requiredConfig(integration, "username")
		if err != nil {
			return nil, fmt.Errorf("username is required when authentication is basic")
		}
		password, err := requiredConfig(integration, "password")
		if err != nil {
			return nil, fmt.Errorf("password is required when authentication is basic")
		}

		client.username = username
		client.password = password
		return client, nil
	case AuthTypeBearer:
		bearerToken, err := requiredConfig(integration, "bearerToken")
		if err != nil {
			return nil, fmt.Errorf("bearerToken is required when authentication is bearer")
		}

		client.bearerToken = bearerToken
		return client, nil
	default:
		return nil, fmt.Errorf("invalid authType %q", authType)
	}
}

func (c *Client) Ready() error {
	_, err := c.execRequest(http.MethodGet, "/ready", nil)
	return err
}

func (c *Client) Push(streams []Stream) error {
	payload := PushRequest{Streams: streams}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push request: %w", err)
	}

	_, err = c.execRequest(http.MethodPost, "/loki/api/v1/push", bytes.NewReader(body))
	return err
}

func (c *Client) QueryRange(query, start, end, limit string) (*QueryData, error) {
	params := url.Values{}
	params.Set("query", query)

	if start != "" {
		params.Set("start", start)
	}
	if end != "" {
		params.Set("end", end)
	}
	if limit != "" {
		params.Set("limit", limit)
	}

	path := "/loki/api/v1/query_range?" + params.Encode()

	body, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response QueryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("Loki query returned status: %s", response.Status)
	}

	return &response.Data, nil
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	apiURL := c.baseURL
	if strings.HasPrefix(path, "/") {
		apiURL += path
	} else {
		apiURL += "/" + path
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.tenantID != "" {
		req.Header.Set("X-Scope-OrgID", c.tenantID)
	}

	if err := c.setAuth(req); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	limitedReader := io.LimitReader(res.Body, MaxResponseSize+1)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(responseBody) > MaxResponseSize {
		return nil, fmt.Errorf("response too large: exceeds maximum size of %d bytes", MaxResponseSize)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

func (c *Client) setAuth(req *http.Request) error {
	switch c.authType {
	case AuthTypeNone:
		return nil
	case AuthTypeBasic:
		req.SetBasicAuth(c.username, c.password)
		return nil
	case AuthTypeBearer:
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		return nil
	default:
		return fmt.Errorf("invalid authType %q", c.authType)
	}
}

func optionalConfig(ctx core.IntegrationContext, name string) string {
	value, err := ctx.GetConfig(name)
	if err != nil {
		return ""
	}
	return string(value)
}

func requiredConfig(ctx core.IntegrationContext, name string) (string, error) {
	value, err := ctx.GetConfig(name)
	if err != nil {
		return "", fmt.Errorf("%s is required", name)
	}

	s := string(value)
	if s == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return s, nil
}

func normalizeBaseURL(baseURL string) string {
	if baseURL == "/" {
		return baseURL
	}

	for len(baseURL) > 0 && strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL[:len(baseURL)-1]
	}

	return baseURL
}
