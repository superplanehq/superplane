package logfire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	logfireUSBaseURL     = "https://logfire-us.pydantic.dev"
	logfireEUBaseURL     = "https://logfire-eu.pydantic.dev"
	logfireUSAPIBaseURL  = "https://api-us.pydantic.dev"
	logfireEUAPIBaseURL  = "https://api-eu.pydantic.dev"
	readTokenSecretName  = "logfireReadToken"
	defaultReadTokenName = "superplane-query-token"
	defaultChannelsPath  = "/api/v1/channels/"
	logfireQueryPath     = "/v1/query"
	validateQuerySQL     = "SELECT start_timestamp FROM records LIMIT 1"
)

type Client struct {
	APIKey     string
	ReadToken  string
	BaseURL    string
	APIBaseURL string
	http       core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKeyConfig, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}
	apiKey := string(apiKeyConfig)
	baseURL := logfireUSBaseURL
	isBaseURLConfigured := false
	configuredBaseURL, err := ctx.GetConfig("baseURL")
	if err == nil {
		trimmedBaseURL := strings.TrimSpace(string(configuredBaseURL))
		if trimmedBaseURL != "" {
			baseURL = trimmedBaseURL
			isBaseURLConfigured = true
		}
	}

	apiBaseURL := deriveAPIBaseURL(baseURL)
	if regionBaseURL, regionAPIBaseURL, ok := deriveRegionBaseURLsFromAPIKey(apiKey); ok {
		// API key region is authoritative for API calls. Key contain us/eu for corresponding region.
		apiBaseURL = regionAPIBaseURL
		if !isBaseURLConfigured {
			baseURL = regionBaseURL
		}
	}

	return &Client{
		APIKey:     apiKey,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIBaseURL: apiBaseURL,
		ReadToken:  findSecretValue(ctx, readTokenSecretName),
		http:       httpCtx,
	}, nil
}

func (c *Client) ValidateCredentials() error {
	if strings.TrimSpace(c.ReadToken) == "" {
		return fmt.Errorf("logfire read token is required")
	}

	_, err := c.ExecuteQuery(QueryRequest{SQL: validateQuerySQL, Limit: 1})
	if err == nil {
		return nil
	}

	if isUnauthorizedError(err) {
		return fmt.Errorf("invalid Logfire read token - please verify your token and try again")
	}
	return fmt.Errorf("could not connect to Logfire - please verify your read token and base URL, then try again: %w", err)
}

// BootstrapIntegration follows the verified API-key flow:
// 1) List projects using API key.
// 2) Create read token for the project using API key.
// 3) Verify the generated read token by running a simple query.
func (c *Client) BootstrapIntegration(readTokenName string) (*IntegrationBootstrapResult, error) {
	projects, err := c.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("invalid Logfire API key - please verify your key and try again: %w", err)
	}
	if len(projects) == 0 {
		return nil, fmt.Errorf("no Logfire projects available for this API key")
	}

	readToken, err := c.CreateReadToken(projects[0].ID, readTokenName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Logfire read token: %w", err)
	}

	c.ReadToken = readToken
	if err := c.ValidateCredentials(); err != nil {
		return nil, err
	}

	return &IntegrationBootstrapResult{
		Project:   projects[0],
		ReadToken: readToken,
	}, nil
}

func isUnauthorizedError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request got %d code: %s", e.StatusCode, e.Body)
}

type QueryRequest struct {
	SQL          string `json:"sql"`
	MinTimestamp string `json:"min_timestamp,omitempty"`
	MaxTimestamp string `json:"max_timestamp,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	RowOriented  bool   `json:"row_oriented,omitempty"`
}

type QueryResponse struct {
	Columns any `json:"columns,omitempty"`
	Rows    any `json:"rows,omitempty"`
}

type Project struct {
	ID               string `json:"id"`
	OrganizationName string `json:"organization_name"`
	ProjectName      string `json:"project_name"`
}

type IntegrationBootstrapResult struct {
	Project   Project
	ReadToken string
}

type AlertChannel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (c *Client) ExecuteQuery(request QueryRequest) (*QueryResponse, error) {
	sql := strings.TrimSpace(request.SQL)
	if sql == "" {
		return nil, fmt.Errorf("sql is required")
	}

	queryURL, err := url.Parse(c.BaseURL + logfireQueryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query url: %w", err)
	}
	queryURL.RawQuery = buildQueryParams(sql, request).Encode()

	body, err := c.execRequest(http.MethodGet, queryURL.String(), nil, c.ReadToken)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	var response QueryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}
	return &response, nil
}

func buildQueryParams(sql string, request QueryRequest) url.Values {
	params := url.Values{"sql": []string{sql}}

	if request.MinTimestamp != "" {
		params.Set("min_timestamp", request.MinTimestamp)
	}
	if request.MaxTimestamp != "" {
		params.Set("max_timestamp", request.MaxTimestamp)
	}
	if request.Limit > 0 {
		params.Set("limit", strconv.Itoa(request.Limit))
	}
	if request.RowOriented {
		params.Set("row_oriented", "true")
	}
	return params
}

func (c *Client) execRequest(method, requestURL string, body io.Reader, token string) ([]byte, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if !isSuccessfulStatusCode(resp.StatusCode) {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(responseBody),
		}
	}

	return responseBody, nil
}

func (c *Client) ListProjects() ([]Project, error) {
	body, err := c.execRequest(http.MethodGet, c.APIBaseURL+"/api/v1/projects/", nil, c.APIKey)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects response: %w", err)
	}

	return projects, nil
}

func (c *Client) CreateReadToken(projectID, name string) (string, error) {
	payload, err := json.Marshal(map[string]any{"name": name})
	if err != nil {
		return "", fmt.Errorf("failed to encode read token payload: %w", err)
	}

	body, err := c.execRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/projects/%s/read-tokens/", c.APIBaseURL, strings.TrimSpace(projectID)),
		bytes.NewReader(payload),
		c.APIKey,
	)
	if err != nil {
		return "", err
	}

	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to decode read token response: %w", err)
	}

	if strings.TrimSpace(response.Token) == "" {
		return "", fmt.Errorf("read token not returned by Logfire")
	}

	return strings.TrimSpace(response.Token), nil
}

func (c *Client) UpsertAlertChannel(label, webhookURL, _ string) (*AlertChannel, string, error) {
	path := defaultChannelsPath

	payload, err := json.Marshal(map[string]any{
		"label": label,
		"config": map[string]any{
			"type":   "webhook",
			"format": "slack-legacy",
			"url":    webhookURL,
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode channel payload: %w", err)
	}

	body, err := c.execRequest(http.MethodPost, c.APIBaseURL+path, bytes.NewReader(payload), c.APIKey)
	if err == nil {
		channel := AlertChannel{}
		if err := json.Unmarshal(body, &channel); err != nil {
			return nil, "", fmt.Errorf("failed to decode channel response: %w", err)
		}

		return &channel, path, nil
	}

	// Fallback for idempotent setup: if channel already exists, find and reuse it.
	if !isChannelAlreadyExistsError(err) {
		return nil, "", err
	}

	channels, listErr := c.listAlertChannels(path)
	if listErr != nil {
		return nil, "", listErr
	}

	for _, channel := range channels {
		if strings.EqualFold(strings.TrimSpace(channel.Label), strings.TrimSpace(label)) {
			return &channel, path, nil
		}
	}

	return nil, "", fmt.Errorf("channel already exists, but could not find it by label %q", label)
}

func (c *Client) DeleteAlertChannel(channelID, channelsPath string) error {
	if strings.TrimSpace(channelID) == "" {
		return nil
	}

	path := strings.TrimSpace(channelsPath)
	if path == "" {
		path = defaultChannelsPath
	}

	_, err := c.execRequest(http.MethodDelete, fmt.Sprintf("%s%s%s/", c.APIBaseURL, path, strings.TrimSpace(channelID)), nil, c.APIKey)
	return err
}

func (c *Client) listAlertChannels(path string) ([]AlertChannel, error) {
	body, err := c.execRequest(http.MethodGet, c.APIBaseURL+path, nil, c.APIKey)
	if err != nil {
		return nil, err
	}

	var channels []AlertChannel
	if err := json.Unmarshal(body, &channels); err != nil {
		return nil, fmt.Errorf("failed to decode channels response: %w", err)
	}

	return channels, nil
}

func isChannelAlreadyExistsError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	if apiErr.StatusCode == http.StatusConflict {
		return true
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		return false
	}

	body := strings.ToLower(strings.TrimSpace(apiErr.Body))
	return strings.Contains(body, "already exists")
}

func isSuccessfulStatusCode(statusCode int) bool {
	return statusCode == http.StatusOK || statusCode == http.StatusCreated || statusCode == http.StatusNoContent
}

func deriveAPIBaseURL(baseURL string) string {
	normalized := strings.ToLower(strings.TrimSpace(baseURL))
	if strings.Contains(normalized, "api-eu.") || strings.Contains(normalized, "logfire-eu.") {
		return logfireEUAPIBaseURL
	}

	return logfireUSAPIBaseURL
}

func deriveRegionBaseURLsFromAPIKey(apiKey string) (string, string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(apiKey))
	switch {
	case strings.Contains(normalized, "_eu_"):
		return logfireEUBaseURL, logfireEUAPIBaseURL, true
	case strings.Contains(normalized, "_us_"):
		return logfireUSBaseURL, logfireUSAPIBaseURL, true
	default:
		return "", "", false
	}
}

func findSecretValue(ctx core.IntegrationContext, name string) string {
	if ctx == nil {
		return ""
	}

	secrets, err := ctx.GetSecrets()
	if err != nil {
		return ""
	}

	for _, secret := range secrets {
		if secret.Name == name {
			return string(secret.Value)
		}
	}

	return ""
}
