package prometheus

import (
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
	baseURL         string
	alertmanagerURL string
	authType        string
	username        string
	password        string
	bearerToken     string
	http            core.HTTPContext
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

type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
	IsEqual bool   `json:"isEqual"`
}

type SilencePayload struct {
	Matchers  []Matcher `json:"matchers"`
	StartsAt  string    `json:"startsAt"`
	EndsAt    string    `json:"endsAt"`
	CreatedBy string    `json:"createdBy"`
	Comment   string    `json:"comment"`
}

type silenceResponse struct {
	SilenceID string `json:"silenceID"`
}

type AlertmanagerSilence struct {
	ID      string                    `json:"id"`
	Status  AlertmanagerSilenceStatus `json:"status"`
	Comment string                    `json:"comment"`
}

type AlertmanagerSilenceStatus struct {
	State string `json:"state"`
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

	alertmanagerURL := optionalConfig(integration, "alertmanagerURL")

	client := &Client{
		baseURL:         normalizeBaseURL(baseURL),
		alertmanagerURL: normalizeBaseURL(alertmanagerURL),
		authType:        authType,
		http:            httpContext,
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

func (c *Client) alertmanagerBaseURL() string {
	if c.alertmanagerURL != "" {
		return c.alertmanagerURL
	}
	return c.baseURL
}

func (c *Client) CreateSilence(silence SilencePayload) (string, error) {
	jsonBody, err := json.Marshal(silence)
	if err != nil {
		return "", fmt.Errorf("failed to marshal silence payload: %w", err)
	}

	apiURL := c.alertmanagerBaseURL() + "/api/v2/silences"
	body, err := c.execRequestWithBodyAndURL(http.MethodPost, apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}

	response := silenceResponse{}
	if err := decodeResponse(body, &response); err != nil {
		return "", err
	}

	return response.SilenceID, nil
}

func (c *Client) ListSilences() ([]AlertmanagerSilence, error) {
	apiURL := c.alertmanagerBaseURL() + "/api/v2/silences"
	body, err := c.execRequestWithBodyAndURL(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	response := []AlertmanagerSilence{}
	if err := decodeResponse(body, &response); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) ExpireSilence(silenceID string) error {
	apiURL := fmt.Sprintf("%s/api/v2/silence/%s", c.alertmanagerBaseURL(), silenceID)
	_, err := c.execRequestWithBodyAndURL(http.MethodDelete, apiURL, nil)
	return err
}

func (c *Client) execRequest(method string, path string) ([]byte, error) {
	return c.execRequestWithBody(method, path, nil)
}

func (c *Client) execRequestWithBodyAndURL(method string, fullURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, fullURL, body)
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

	return c.doRequest(req)
}

func (c *Client) execRequestWithBody(method string, path string, body io.Reader) ([]byte, error) {
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
	if err := c.setAuth(req); err != nil {
		return nil, err
	}

	return c.doRequest(req)
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
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
