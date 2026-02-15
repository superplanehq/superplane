package prometheus

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
	http        core.HTTPContext
}

type prometheusResponse[T any] struct {
	Status    string `json:"status"`
	Data      T      `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

type PrometheusAlertsData struct {
	Alerts []PrometheusAlert `json:"alerts"`
}

type PrometheusAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    string            `json:"activeAt,omitempty"`
	Value       string            `json:"value,omitempty"`
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
		http:     httpContext,
	}

	switch authType {
	case AuthTypeNone:
		return client, nil
	case AuthTypeBasic:
		username, err := requiredConfig(integration, "username")
		if err != nil {
			return nil, fmt.Errorf("username is required when authType is basic")
		}
		password, err := requiredConfig(integration, "password")
		if err != nil {
			return nil, fmt.Errorf("password is required when authType is basic")
		}

		client.username = username
		client.password = password
		return client, nil
	case AuthTypeBearer:
		bearerToken, err := requiredConfig(integration, "bearerToken")
		if err != nil {
			return nil, fmt.Errorf("bearerToken is required when authType is bearer")
		}

		client.bearerToken = bearerToken
		return client, nil
	default:
		return nil, fmt.Errorf("invalid authType %q", authType)
	}
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

func (c *Client) GetAlertsFromPrometheus() ([]PrometheusAlert, error) {
	body, err := c.execRequest(http.MethodGet, "/api/v1/alerts")
	if err != nil {
		return nil, err
	}

	response := prometheusResponse[PrometheusAlertsData]{}
	if err := decodeResponse(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, formatPrometheusError(response.ErrorType, response.Error)
	}

	return response.Data.Alerts, nil
}

func (c *Client) Query(query string) (map[string]any, error) {
	apiPath := fmt.Sprintf("/api/v1/query?query=%s", url.QueryEscape(query))
	body, err := c.execRequest(http.MethodGet, apiPath)
	if err != nil {
		return nil, err
	}

	response := prometheusResponse[map[string]any]{}
	if err := decodeResponse(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, formatPrometheusError(response.ErrorType, response.Error)
	}

	return response.Data, nil
}

// AlertmanagerSilence represents a silence object from the Alertmanager API.
type AlertmanagerSilence struct {
	ID        string            `json:"id"`
	Status    *SilenceStatus    `json:"status,omitempty"`
	Matchers  []SilenceMatcher  `json:"matchers"`
	StartsAt  string            `json:"startsAt"`
	EndsAt    string            `json:"endsAt"`
	CreatedBy string            `json:"createdBy"`
	Comment   string            `json:"comment"`
	UpdatedAt string            `json:"updatedAt,omitempty"`
}

// SilenceStatus represents the status of a silence.
type SilenceStatus struct {
	State string `json:"state"`
}

// SilenceMatcher represents a matcher within a silence.
type SilenceMatcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
	IsEqual bool   `json:"isEqual"`
}

// CreateSilenceRequest is the request body for creating a silence.
type CreateSilenceRequest struct {
	Matchers  []SilenceMatcher `json:"matchers"`
	StartsAt  string           `json:"startsAt"`
	EndsAt    string           `json:"endsAt"`
	CreatedBy string           `json:"createdBy"`
	Comment   string           `json:"comment"`
}

// CreateSilenceResponse is the response from creating a silence.
type CreateSilenceResponse struct {
	SilenceID string `json:"silenceID"`
}

// CreateSilence creates a silence in Alertmanager.
func (c *Client) CreateSilence(request CreateSilenceRequest) (string, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := c.execRequestWithBody(http.MethodPost, "/api/v2/silences", body)
	if err != nil {
		return "", err
	}

	response := CreateSilenceResponse{}
	if err := decodeResponse(respBody, &response); err != nil {
		return "", err
	}

	if response.SilenceID == "" {
		return "", fmt.Errorf("empty silence ID in response")
	}

	return response.SilenceID, nil
}

// ExpireSilence expires (deletes) a silence in Alertmanager.
func (c *Client) ExpireSilence(silenceID string) error {
	path := fmt.Sprintf("/api/v2/silence/%s", url.PathEscape(silenceID))
	_, err := c.execRequestWithBody(http.MethodDelete, path, nil)
	return err
}

// GetSilence retrieves a silence by ID from Alertmanager.
func (c *Client) GetSilence(silenceID string) (*AlertmanagerSilence, error) {
	path := fmt.Sprintf("/api/v2/silence/%s", url.PathEscape(silenceID))
	body, err := c.execRequest(http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	silence := AlertmanagerSilence{}
	if err := decodeResponse(body, &silence); err != nil {
		return nil, err
	}

	return &silence, nil
}

// QueryRange executes a range query against the Prometheus API.
func (c *Client) QueryRange(query, start, end, step string) (map[string]any, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", start)
	params.Set("end", end)
	params.Set("step", step)

	apiPath := fmt.Sprintf("/api/v1/query_range?%s", params.Encode())
	body, err := c.execRequest(http.MethodGet, apiPath)
	if err != nil {
		return nil, err
	}

	response := prometheusResponse[map[string]any]{}
	if err := decodeResponse(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, formatPrometheusError(response.ErrorType, response.Error)
	}

	return response.Data, nil
}

func (c *Client) execRequestWithBody(method string, path string, body []byte) ([]byte, error) {
	apiURL := c.baseURL
	if strings.HasPrefix(path, "/") {
		apiURL += path
	} else {
		apiURL += "/" + path
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, apiURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(respBody) > MaxResponseSize {
		return nil, fmt.Errorf("response too large: exceeds maximum size of %d bytes", MaxResponseSize)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", res.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Client) execRequest(method string, path string) ([]byte, error) {
	apiURL := c.baseURL
	if strings.HasPrefix(path, "/") {
		apiURL += path
	} else {
		apiURL += "/" + path
	}

	req, err := http.NewRequest(method, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if err := c.setAuth(req); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	limitedReader := io.LimitReader(res.Body, MaxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) > MaxResponseSize {
		return nil, fmt.Errorf("response too large: exceeds maximum size of %d bytes", MaxResponseSize)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", res.StatusCode, string(body))
	}

	return body, nil
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

func decodeResponse[T any](body []byte, out *T) error {
	if len(body) == 0 {
		return fmt.Errorf("empty response body")
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to decode response JSON: %w", err)
	}

	return nil
}

func formatPrometheusError(errorType string, errorMessage string) error {
	if errorType == "" && errorMessage == "" {
		return fmt.Errorf("prometheus API returned non-success status")
	}

	if errorType == "" {
		return fmt.Errorf("prometheus API error: %s", errorMessage)
	}

	if errorMessage == "" {
		return fmt.Errorf("prometheus API error type: %s", errorType)
	}

	return fmt.Errorf("prometheus API error (%s): %s", errorType, errorMessage)
}
