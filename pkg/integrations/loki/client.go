package loki

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL  string
	AuthType string
	Username string
	Password string
	Token    string
	TenantID string
	http     core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	urlBytes, err := ctx.GetConfig("url")
	if err != nil {
		return nil, fmt.Errorf("error getting url: %v", err)
	}

	url := strings.TrimRight(string(urlBytes), "/")

	authTypeBytes, _ := ctx.GetConfig("authType")
	authType := string(authTypeBytes)

	var username, password, token, tenantID string

	if authType == "basic" {
		usernameBytes, _ := ctx.GetConfig("username")
		username = string(usernameBytes)

		passwordBytes, _ := ctx.GetConfig("password")
		password = string(passwordBytes)
	}

	if authType == "bearer" {
		tokenBytes, _ := ctx.GetConfig("token")
		token = string(tokenBytes)
	}

	tenantIDBytes, _ := ctx.GetConfig("tenantId")
	tenantID = string(tenantIDBytes)

	return &Client{
		BaseURL:  url,
		AuthType: authType,
		Username: username,
		Password: password,
		Token:    token,
		TenantID: tenantID,
		http:     httpCtx,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

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

func (c *Client) setAuth(req *http.Request) {
	if c.TenantID != "" {
		req.Header.Set("X-Scope-OrgID", c.TenantID)
	}

	switch c.AuthType {
	case "basic":
		credentials := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
		req.Header.Set("Authorization", "Basic "+credentials)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

// Ping checks connectivity by calling the Loki ready endpoint.
func (c *Client) Ping() error {
	url := fmt.Sprintf("%s/ready", c.BaseURL)
	_, err := c.execRequest(http.MethodGet, url, nil)
	return err
}

// PushLogStream represents a single stream in the Loki push request.
type PushLogStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// PushLogRequest represents the Loki push API request body.
type PushLogRequest struct {
	Streams []PushLogStream `json:"streams"`
}

// PushLogs sends log entries to Loki.
func (c *Client) PushLogs(req PushLogRequest) error {
	url := fmt.Sprintf("%s/loki/api/v1/push", c.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	_, err = c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	return nil
}

// QueryResponse represents the Loki query API response.
type QueryResponse struct {
	Status string    `json:"status"`
	Data   QueryData `json:"data"`
}

// QueryData contains the result data from a Loki query.
type QueryData struct {
	ResultType string         `json:"resultType"`
	Result     []StreamResult `json:"result"`
}

// StreamResult represents a single stream in the query result.
type StreamResult struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// QueryLogs queries logs from Loki using LogQL.
func (c *Client) QueryLogs(query, start, end, limit, direction string) (*QueryResponse, error) {
	url := fmt.Sprintf("%s/loki/api/v1/query_range", c.BaseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	q := req.URL.Query()
	q.Set("query", query)

	if start != "" {
		q.Set("start", start)
	}

	if end != "" {
		q.Set("end", end)
	}

	if limit != "" {
		q.Set("limit", limit)
	}

	if direction != "" {
		q.Set("direction", direction)
	}

	req.URL.RawQuery = q.Encode()

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

	var response QueryResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response, nil
}
