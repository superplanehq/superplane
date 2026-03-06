package dash0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// MaxResponseSize limits the size of Prometheus API responses to prevent excessive memory usage
	// 1MB should be sufficient for most Prometheus queries while preventing abuse
	MaxResponseSize = 1 * 1024 * 1024 // 1MB
	// Dash0DatasetHeader is the HTTP header name used by Dash0 for dataset routing.
	Dash0DatasetHeader = "Dash0-Dataset"
)

type Client struct {
	Token   string
	BaseURL string
	http    core.HTTPContext
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

	return &Client{
		Token:   string(apiToken),
		BaseURL: baseURL,
		http:    http,
	}, nil
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
	Value  []any             `json:"value,omitempty"`  // For instant queries: [timestamp, value]
	Values [][]any           `json:"values,omitempty"` // For range queries: [[timestamp, value], ...]
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

	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil, "")
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

// CheckRuleRequest represents the request body for creating or updating a check rule.
type CheckRuleRequest struct {
	Dataset       string               `json:"dataset,omitempty"`
	ID            string               `json:"id,omitempty"`
	Name          string               `json:"name"`
	Expression    string               `json:"expression"`
	Thresholds    *CheckRuleThresholds `json:"thresholds,omitempty"`
	Summary       string               `json:"summary,omitempty"`
	Description   string               `json:"description,omitempty"`
	Interval      string               `json:"interval,omitempty"`
	For           string               `json:"for,omitempty"`
	KeepFiringFor string               `json:"keepFiringFor,omitempty"`
	Labels        map[string]string    `json:"labels,omitempty"`
	Annotations   map[string]string    `json:"annotations,omitempty"`
	Enabled       *bool                `json:"enabled,omitempty"`
}

// CheckRuleThresholds represents the degraded and critical thresholds for a check rule.
type CheckRuleThresholds struct {
	Degraded *float64 `json:"degraded,omitempty"`
	Critical *float64 `json:"critical,omitempty"`
}

// CreateCheckRule creates a new check rule (Prometheus alert rule) in Dash0.
func (c *Client) CreateCheckRule(request CheckRuleRequest, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/alerting/check-rules?dataset=%s", c.BaseURL, url.QueryEscape(dataset))

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, apiURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// GetCheckRule retrieves a specific check rule by its origin or ID.
func (c *Client) GetCheckRule(originOrID string, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/alerting/check-rules/%s?dataset=%s", c.BaseURL, url.PathEscape(originOrID), url.QueryEscape(dataset))

	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil, "")
	if err != nil {
		return nil, err
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// UpdateCheckRule updates an existing check rule by its origin or ID.
func (c *Client) UpdateCheckRule(originOrID string, request CheckRuleRequest, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/alerting/check-rules/%s?dataset=%s", c.BaseURL, url.PathEscape(originOrID), url.QueryEscape(dataset))

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, apiURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// DeleteCheckRule deletes a specific check rule by its origin or ID.
func (c *Client) DeleteCheckRule(originOrID string, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/alerting/check-rules/%s?dataset=%s", c.BaseURL, url.PathEscape(originOrID), url.QueryEscape(dataset))

	responseBody, err := c.execRequest(http.MethodDelete, apiURL, nil, "")
	if err != nil {
		return nil, err
	}

	// DELETE might return empty body on success
	if len(responseBody) == 0 {
		return map[string]any{"deleted": true, "id": originOrID}, nil
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// SyntheticCheckAssertion represents a single assertion in a synthetic check.
type SyntheticCheckAssertion struct {
	Kind string         `json:"kind"`
	Spec map[string]any `json:"spec"`
}

// SyntheticCheckHeader represents an HTTP header key-value pair.
type SyntheticCheckHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SyntheticCheckRequest represents the full request payload for creating a synthetic check.
// Matches the Dash0 API envelope: kind + metadata + spec.
type SyntheticCheckRequest struct {
	Kind     string                     `json:"kind"`
	Metadata SyntheticCheckMetadata     `json:"metadata"`
	Spec     SyntheticCheckTopLevelSpec `json:"spec"`
}

// SyntheticCheckMetadata contains the check name and labels.
type SyntheticCheckMetadata struct {
	Name   string         `json:"name"`
	Labels map[string]any `json:"labels"`
}

// SyntheticCheckTopLevelSpec wraps the plugin, schedule, retries, and enabled flag.
type SyntheticCheckTopLevelSpec struct {
	Enabled  bool                   `json:"enabled"`
	Schedule SyntheticCheckSchedule `json:"schedule"`
	Plugin   SyntheticCheckPlugin   `json:"plugin"`
}

// SyntheticCheckPlugin contains the check type, display metadata, and specification.
type SyntheticCheckPlugin struct {
	Display SyntheticCheckDisplay    `json:"display"`
	Kind    string                   `json:"kind"`
	Spec    SyntheticCheckPluginSpec `json:"spec"`
}

// SyntheticCheckDisplay contains the display name for a synthetic check.
type SyntheticCheckDisplay struct {
	Name string `json:"name"`
}

// SyntheticCheckPluginSpec contains the HTTP request, assertions, and retries for a synthetic check.
type SyntheticCheckPluginSpec struct {
	Request    SyntheticCheckHTTPRequest `json:"request"`
	Assertions SyntheticCheckAssertions  `json:"assertions"`
	Retries    SyntheticCheckRetries     `json:"retries"`
}

// SyntheticCheckHTTPRequest defines the HTTP request configuration.
type SyntheticCheckHTTPRequest struct {
	Method          string                 `json:"method"`
	URL             string                 `json:"url"`
	Headers         []SyntheticCheckHeader `json:"headers"`
	QueryParameters []any                  `json:"queryParameters"`
	Body            *string                `json:"body,omitempty"`
	Redirects       string                 `json:"redirects"`
	TLS             SyntheticCheckTLS      `json:"tls"`
	Tracing         SyntheticCheckTracing  `json:"tracing"`
}

// SyntheticCheckTLS holds TLS configuration.
type SyntheticCheckTLS struct {
	AllowInsecure bool `json:"allowInsecure"`
}

// SyntheticCheckTracing holds tracing configuration.
type SyntheticCheckTracing struct {
	AddTracingHeaders bool `json:"addTracingHeaders"`
}

// SyntheticCheckAssertions groups critical and degraded assertions.
type SyntheticCheckAssertions struct {
	CriticalAssertions []SyntheticCheckAssertion `json:"criticalAssertions"`
	DegradedAssertions []SyntheticCheckAssertion `json:"degradedAssertions"`
}

// SyntheticCheckSchedule defines how often and where a check runs.
type SyntheticCheckSchedule struct {
	Interval  string   `json:"interval"`
	Locations []string `json:"locations"`
	Strategy  string   `json:"strategy"`
}

// SyntheticCheckRetries defines retry behavior for failed checks.
type SyntheticCheckRetries struct {
	Kind string                    `json:"kind"`
	Spec SyntheticCheckRetriesSpec `json:"spec"`
}

// SyntheticCheckRetriesSpec contains the retry parameters.
type SyntheticCheckRetriesSpec struct {
	Attempts int    `json:"attempts"`
	Delay    string `json:"delay"`
}

func (c *Client) ListSyntheticChecks(dataset string) ([]map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/synthetic-checks?dataset=%s", c.BaseURL, url.QueryEscape(dataset))

	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil, "")
	if err != nil {
		return nil, err
	}

	// Try bare array first
	var items []map[string]any
	if err := json.Unmarshal(responseBody, &items); err == nil {
		return items, nil
	}

	// Fall back to wrapped object e.g. {"items": [...]}
	var wrapped struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(responseBody, &wrapped); err != nil {
		return nil, fmt.Errorf("error parsing synthetic checks response: %v (body: %s)", err, string(responseBody))
	}

	return wrapped.Items, nil
}

func (c *Client) CreateSyntheticCheck(request SyntheticCheckRequest, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/synthetic-checks?dataset=%s", c.BaseURL, url.QueryEscape(dataset))

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, apiURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// DeleteSyntheticCheck deletes a synthetic check by ID (DELETE).
func (c *Client) DeleteSyntheticCheck(checkID string, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/synthetic-checks/%s?dataset=%s", c.BaseURL, url.PathEscape(checkID), url.QueryEscape(dataset))

	responseBody, err := c.execRequest(http.MethodDelete, apiURL, nil, "")
	if err != nil {
		return nil, err
	}

	// DELETE may return 204 No Content with empty body
	if len(responseBody) == 0 {
		return map[string]any{"deleted": true, "id": checkID}, nil
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// GetSyntheticCheck retrieves a single synthetic check by ID (GET).
func (c *Client) GetSyntheticCheck(checkID string, dataset string) (*SyntheticCheckResponse, error) {
	apiURL := fmt.Sprintf("%s/api/synthetic-checks/%s?dataset=%s", c.BaseURL, url.PathEscape(checkID), url.QueryEscape(dataset))

	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil, "")
	if err != nil {
		return nil, err
	}

	var response SyntheticCheckResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response, nil
}

// LogRecord represents a structured log record to be sent to Dash0 via OTLP HTTP ingestion.
type LogRecord struct {
	SeverityText string            `json:"severityText"`
	Body         string            `json:"body"`
	EventName    string            `json:"eventName"`
	ServiceName  string            `json:"serviceName,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// SendLogRecord sends a log record to Dash0 via OTLP HTTP ingestion (POST).
// It builds the full OTLP ExportLogsServiceRequest payload from the given LogRecord.
func (c *Client) SendLogRecord(dataset string, record LogRecord) (map[string]any, error) {
	otlpBaseURL, err := deriveOTLPEndpoint(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("error deriving OTLP endpoint: %v", err)
	}

	apiURL := fmt.Sprintf("%s/v1/logs", otlpBaseURL)

	otlpPayload := buildOTLPLogPayload(record)
	jsonBody, err := json.Marshal(otlpPayload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling log body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	if dataset != "" && dataset != "default" {
		req.Header.Set(Dash0DatasetHeader, dataset)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	limitedReader := io.LimitReader(res.Body, MaxResponseSize)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if len(responseBody) >= MaxResponseSize {
		return nil, fmt.Errorf("response too large: exceeds maximum size of %d bytes", MaxResponseSize)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return map[string]any{"sent": true}, nil
}

// deriveOTLPEndpoint derives the OTLP HTTP ingress endpoint from the Dash0 API base URL.
// The API base URL has the format: https://api.{region}.aws.dash0.com
// The OTLP HTTP ingress endpoint has the format: https://ingress.{region}.aws.dash0.com:4318
func deriveOTLPEndpoint(apiBaseURL string) (string, error) {
	parsed, err := url.Parse(apiBaseURL)
	if err != nil {
		return "", fmt.Errorf("error parsing API base URL: %v", err)
	}

	host := parsed.Hostname()
	if !strings.HasPrefix(host, "api.") {
		return "", fmt.Errorf("cannot derive OTLP endpoint: API base URL host %q does not start with 'api.'", host)
	}

	ingressHost := "ingress." + strings.TrimPrefix(host, "api.")
	return fmt.Sprintf("https://%s:4318", ingressHost), nil
}

// buildOTLPLogPayload constructs a full OTLP ExportLogsServiceRequest JSON structure from a LogRecord.
func buildOTLPLogPayload(record LogRecord) map[string]any {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)

	attributes := []map[string]any{}
	for key, value := range record.Attributes {
		attributes = append(attributes, map[string]any{
			"key": key,
			"value": map[string]any{
				"stringValue": value,
			},
		})
	}

	if record.EventName != "" {
		attributes = append(attributes, map[string]any{
			"key": "event.name",
			"value": map[string]any{
				"stringValue": record.EventName,
			},
		})
	}

	severityNumber := severityTextToNumber(record.SeverityText)

	// Build resource attributes (service.name goes here per OTLP spec)
	resourceAttributes := []map[string]any{}
	if record.ServiceName != "" {
		resourceAttributes = append(resourceAttributes, map[string]any{
			"key": "service.name",
			"value": map[string]any{
				"stringValue": record.ServiceName,
			},
		})
	}

	return map[string]any{
		"resourceLogs": []map[string]any{
			{
				"resource": map[string]any{
					"attributes": resourceAttributes,
				},
				"scopeLogs": []map[string]any{
					{
						"logRecords": []map[string]any{
							{
								"timeUnixNano":   timestamp,
								"severityNumber": severityNumber,
								"severityText":   record.SeverityText,
								"body": map[string]any{
									"stringValue": record.Body,
								},
								"attributes": attributes,
							},
						},
					},
				},
			},
		},
	}
}

// severityTextToNumber maps OTLP severity text to its corresponding severity number.
func severityTextToNumber(severityText string) int {
	switch strings.ToUpper(severityText) {
	case "TRACE":
		return 1
	case "DEBUG":
		return 5
	case "INFO":
		return 9
	case "WARN":
		return 13
	case "ERROR":
		return 17
	case "FATAL":
		return 21
	default:
		return 9 // INFO
	}
}

// UpdateSyntheticCheck updates an existing synthetic check by ID (PUT).
// The check ID is typically from metadata.labels["dash0.com/id"] in a create response.
func (c *Client) UpdateSyntheticCheck(checkID string, request SyntheticCheckRequest, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/synthetic-checks/%s?dataset=%s", c.BaseURL, url.PathEscape(checkID), url.QueryEscape(dataset))

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, apiURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

// SyntheticCheckResponse represents the response from the Dash0 GET Synthetic Check API.
type SyntheticCheckResponse struct {
	Kind     string                         `json:"kind"`
	Metadata SyntheticCheckResponseMetadata `json:"metadata"`
	Spec     SyntheticCheckResponseSpec     `json:"spec"`
}

type SyntheticCheckResponseMetadata struct {
	Annotations map[string]any    `json:"annotations"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	Name        string            `json:"name"`
}

type SyntheticCheckResponseSpec struct {
	Display       SyntheticCheckDisplay         `json:"display"`
	Enabled       bool                          `json:"enabled"`
	Labels        map[string]any                `json:"labels"`
	Notifications SyntheticCheckNotifications   `json:"notifications"`
	Plugin        SyntheticCheckResponsePlugin  `json:"plugin"`
	Retries       SyntheticCheckResponseRetries `json:"retries"`
	Schedule      SyntheticCheckSchedule        `json:"schedule"`
}

type SyntheticCheckNotifications struct {
	Channels             []string `json:"channels"`
	OnlyCriticalChannels []string `json:"onlyCriticalChannels"`
}

type SyntheticCheckResponsePlugin struct {
	Kind string                           `json:"kind"`
	Spec SyntheticCheckResponsePluginSpec `json:"spec"`
}

type SyntheticCheckResponsePluginSpec struct {
	Assertions SyntheticCheckAssertions  `json:"assertions"`
	Request    SyntheticCheckHTTPRequest `json:"request"`
}

type SyntheticCheckResponseRetries struct {
	Kind string         `json:"kind"`
	Spec map[string]any `json:"spec"`
}
