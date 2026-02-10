package dash0

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
	// MaxResponseSize limits the size of Prometheus API responses to prevent excessive memory usage
	// 1MB should be sufficient for most Prometheus queries while preventing abuse
	MaxResponseSize = 1 * 1024 * 1024 // 1MB
)

type Client struct {
	Token         string
	BaseURL       string
	LogsIngestURL string
	Dataset       string
	http          core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting api token: %v", err)
	}

	baseURL := ""
	baseURLConfig, err := ctx.GetConfig("baseURL")
	if err == nil && baseURLConfig != nil && len(baseURLConfig) > 0 {
		baseURL = strings.TrimSuffix(string(baseURLConfig), "/")
	}

	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is required for Dash0 Cloud. Find your API URL in Dash0 dashboard under Organization Settings > Endpoints Reference")
	}

	// Strip /api/prometheus if user included it in the base URL
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
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("error parsing request URL: %v", err)
	}

	query := parsedURL.Query()
	query.Set("dataset", c.Dataset)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func (c *Client) execRequest(method, url string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	// Limit response size to prevent excessive memory usage
	limitedReader := io.LimitReader(res.Body, MaxResponseSize)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	// Check if we hit the limit (response was truncated)
	if len(responseBody) >= MaxResponseSize {
		return nil, fmt.Errorf("response too large: exceeds maximum size of %d bytes", MaxResponseSize)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type PrometheusResponse struct {
	Status string                 `json:"status"`
	Data   PrometheusResponseData `json:"data"`
}

type PrometheusResponseData struct {
	ResultType string                  `json:"resultType"`
	Result     []PrometheusQueryResult `json:"result"`
}

type PrometheusQueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value,omitempty"`  // For instant queries: [timestamp, value]
	Values [][]interface{}   `json:"values,omitempty"` // For range queries: [[timestamp, value], ...]
}

func (c *Client) ExecutePrometheusInstantQuery(promQLQuery, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/prometheus/api/v1/query", c.BaseURL)

	data := url.Values{}
	data.Set("dataset", dataset)
	data.Set("query", promQLQuery)

	responseBody, err := c.execRequest(http.MethodPost, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	var response PrometheusResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed with status: %s", response.Status)
	}

	return map[string]any{
		"status": response.Status,
		"data":   response.Data,
	}, nil
}

func (c *Client) ExecutePrometheusRangeQuery(promQLQuery, dataset, start, end, step string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/prometheus/api/v1/query_range", c.BaseURL)

	data := url.Values{}
	data.Set("dataset", dataset)
	data.Set("query", promQLQuery)
	data.Set("start", start)
	data.Set("end", end)
	data.Set("step", step)

	responseBody, err := c.execRequest(http.MethodPost, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	var response PrometheusResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed with status: %s", response.Status)
	}

	return map[string]any{
		"status": response.Status,
		"data":   response.Data,
	}, nil
}

type CheckRule struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

func (c *Client) ListCheckRules() ([]CheckRule, error) {
	apiURL := fmt.Sprintf("%s/api/alerting/check-rules", c.BaseURL)
	requestURL, err := c.withDatasetQuery(apiURL)
	if err != nil {
		return nil, err
	}

	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil, "")
	if err != nil {
		return nil, err
	}

	// The API might return either:
	// 1. A list of strings (IDs only)
	// 2. A list of objects with id and name
	var checkRules []CheckRule

	// Try parsing as list of strings first
	var stringList []string
	if err := json.Unmarshal(responseBody, &stringList); err == nil {
		// If successful, convert strings to CheckRule objects
		checkRules = make([]CheckRule, len(stringList))
		for i, id := range stringList {
			checkRules[i] = CheckRule{
				ID:   id,
				Name: id, // Use ID as name if name is not available
			}
		}
		return checkRules, nil
	}

	// Try parsing as list of CheckRule objects
	if err := json.Unmarshal(responseBody, &checkRules); err != nil {
		return nil, fmt.Errorf("error parsing check rules response: %v", err)
	}

	// If names are empty, use ID as name
	for i := range checkRules {
		if checkRules[i].Name == "" {
			checkRules[i].Name = checkRules[i].ID
		}
	}

	return checkRules, nil
}

// ListSyntheticChecks lists Dash0 synthetic checks for resource pickers and lookups.
func (c *Client) ListSyntheticChecks() ([]SyntheticCheck, error) {
	apiURL := fmt.Sprintf("%s/api/synthetic-checks", c.BaseURL)
	requestURL, err := c.withDatasetQuery(apiURL)
	if err != nil {
		return nil, err
	}

	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil, "")
	if err != nil {
		return nil, err
	}

	var checks []SyntheticCheck

	var stringList []string
	if err := json.Unmarshal(responseBody, &stringList); err == nil {
		checks = make([]SyntheticCheck, 0, len(stringList))
		for _, id := range stringList {
			trimmed := strings.TrimSpace(id)
			if trimmed == "" {
				continue
			}
			checks = append(checks, SyntheticCheck{
				ID:     trimmed,
				Name:   trimmed,
				Origin: trimmed,
			})
		}
		return checks, nil
	}

	if err := json.Unmarshal(responseBody, &checks); err != nil {
		return nil, fmt.Errorf("error parsing synthetic checks response: %v", err)
	}

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

	return checks, nil
}

// SendLogEvents sends OTLP log batches to Dash0 ingestion endpoint.
func (c *Client) SendLogEvents(request OTLPLogsRequest) (map[string]any, error) {
	requestURL := fmt.Sprintf("%s/v1/logs", c.LogsIngestURL)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling logs request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, requestURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}

	parsed, err := parseJSONResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("error parsing send log event response: %v", err)
	}

	return parsed, nil
}

// GetCheckDetails fetches check context by failed-check ID with check-rules fallback.
func (c *Client) GetCheckDetails(checkID string, includeHistory bool) (map[string]any, error) {
	trimmedCheckID := strings.TrimSpace(checkID)
	if trimmedCheckID == "" {
		return nil, fmt.Errorf("check id is required")
	}

	querySuffix := ""
	if includeHistory {
		querySuffix = "?include_history=true"
	}

	escapedID := url.PathEscape(trimmedCheckID)
	requestURL := fmt.Sprintf("%s/api/alerting/failed-checks/%s%s", c.BaseURL, escapedID, querySuffix)

	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil, "")
	if err != nil {
		if strings.Contains(err.Error(), "request got 404 code") {
			fallbackURL := fmt.Sprintf("%s/api/alerting/check-rules/%s%s", c.BaseURL, escapedID, querySuffix)
			responseBody, err = c.execRequest(http.MethodGet, fallbackURL, nil, "")
			if err != nil {
				return nil, fmt.Errorf("fallback check-rules lookup failed: %v", err)
			}
		} else {
			return nil, err
		}
	}

	parsed, err := parseJSONResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("error parsing check details response: %v", err)
	}

	if _, ok := parsed["checkId"]; !ok {
		parsed["checkId"] = trimmedCheckID
	}

	return parsed, nil
}

// UpsertSyntheticCheck creates or updates a synthetic check by origin/id.
func (c *Client) UpsertSyntheticCheck(originOrID string, specification map[string]any) (map[string]any, error) {
	trimmedOriginOrID := strings.TrimSpace(originOrID)
	if trimmedOriginOrID == "" {
		return nil, fmt.Errorf("origin/id is required")
	}

	requestURL := fmt.Sprintf("%s/api/synthetic-checks/%s", c.BaseURL, url.PathEscape(trimmedOriginOrID))
	requestURL, err := c.withDatasetQuery(requestURL)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(specification)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, requestURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}

	parsed, err := parseJSONResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("error parsing upsert synthetic check response: %v", err)
	}

	if _, ok := parsed["originOrId"]; !ok {
		parsed["originOrId"] = trimmedOriginOrID
	}

	return parsed, nil
}

// UpsertCheckRule creates or updates a check rule by origin/id.
func (c *Client) UpsertCheckRule(originOrID string, specification map[string]any) (map[string]any, error) {
	trimmedOriginOrID := strings.TrimSpace(originOrID)
	if trimmedOriginOrID == "" {
		return nil, fmt.Errorf("origin/id is required")
	}

	requestURL := fmt.Sprintf("%s/api/alerting/check-rules/%s", c.BaseURL, url.PathEscape(trimmedOriginOrID))
	requestURL, err := c.withDatasetQuery(requestURL)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(specification)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, requestURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}

	parsed, err := parseJSONResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("error parsing upsert check rule response: %v", err)
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
