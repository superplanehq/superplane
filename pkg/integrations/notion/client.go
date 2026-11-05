package notion

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	baseURL         = "https://api.notion.com/v1"
	apiVersion      = "2026-03-11"
	maxRequestBytes = 500 * 1024
)

var errRequestTooLarge = errors.New("the Notion request exceeds the 500 KB limit")

type Client struct {
	token string
	http  core.HTTPContext
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RetryAfter string
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("Notion request failed (HTTP %d)", e.StatusCode)
	if e.Code != "" {
		message += ": " + e.Code
	}
	if e.Message != "" {
		message += ". " + e.Message
	}
	switch e.StatusCode {
	case http.StatusUnauthorized:
		message += ". Check the internal integration secret in the connection settings."
	case http.StatusForbidden, http.StatusNotFound:
		message += ". Check the resource ID, connection capabilities, and shared content in Notion."
	case http.StatusTooManyRequests, 529:
		if e.RetryAfter != "" {
			message += ". Wait " + e.RetryAfter + " seconds before another request."
		}
	}
	return message
}

func NewClient(httpContext core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	if integration == nil {
		return nil, fmt.Errorf("a Notion connection is required")
	}
	token, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("could not read the Notion integration secret")
	}
	value := strings.TrimSpace(string(token))
	if value == "" {
		return nil, fmt.Errorf("the Notion integration secret is required")
	}
	if httpContext == nil {
		return nil, fmt.Errorf("HTTP context is required")
	}
	return &Client{token: value, http: httpContext}, nil
}

func (c *Client) request(method, path string, body any) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("could not encode the Notion request: %w", err)
		}
		if len(encoded) > maxRequestBytes {
			return nil, errRequestTooLarge
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("could not build the Notion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", apiVersion)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not complete the Notion request: %s", c.safeMessage(err.Error()))
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the Notion response: %s", c.safeMessage(err.Error()))
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var result struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		// Do not include an unstructured response body in logs or execution errors.
		_ = json.Unmarshal(data, &result)
		return nil, &APIError{
			StatusCode: res.StatusCode, Code: c.safeMessage(result.Code),
			Message: c.safeMessage(result.Message), RetryAfter: c.safeMessage(res.Header.Get("Retry-After")),
		}
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil || result == nil {
		return nil, fmt.Errorf("Notion returned an invalid JSON object")
	}
	return result, nil
}

func (c *Client) safeMessage(message string) string {
	message = strings.ReplaceAll(message, c.token, "[redacted]")
	message = strings.Join(strings.Fields(message), " ")
	characters := []rune(message)
	if len(characters) > 1000 {
		return string(characters[:1000]) + "…"
	}
	return message
}

func (c *Client) Verify() error {
	user, err := c.request(http.MethodGet, "/users/me", nil)
	if err != nil {
		return err
	}
	if user["object"] != "user" || user["type"] != "bot" {
		return fmt.Errorf("Notion did not return an integration bot user")
	}
	return nil
}

type searchRequest struct {
	Query       string        `json:"query,omitempty"`
	Filter      *searchFilter `json:"filter,omitempty"`
	Sort        *searchSort   `json:"sort,omitempty"`
	PageSize    int           `json:"page_size"`
	StartCursor string        `json:"start_cursor,omitempty"`
}

type searchFilter struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

type searchSort struct {
	Direction string `json:"direction"`
	Timestamp string `json:"timestamp"`
}

func (c *Client) Search(request searchRequest) (map[string]any, error) {
	result, err := c.request(http.MethodPost, "/search", request)
	if err != nil {
		return nil, err
	}
	if _, ok := result["results"].([]any); !ok || result["object"] != "list" {
		return nil, fmt.Errorf("Notion returned an invalid search result")
	}
	return result, nil
}

func (c *Client) DataSource(id string) (map[string]any, error) {
	return c.request(http.MethodGet, "/data_sources/"+url.PathEscape(id), nil)
}

type createPageRequest struct {
	Parent     map[string]string `json:"parent"`
	Properties map[string]any    `json:"properties"`
	Children   []map[string]any  `json:"children,omitempty"`
}

func (c *Client) CreatePage(request createPageRequest) (map[string]any, error) {
	page, err := c.request(http.MethodPost, "/pages", request)
	if err != nil {
		if errors.Is(err, errRequestTooLarge) {
			return nil, err
		}
		var apiError *APIError
		if !errors.As(err, &apiError) || apiError.StatusCode >= 500 {
			return nil, fmt.Errorf("%w. The page may exist. Check Notion before you try again", err)
		}
		return nil, err
	}
	if page["object"] != "page" || page["id"] == nil || page["id"] == "" {
		return nil, fmt.Errorf("Notion returned an invalid page. Check Notion before you try again")
	}
	return page, nil
}
