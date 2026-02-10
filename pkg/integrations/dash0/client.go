package dash0

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
	// MaxResponseSize limits API responses to prevent excessive memory usage.
	MaxResponseSize = 1 * 1024 * 1024 // 1MB
)

// Client wraps authenticated HTTP access to Dash0 APIs.
type Client struct {
	Token         string
	BaseURL       string
	LogsIngestURL string
	Dataset       string
	http          core.HTTPContext
}

// NewClient builds a Dash0 API client from integration configuration.
func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("dash0 client: get api token: %w", err)
	}

	baseURL := ""
	baseURLConfig, err := ctx.GetConfig("baseURL")
	if err == nil && baseURLConfig != nil && len(baseURLConfig) > 0 {
		baseURL = strings.TrimSuffix(string(baseURLConfig), "/")
	}

	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("dash0 client: baseURL is required for Dash0 Cloud. Find your API URL in Dash0 dashboard under Organization Settings > Endpoints Reference")
	}

	// Strip /api/prometheus if user included it in the base URL.
	baseURL = strings.TrimSuffix(baseURL, "/api/prometheus")

	dataset := "default"
	datasetConfig, err := ctx.GetConfig("dataset")
	if err == nil && datasetConfig != nil && len(datasetConfig) > 0 {
		trimmedDataset := strings.TrimSpace(string(datasetConfig))
		if trimmedDataset != "" {
			dataset = trimmedDataset
		}
	}

	logsIngestURL := deriveLogsIngestURL(baseURL)

	return &Client{
		Token:         string(apiToken),
		BaseURL:       baseURL,
		LogsIngestURL: logsIngestURL,
		Dataset:       dataset,
		http:          http,
	}, nil
}

// deriveLogsIngestURL derives the OTLP logs ingress host from the configured API base URL.
func deriveLogsIngestURL(baseURL string) string {
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return strings.TrimSuffix(baseURL, "/")
	}

	hostname := parsedURL.Hostname()
	if strings.HasPrefix(hostname, "api.") {
		hostname = "ingress." + strings.TrimPrefix(hostname, "api.")
	}

	if port := parsedURL.Port(); port != "" {
		parsedURL.Host = fmt.Sprintf("%s:%s", hostname, port)
	} else {
		parsedURL.Host = hostname
	}

	parsedURL.Path = ""
	parsedURL.RawPath = ""
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	return strings.TrimSuffix(parsedURL.String(), "/")
}

// withDatasetQuery appends the configured dataset query parameter to a request URL.
func (c *Client) withDatasetQuery(requestURL string) (string, error) {
	operation := "dash0 client: apply dataset query"

	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("%s: parse request URL: %w", operation, err)
	}

	query := parsedURL.Query()
	query.Set("dataset", c.Dataset)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// execRequest executes an HTTP request with authentication and response validation.
func (c *Client) execRequest(operation, method, requestURL string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", operation, err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: execute request: %w", operation, err)
	}
	defer res.Body.Close()

	limitedReader := io.LimitReader(res.Body, MaxResponseSize)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("%s: read response: %w", operation, err)
	}

	if len(responseBody) >= MaxResponseSize {
		return nil, fmt.Errorf("%s: response too large: exceeds maximum size of %d bytes", operation, MaxResponseSize)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, &HTTPRequestStatusError{
			Operation:  operation,
			StatusCode: res.StatusCode,
			Body:       string(responseBody),
		}
	}

	return responseBody, nil
}

// ExecutePrometheusInstantQuery runs an instant PromQL query against Dash0.
func (c *Client) ExecutePrometheusInstantQuery(promQLQuery, dataset string) (map[string]any, error) {
	operation := "dash0 client: execute prometheus instant query"
	requestURL := fmt.Sprintf("%s/api/prometheus/api/v1/query", c.BaseURL)

	data := url.Values{}
	data.Set("dataset", dataset)
	data.Set("query", promQLQuery)

	responseBody, err := c.execRequest(operation, http.MethodPost, requestURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	var response PrometheusResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("%s: parse response: %w", operation, err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("%s: prometheus query returned non-success status: %s", operation, response.Status)
	}

	return map[string]any{
		"status": response.Status,
		"data":   response.Data,
	}, nil
}

// ExecutePrometheusRangeQuery runs a range PromQL query against Dash0.
func (c *Client) ExecutePrometheusRangeQuery(promQLQuery, dataset, start, end, step string) (map[string]any, error) {
	operation := "dash0 client: execute prometheus range query"
	requestURL := fmt.Sprintf("%s/api/prometheus/api/v1/query_range", c.BaseURL)

	data := url.Values{}
	data.Set("dataset", dataset)
	data.Set("query", promQLQuery)
	data.Set("start", start)
	data.Set("end", end)
	data.Set("step", step)

	responseBody, err := c.execRequest(operation, http.MethodPost, requestURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	var response PrometheusResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("%s: parse response: %w", operation, err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("%s: prometheus query returned non-success status: %s", operation, response.Status)
	}

	return map[string]any{
		"status": response.Status,
		"data":   response.Data,
	}, nil
}

// ListCheckRules lists Dash0 check rules for resource pickers and lookups.
func (c *Client) ListCheckRules() ([]CheckRule, error) {
	operation := "dash0 client: list check rules"
	requestURL := fmt.Sprintf("%s/api/alerting/check-rules", c.BaseURL)
	requestURL, err := c.withDatasetQuery(requestURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	responseBody, err := c.execRequest(operation, http.MethodGet, requestURL, nil, "")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	rules, err := parseCheckRules(responseBody)
	if err != nil {
		return nil, fmt.Errorf("%s: parse check rules response: %w", operation, err)
	}

	return rules, nil
}

// ListSyntheticChecks lists Dash0 synthetic checks for resource pickers and lookups.
func (c *Client) ListSyntheticChecks() ([]SyntheticCheck, error) {
	operation := "dash0 client: list synthetic checks"
	requestURL := fmt.Sprintf("%s/api/synthetic-checks", c.BaseURL)
	requestURL, err := c.withDatasetQuery(requestURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	responseBody, err := c.execRequest(operation, http.MethodGet, requestURL, nil, "")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	checks, err := parseSyntheticChecks(responseBody)
	if err != nil {
		return nil, fmt.Errorf("%s: parse synthetic checks response: %w", operation, err)
	}

	return checks, nil
}

// SendLogEvents sends OTLP log batches to Dash0 ingestion endpoint.
func (c *Client) SendLogEvents(request OTLPLogsRequest) (map[string]any, error) {
	operation := "dash0 client: send log events"
	requestURL := fmt.Sprintf("%s/v1/logs", c.LogsIngestURL)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", operation, err)
	}

	responseBody, err := c.execRequest(operation, http.MethodPost, requestURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	parsed, err := parseJSONResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("%s: parse response: %w", operation, err)
	}

	return parsed, nil
}

// GetCheckDetails fetches check context by failed-check ID with check-rules fallback.
func (c *Client) GetCheckDetails(checkID string, includeHistory bool) (map[string]any, error) {
	operation := "dash0 client: get check details"
	trimmedCheckID := strings.TrimSpace(checkID)
	if trimmedCheckID == "" {
		return nil, fmt.Errorf("%s: check id is required", operation)
	}

	querySuffix := ""
	if includeHistory {
		querySuffix = "?include_history=true"
	}

	escapedID := url.PathEscape(trimmedCheckID)

	requestURL := fmt.Sprintf("%s/api/alerting/failed-checks/%s%s", c.BaseURL, escapedID, querySuffix)
	requestURL, err := c.withDatasetQuery(requestURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	responseBody, err := c.execRequest(operation, http.MethodGet, requestURL, nil, "")
	if err != nil {
		var statusErr *HTTPRequestStatusError
		if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
			// Fallback for organizations where check details are exposed via check-rules endpoint.
			fallbackURL := fmt.Sprintf("%s/api/alerting/check-rules/%s%s", c.BaseURL, escapedID, querySuffix)
			fallbackURL, err = c.withDatasetQuery(fallbackURL)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", operation, err)
			}
			responseBody, err = c.execRequest(operation, http.MethodGet, fallbackURL, nil, "")
			if err != nil {
				return nil, fmt.Errorf("%s: fallback check-rules lookup failed: %w", operation, err)
			}
		} else {
			return nil, fmt.Errorf("%s: %w", operation, err)
		}
	}

	parsed, err := parseJSONResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("%s: parse response: %w", operation, err)
	}

	if _, ok := parsed["checkId"]; !ok {
		parsed["checkId"] = trimmedCheckID
	}

	return parsed, nil
}

// UpsertSyntheticCheck creates or updates a synthetic check by origin/id.
func (c *Client) UpsertSyntheticCheck(originOrID string, specification map[string]any) (map[string]any, error) {
	operation := "dash0 client: upsert synthetic check"
	trimmedOriginOrID := strings.TrimSpace(originOrID)
	if trimmedOriginOrID == "" {
		return nil, fmt.Errorf("%s: origin/id is required", operation)
	}

	requestURL := fmt.Sprintf("%s/api/synthetic-checks/%s", c.BaseURL, url.PathEscape(trimmedOriginOrID))
	requestURL, err := c.withDatasetQuery(requestURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	body, err := json.Marshal(specification)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", operation, err)
	}

	responseBody, err := c.execRequest(operation, http.MethodPut, requestURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	parsed, err := parseJSONResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("%s: parse response: %w", operation, err)
	}

	if _, ok := parsed["originOrId"]; !ok {
		parsed["originOrId"] = trimmedOriginOrID
	}

	return parsed, nil
}

// UpsertCheckRule creates or updates a check rule by origin/id.
func (c *Client) UpsertCheckRule(originOrID string, specification map[string]any) (map[string]any, error) {
	operation := "dash0 client: upsert check rule"
	trimmedOriginOrID := strings.TrimSpace(originOrID)
	if trimmedOriginOrID == "" {
		return nil, fmt.Errorf("%s: origin/id is required", operation)
	}

	requestURL := fmt.Sprintf("%s/api/alerting/check-rules/%s", c.BaseURL, url.PathEscape(trimmedOriginOrID))
	requestURL, err := c.withDatasetQuery(requestURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	body, err := json.Marshal(specification)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", operation, err)
	}

	responseBody, err := c.execRequest(operation, http.MethodPut, requestURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	parsed, err := parseJSONResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("%s: parse response: %w", operation, err)
	}

	if _, ok := parsed["originOrId"]; !ok {
		parsed["originOrId"] = trimmedOriginOrID
	}

	return parsed, nil
}

// parseJSONResponse normalizes object or array JSON responses into a map.
func parseJSONResponse(responseBody []byte) (map[string]any, error) {
	trimmedBody := strings.TrimSpace(string(responseBody))
	if trimmedBody == "" {
		return map[string]any{}, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal(responseBody, &parsed); err == nil {
		return parsed, nil
	}

	var parsedArray []any
	if err := json.Unmarshal(responseBody, &parsedArray); err == nil {
		return map[string]any{"items": parsedArray}, nil
	}

	return nil, fmt.Errorf("unexpected response payload shape")
}

// parseCheckRules decodes check rules from multiple supported Dash0 response shapes.
func parseCheckRules(responseBody []byte) ([]CheckRule, error) {
	var ids []string
	if err := json.Unmarshal(responseBody, &ids); err == nil {
		rules := make([]CheckRule, 0, len(ids))
		for _, id := range ids {
			trimmed := strings.TrimSpace(id)
			if trimmed == "" {
				continue
			}
			rules = append(rules, CheckRule{ID: trimmed, Name: trimmed, Origin: trimmed})
		}
		return rules, nil
	}

	var rules []CheckRule
	if err := json.Unmarshal(responseBody, &rules); err == nil {
		normalizeCheckRules(rules)
		return rules, nil
	}

	var wrapped map[string]any
	if err := json.Unmarshal(responseBody, &wrapped); err != nil {
		return nil, err
	}

	items, ok := wrapped["items"].([]any)
	if !ok {
		if data, ok := wrapped["data"].([]any); ok {
			items = data
		}
	}
	if len(items) == 0 {
		return []CheckRule{}, nil
	}

	rules = convertAnySliceToCheckRules(items)
	normalizeCheckRules(rules)
	return rules, nil
}

// parseSyntheticChecks decodes synthetic checks from supported response shapes.
func parseSyntheticChecks(responseBody []byte) ([]SyntheticCheck, error) {
	var ids []string
	if err := json.Unmarshal(responseBody, &ids); err == nil {
		checks := make([]SyntheticCheck, 0, len(ids))
		for _, id := range ids {
			trimmed := strings.TrimSpace(id)
			if trimmed == "" {
				continue
			}
			checks = append(checks, SyntheticCheck{ID: trimmed, Name: trimmed, Origin: trimmed})
		}
		return checks, nil
	}

	var checks []SyntheticCheck
	if err := json.Unmarshal(responseBody, &checks); err == nil {
		normalizeSyntheticChecks(checks)
		return checks, nil
	}

	var wrapped map[string]any
	if err := json.Unmarshal(responseBody, &wrapped); err != nil {
		return nil, err
	}

	items, ok := wrapped["items"].([]any)
	if !ok {
		if data, ok := wrapped["data"].([]any); ok {
			items = data
		}
	}
	if len(items) == 0 {
		return []SyntheticCheck{}, nil
	}

	checks = convertAnySliceToSyntheticChecks(items)
	normalizeSyntheticChecks(checks)
	return checks, nil
}

// normalizeCheckRules fills missing check rule identifiers and names.
func normalizeCheckRules(rules []CheckRule) {
	for index := range rules {
		if rules[index].ID == "" {
			rules[index].ID = rules[index].Origin
		}
		if rules[index].Origin == "" {
			rules[index].Origin = rules[index].ID
		}
		if rules[index].Name == "" {
			if rules[index].ID != "" {
				rules[index].Name = rules[index].ID
			} else {
				rules[index].Name = rules[index].Origin
			}
		}
	}
}

// normalizeSyntheticChecks fills missing synthetic check identifiers and names.
func normalizeSyntheticChecks(checks []SyntheticCheck) {
	for index := range checks {
		if checks[index].ID == "" {
			checks[index].ID = checks[index].Origin
		}
		if checks[index].Origin == "" {
			checks[index].Origin = checks[index].ID
		}
		if checks[index].Name == "" {
			if checks[index].ID != "" {
				checks[index].Name = checks[index].ID
			} else {
				checks[index].Name = checks[index].Origin
			}
		}
	}
}

// convertAnySliceToCheckRules maps generic list entries into CheckRule values.
func convertAnySliceToCheckRules(values []any) []CheckRule {
	rules := make([]CheckRule, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}

		rule := CheckRule{
			ID:     firstNonEmptyString(item["id"], item["checkRuleId"], item["ruleId"]),
			Origin: firstNonEmptyString(item["origin"]),
			Name:   firstNonEmptyString(item["name"], item["label"], item["title"]),
		}

		if rule.ID == "" && rule.Origin == "" {
			continue
		}

		rules = append(rules, rule)
	}
	return rules
}

// convertAnySliceToSyntheticChecks maps generic list entries into SyntheticCheck values.
func convertAnySliceToSyntheticChecks(values []any) []SyntheticCheck {
	checks := make([]SyntheticCheck, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}

		check := SyntheticCheck{
			ID:     firstNonEmptyString(item["id"], item["syntheticCheckId"]),
			Origin: firstNonEmptyString(item["origin"]),
			Name:   firstNonEmptyString(item["name"], item["label"], item["title"]),
		}

		if check.ID == "" && check.Origin == "" {
			continue
		}

		checks = append(checks, check)
	}
	return checks
}

// firstNonEmptyString returns the first non-empty string in the provided values.
func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
