package grafana

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

const (
	maxResponseSize = 2 * 1024 * 1024 // 2MB
)

type Client struct {
	BaseURL  string
	APIToken string
	http     core.HTTPContext
}

type contactPoint struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type DataSource struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type apiStatusError struct {
	Operation    string
	StatusCode   int
	ResponseBody string
}

func (e *apiStatusError) Error() string {
	return fmt.Sprintf("%s failed with status %d: %s", e.Operation, e.StatusCode, e.ResponseBody)
}

func newAPIStatusError(operation string, status int, responseBody []byte) error {
	return &apiStatusError{
		Operation:    operation,
		StatusCode:   status,
		ResponseBody: string(responseBody),
	}
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext, requireToken bool) (*Client, error) {
	baseURL, err := readBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	apiToken, err := readAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	if requireToken && apiToken == "" {
		return nil, fmt.Errorf("apiToken is required")
	}

	return &Client{
		BaseURL:  baseURL,
		APIToken: apiToken,
		http:     httpCtx,
	}, nil
}

func readBaseURL(ctx core.IntegrationContext) (string, error) {
	baseURLConfig, err := ctx.GetConfig("baseURL")
	if err != nil {
		return "", fmt.Errorf("error reading baseURL: %v", err)
	}

	if baseURLConfig == nil {
		return "", fmt.Errorf("baseURL is required")
	}

	baseURLRaw := strings.TrimSpace(string(baseURLConfig))
	if baseURLRaw == "" {
		return "", fmt.Errorf("baseURL is required")
	}

	parsed, err := url.Parse(baseURLRaw)
	if err != nil {
		return "", fmt.Errorf("invalid baseURL: %v", err)
	}

	// url.Parse accepts relative URLs (e.g. "grafana.local"), which will fail later in http.NewRequest.
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid baseURL: must include scheme and host (e.g. https://grafana.example.com)")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid baseURL: unsupported scheme %q (expected http or https)", parsed.Scheme)
	}

	return strings.TrimSuffix(baseURLRaw, "/"), nil
}

func readAPIToken(ctx core.IntegrationContext) (string, error) {
	type optionalConfigReader interface {
		GetOptionalConfig(name string) ([]byte, error)
	}

	var (
		apiTokenConfig []byte
		err            error
	)

	if optionalCtx, ok := ctx.(optionalConfigReader); ok {
		apiTokenConfig, err = optionalCtx.GetOptionalConfig("apiToken")
	} else {
		apiTokenConfig, err = ctx.GetConfig("apiToken")
		if err != nil && strings.Contains(err.Error(), "config apiToken not found") {
			return "", nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("error reading apiToken: %v", err)
	}

	if apiTokenConfig == nil {
		return "", nil
	}

	return strings.TrimSpace(string(apiTokenConfig)), nil
}

func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.BaseURL, "/"), strings.TrimPrefix(path, "/"))
}

func (c *Client) execRequest(method, path string, body io.Reader, contentType string) ([]byte, int, error) {
	return c.execRequestWithHeaders(method, path, body, contentType, nil)
}

func (c *Client) execRequestWithHeaders(
	method, path string,
	body io.Reader,
	contentType string,
	headers map[string]string,
) ([]byte, int, error) {
	req, err := http.NewRequest(method, c.buildURL(path), body)
	if err != nil {
		return nil, 0, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.APIToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	// Read one byte beyond the max to detect overflow without rejecting an exact-limit response.
	limitedReader := io.LimitReader(res.Body, int64(maxResponseSize)+1)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("error reading body: %v", err)
	}

	if len(responseBody) > maxResponseSize {
		return nil, res.StatusCode, fmt.Errorf("response too large: exceeds maximum size of %d bytes", maxResponseSize)
	}

	return responseBody, res.StatusCode, nil
}

func (c *Client) listContactPoints() ([]contactPoint, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/api/v1/provisioning/contact-points", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error listing contact points: %v", err)
	}

	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana contact point list", status, responseBody)
	}

	var direct []contactPoint
	if err := json.Unmarshal(responseBody, &direct); err == nil {
		return direct, nil
	}

	wrapped := struct {
		Items json.RawMessage `json:"items"`
	}{}
	if err := json.Unmarshal(responseBody, &wrapped); err == nil {
		if wrapped.Items == nil || bytes.Equal(bytes.TrimSpace(wrapped.Items), []byte("null")) {
			return nil, fmt.Errorf("error parsing contact points response")
		}

		var items []contactPoint
		if err := json.Unmarshal(wrapped.Items, &items); err != nil {
			return nil, fmt.Errorf("error parsing contact points response")
		}

		return items, nil
	}

	return nil, fmt.Errorf("error parsing contact points response")
}

func (c *Client) UpsertWebhookContactPoint(name, webhookURL, bearerToken string) (string, error) {
	points, err := c.listContactPoints()
	if err != nil {
		return "", err
	}

	existingUID := ""
	for _, point := range points {
		if strings.TrimSpace(point.Name) == name {
			existingUID = strings.TrimSpace(point.UID)
			break
		}
	}

	payload := map[string]any{
		"name":                  name,
		"type":                  "webhook",
		"disableResolveMessage": false,
		"settings": map[string]any{
			"url":        webhookURL,
			"httpMethod": "POST",
		},
	}

	if bearerToken != "" {
		settings := payload["settings"].(map[string]any)
		settings["authorization_scheme"] = "Bearer"
		settings["authorization_credentials"] = bearerToken
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling contact point payload: %v", err)
	}

	if existingUID != "" {
		responseBody, status, err := c.execRequestWithHeaders(
			http.MethodPut,
			fmt.Sprintf("/api/v1/provisioning/contact-points/%s", existingUID),
			bytes.NewReader(body),
			"application/json",
			map[string]string{
				"X-Disable-Provenance": "true",
			},
		)
		if err != nil {
			return "", fmt.Errorf("error updating contact point: %v", err)
		}
		if status < 200 || status >= 300 {
			return "", newAPIStatusError("grafana contact point update", status, responseBody)
		}
		return existingUID, nil
	}

	responseBody, status, err := c.execRequestWithHeaders(
		http.MethodPost,
		"/api/v1/provisioning/contact-points",
		bytes.NewReader(body),
		"application/json",
		map[string]string{
			"X-Disable-Provenance": "true",
		},
	)
	if err != nil {
		return "", fmt.Errorf("error creating contact point: %v", err)
	}
	if status < 200 || status >= 300 {
		return "", newAPIStatusError("grafana contact point create", status, responseBody)
	}

	created := contactPoint{}
	if err := json.Unmarshal(responseBody, &created); err == nil && strings.TrimSpace(created.UID) != "" {
		return strings.TrimSpace(created.UID), nil
	}

	refreshedPoints, err := c.listContactPoints()
	if err != nil {
		return "", err
	}

	for _, point := range refreshedPoints {
		if strings.TrimSpace(point.Name) == name && strings.TrimSpace(point.UID) != "" {
			return strings.TrimSpace(point.UID), nil
		}
	}

	return "", fmt.Errorf("contact point created but uid was not returned")
}

func (c *Client) DeleteContactPoint(uid string) error {
	if strings.TrimSpace(uid) == "" {
		return nil
	}

	responseBody, status, err := c.execRequest(http.MethodDelete, fmt.Sprintf("/api/v1/provisioning/contact-points/%s", uid), nil, "")
	if err != nil {
		return fmt.Errorf("error deleting contact point: %v", err)
	}

	if status == http.StatusNotFound {
		return nil
	}

	if status < 200 || status >= 300 {
		return newAPIStatusError("grafana contact point delete", status, responseBody)
	}

	return nil
}

func (c *Client) ListDataSources() ([]DataSource, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/api/datasources", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error listing data sources: %v", err)
	}

	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana data source list", status, responseBody)
	}

	var sources []DataSource
	if err := json.Unmarshal(responseBody, &sources); err != nil {
		return nil, fmt.Errorf("error parsing data sources response: %v", err)
	}

	return sources, nil
}
