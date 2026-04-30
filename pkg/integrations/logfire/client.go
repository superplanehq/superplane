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
			baseURL = ensureScheme(trimmedBaseURL)
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
		http:       httpCtx,
	}, nil
}

func (c *Client) validateReadToken(readToken string) error {
	if strings.TrimSpace(readToken) == "" {
		return fmt.Errorf("logfire read token is required")
	}

	_, err := c.executeQuery(readToken, QueryRequest{SQL: validateQuerySQL, Limit: 1})
	if err == nil {
		return nil
	}

	if isUnauthorizedError(err) {
		return fmt.Errorf("invalid Logfire read token - please verify your token and try again")
	}
	return fmt.Errorf("could not connect to Logfire - please verify your read token and base URL, then try again: %w", err)
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

type AlertChannel struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Config *struct {
		URL string `json:"url"`
	} `json:"config,omitempty"`
}

func (c *Client) ExecuteQueryWithToken(readToken string, request QueryRequest) (*QueryResponse, error) {
	return c.executeQuery(readToken, request)
}

func readTokenSecretNameForProject(projectID string) string {
	return readTokenSecretName + ":" + strings.TrimSpace(projectID)
}

func (c *Client) executeQuery(readToken string, request QueryRequest) (*QueryResponse, error) {
	sql := strings.TrimSpace(request.SQL)
	if sql == "" {
		return nil, fmt.Errorf("sql is required")
	}

	queryURL, err := url.Parse(c.BaseURL + logfireQueryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query url: %w", err)
	}
	queryURL.RawQuery = buildQueryParams(sql, request).Encode()

	body, err := c.execRequest(http.MethodGet, queryURL.String(), nil, readToken)
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
		// Logfire's query API expects json_rows=true for row-oriented JSON payloads.
		params.Set("json_rows", "true")
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

type readTokenCreateResult struct {
	ID    string
	Token string
}

func (c *Client) createReadToken(projectID, name string) (readTokenCreateResult, error) {
	payload, err := json.Marshal(map[string]any{"name": name})
	if err != nil {
		return readTokenCreateResult{}, fmt.Errorf("failed to encode read token payload: %w", err)
	}

	body, err := c.execRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/projects/%s/read-tokens/", c.APIBaseURL, strings.TrimSpace(projectID)),
		bytes.NewReader(payload),
		c.APIKey,
	)
	if err != nil {
		return readTokenCreateResult{}, err
	}

	var response struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return readTokenCreateResult{}, fmt.Errorf("failed to decode read token response: %w", err)
	}

	if strings.TrimSpace(response.Token) == "" {
		return readTokenCreateResult{}, fmt.Errorf("read token not returned by Logfire")
	}

	return readTokenCreateResult{
		ID:    strings.TrimSpace(response.ID),
		Token: strings.TrimSpace(response.Token),
	}, nil
}

// ProvisionReadToken creates a read token for the given project and validates it.
// If validation fails, the created token is cleaned up.
func (c *Client) ProvisionReadToken(projectID string) (string, error) {
	created, err := c.createReadToken(projectID, defaultReadTokenName)
	if err != nil {
		return "", fmt.Errorf("failed to create read token for project %s: %w", projectID, err)
	}

	if err := c.validateReadToken(created.Token); err != nil {
		if strings.TrimSpace(created.ID) != "" {
			_ = c.DeleteReadToken(projectID, created.ID)
		}
		return "", fmt.Errorf("created read token for project %s is not usable: %w", projectID, err)
	}

	return created.Token, nil
}

func (c *Client) DeleteReadToken(projectID, readTokenID string) error {
	if strings.TrimSpace(readTokenID) == "" {
		return nil
	}

	path := fmt.Sprintf(
		"%s/api/v1/projects/%s/read-tokens/%s/",
		c.APIBaseURL,
		strings.TrimSpace(projectID),
		strings.TrimSpace(readTokenID),
	)
	_, err := c.execRequest(http.MethodDelete, path, nil, c.APIKey)
	return err
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

func (c *Client) FindAssignedAlertChannelID(projectID, alertID, preferredLabel, webhookURL string) (string, bool, error) {
	alertChannelIDs, err := c.GetAlertChannelIDs(projectID, alertID)
	if err != nil {
		return "", false, err
	}
	if len(alertChannelIDs) == 0 {
		return "", false, nil
	}

	channels, err := c.listAlertChannels(defaultChannelsPath)
	if err != nil {
		return "", false, err
	}

	// Prefer a channel with the expected label, but fall back to any channel that matches our webhook URL.
	var preferredID string
	var urlMatchID string
	for _, ch := range channels {
		if strings.EqualFold(ch.Label, preferredLabel) {
			if ch.Config == nil || strings.TrimSpace(ch.Config.URL) == "" || strings.TrimSpace(ch.Config.URL) != webhookURL {
				continue
			}
			preferredID = ch.ID
		}

		if ch.Config != nil && strings.TrimSpace(ch.Config.URL) == webhookURL {
			urlMatchID = ch.ID
		}
	}

	if preferredID != "" {
		for _, id := range alertChannelIDs {
			if id == preferredID {
				return preferredID, true, nil
			}
		}
	}

	if urlMatchID != "" {
		for _, id := range alertChannelIDs {
			if id == urlMatchID {
				return urlMatchID, true, nil
			}
		}
	}

	return "", false, nil
}

func (c *Client) EnsureAlertHasChannelID(projectID, alertID, channelID string) error {
	channelIDs, err := c.GetAlertChannelIDs(projectID, alertID)
	if err != nil {
		return err
	}

	for _, id := range channelIDs {
		if id == channelID {
			return nil
		}
	}

	channelIDs = append(channelIDs, channelID)

	payload, err := json.Marshal(map[string]any{
		"channel_ids": channelIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to encode alert patch payload: %w", err)
	}

	// Logfire API supports PATCH on alert endpoints. If not, this will fail and be retried.
	alertURL := fmt.Sprintf("%s/api/v1/projects/%s/alerts/%s/", c.APIBaseURL, strings.TrimSpace(projectID), strings.TrimSpace(alertID))

	attemptMethods := []string{http.MethodPatch, http.MethodPut, http.MethodPost}
	var lastErr error
	for _, method := range attemptMethods {
		_, methodErr := c.execRequest(method, alertURL, bytes.NewReader(payload), c.APIKey)
		if methodErr == nil {
			return nil
		}

		lastErr = methodErr
		var apiErr *APIError
		if !errors.As(methodErr, &apiErr) || apiErr.StatusCode != http.StatusMethodNotAllowed {
			// For non-405 errors, don't keep retrying with other methods.
			return methodErr
		}
	}

	return fmt.Errorf("failed to update Logfire alert channels (methods tried: %v): %w", attemptMethods, lastErr)
}

func (c *Client) GetAlertChannelIDs(projectID, alertID string) ([]string, error) {
	alertURL := fmt.Sprintf("%s/api/v1/projects/%s/alerts/%s/", c.APIBaseURL, strings.TrimSpace(projectID), strings.TrimSpace(alertID))
	body, err := c.execRequest(http.MethodGet, alertURL, nil, c.APIKey)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode alert details: %w", err)
	}

	var ids []string
	for _, key := range []string{"channel_ids", "channelIds", "channels"} {
		v, ok := raw[key]
		if !ok {
			continue
		}

		if typed, ok := v.([]any); ok {
			for _, item := range typed {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
					ids = append(ids, s)
				}
			}
		}

		break
	}

	return ids, nil
}

type Alert struct {
	ID   string
	Name string
}

func (c *Client) ListAlerts(projectID string) ([]Alert, error) {
	alertsURL := fmt.Sprintf("%s/api/v1/projects/%s/alerts/", c.APIBaseURL, strings.TrimSpace(projectID))
	body, err := c.execRequest(http.MethodGet, alertsURL, nil, c.APIKey)
	if err != nil {
		return nil, err
	}

	var raw []map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		// Try list shape that isn't an array of objects.
		var fallback any
		if err2 := json.Unmarshal(body, &fallback); err2 != nil {
			return nil, fmt.Errorf("failed to decode alerts response: %w", err)
		}
		return nil, fmt.Errorf("failed to decode alerts response")
	}

	out := make([]Alert, 0, len(raw))
	for _, item := range raw {
		id := ""
		if v, ok := item["id"].(string); ok {
			id = v
		} else if v, ok := item["alert_id"].(string); ok {
			id = v
		}

		name := ""
		if v, ok := item["name"].(string); ok {
			name = v
		} else if v, ok := item["alert_name"].(string); ok {
			name = v
		}

		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		out = append(out, Alert{ID: id, Name: strings.TrimSpace(name)})
	}

	return out, nil
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

func ensureScheme(rawURL string) string {
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	return "https://" + rawURL
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
