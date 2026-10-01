package datadog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	maxErrorTrackingSearchLimit = 100
	errorTrackingSearchWindow   = 30 * 24 * time.Hour
)

// ErrorTrackingForbiddenMessage is the user-facing text for ErrErrorTrackingForbidden.
const ErrorTrackingForbiddenMessage = "Datadog refused the request. The application key needs error_tracking_read."

// ErrErrorTrackingForbidden is returned when Datadog refuses an Error Tracking
// call, usually because a restricted application key lacks error_tracking_read.
var ErrErrorTrackingForbidden = errors.New(
	"datadog refused the request: the application key needs error_tracking_read",
)

// APIError is a non-2xx response from Datadog.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if msg := strings.TrimSpace(e.Body); msg != "" {
		return msg
	}
	return fmt.Sprintf("Datadog request failed with status %d", e.StatusCode)
}

type Client struct {
	APIKey          string
	AppKey          string
	Site            string
	BaseURL         string
	http            core.HTTPContext
	requestDeadline time.Time
}

// SetRequestDeadline stops later Datadog calls at deadline. The zero time
// clears the limit.
func (c *Client) SetRequestDeadline(deadline time.Time) {
	c.requestDeadline = deadline
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error getting apiKey: %v", err)
	}

	appKey, err := ctx.GetConfig("appKey")
	if err != nil {
		return nil, fmt.Errorf("error getting appKey: %v", err)
	}

	site, err := ctx.GetConfig("site")
	if err != nil {
		return nil, fmt.Errorf("error getting site: %v", err)
	}

	siteValue := strings.TrimSpace(string(site))
	return &Client{
		APIKey:  string(apiKey),
		AppKey:  string(appKey),
		Site:    siteValue,
		BaseURL: fmt.Sprintf("https://api.%s", siteValue),
		http:    httpCtx,
	}, nil
}

func (c *Client) execRequest(method, requestURL string, body io.Reader) ([]byte, error) {
	ctx := context.Background()
	if !c.requestDeadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, c.requestDeadline)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("DD-API-KEY", c.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", c.AppKey)

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
		return nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return responseBody, nil
}

// ValidateCredentials verifies that the API and Application keys are valid
// by calling the Datadog validate endpoint.
func (c *Client) ValidateCredentials() error {
	requestURL := fmt.Sprintf("%s/api/v1/validate", c.BaseURL)
	_, err := c.execRequest(http.MethodGet, requestURL, nil)
	return err
}

// CreateEventRequest represents the request payload for creating a Datadog event.
type CreateEventRequest struct {
	Title     string   `json:"title"`
	Text      string   `json:"text"`
	AlertType string   `json:"alert_type,omitempty"`
	Priority  string   `json:"priority,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// Event represents a Datadog event response.
type Event struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	Text         string   `json:"text"`
	DateHappened int64    `json:"date_happened"`
	AlertType    string   `json:"alert_type"`
	Priority     string   `json:"priority"`
	Tags         []string `json:"tags"`
	URL          string   `json:"url"`
}

// CreateEventResponse represents the response from creating an event.
type CreateEventResponse struct {
	Event  Event  `json:"event"`
	Status string `json:"status"`
}

// CreateEvent creates a new event in Datadog.
func (c *Client) CreateEvent(req CreateEventRequest) (*Event, error) {
	requestURL := fmt.Sprintf("%s/api/v1/events", c.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateEventResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Event, nil
}

// ErrorTrackingIssue is one Error Tracking issue from Datadog.
type ErrorTrackingIssue struct {
	ID               string
	ErrorType        string
	ErrorMessage     string
	Service          string
	FilePath         string
	FunctionName     string
	Platform         string
	State            string
	Languages        []string
	IsCrash          bool
	FirstSeen        time.Time
	LastSeen         time.Time
	FirstSeenVersion string
	LastSeenVersion  string
	Regression       *ErrorTrackingRegression
	Assignee         string
	Teams            []string
	CaseKey          string
	CaseTitle        string
	CaseURL          string
	TotalCount       int64
	ImpactedUsers    int64
	ImpactedSessions int64
	HasActivity      bool
	Stack            string
	Sample           *ErrorSample
	RelatedLogs      []LogLine
	URL              string
}

// ErrorSample is one error event that Datadog grouped into the issue.
type ErrorSample struct {
	Source      string
	Timestamp   time.Time
	TraceID     string
	SpanID      string
	Env         string
	Version     string
	Host        string
	Resource    string
	HTTPMethod  string
	HTTPPath    string
	HTTPStatus  string
	UserID      string
	RequestID   string
	Fingerprint string
	Origin      string
	Route       string
	Action      string
	Breadcrumbs []string
	Stack       string
	TraceURL    string
	LogsURL     string
}

// LogLine is one log that shares the error sample trace or span.
type LogLine struct {
	Timestamp  time.Time
	Status     string
	Service    string
	Message    string
	HTTPMethod string
	HTTPPath   string
	HTTPStatus string
	ErrorKind  string
}

// ErrorTrackingRegression is a resolved issue that started again.
type ErrorTrackingRegression struct {
	RegressedAt        time.Time
	RegressedAtVersion string
	ResolvedAt         time.Time
}

// IssueTitle is the list and import title for an Error Tracking issue.
func (i ErrorTrackingIssue) IssueTitle() string {
	errorType := strings.TrimSpace(i.ErrorType)
	message := strings.TrimSpace(i.ErrorMessage)
	switch {
	case errorType != "" && message != "":
		return errorType + ": " + message
	case message != "":
		return message
	case errorType != "":
		return errorType
	default:
		return "Datadog error"
	}
}

func (c *Client) appSite() string {
	site := strings.TrimSpace(c.Site)
	if site == "" {
		return "datadoghq.com"
	}
	return site
}

// IssueURL is the Datadog app URL for this issue on the client's site.
func (c *Client) IssueURL(issueID string) string {
	return fmt.Sprintf("https://app.%s/error-tracking/issue/%s", c.appSite(), strings.TrimSpace(issueID))
}

// TraceURL is the Datadog APM URL for a trace. A 128-bit hex ID uses the
// decimal lower 64 bits, which the Datadog app expects.
func (c *Client) TraceURL(traceID string) string {
	id := strings.TrimSpace(traceID)
	if decimal := decimalLower64(id); decimal != "" {
		id = decimal
	}
	if id == "" {
		return ""
	}
	return fmt.Sprintf("https://app.%s/apm/trace/%s", c.appSite(), id)
}

// LogsURL is the Datadog Logs explorer URL for a query and time window.
func (c *Client) LogsURL(query string, from, to time.Time) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	values := url.Values{}
	values.Set("query", query)
	values.Set("live", "false")
	if !from.IsZero() {
		values.Set("from_ts", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		values.Set("to_ts", strconv.FormatInt(to.UnixMilli(), 10))
	}
	return fmt.Sprintf("https://app.%s/logs?%s", c.appSite(), values.Encode())
}

type errorTrackingSearchRequest struct {
	Data errorTrackingSearchRequestData `json:"data"`
}

type errorTrackingSearchRequestData struct {
	Type       string                               `json:"type"`
	Attributes errorTrackingSearchRequestAttributes `json:"attributes"`
}

type errorTrackingSearchRequestAttributes struct {
	Query   string   `json:"query"`
	From    int64    `json:"from"`
	To      int64    `json:"to"`
	Persona string   `json:"persona"`
	States  []string `json:"states"`
}

type errorTrackingSearchResponse struct {
	Data     []errorTrackingSearchResult  `json:"data"`
	Included []errorTrackingIssueResource `json:"included"`
}

type errorTrackingSearchResult struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Relationships struct {
		Issue struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
		} `json:"issue"`
	} `json:"relationships"`
}

type errorTrackingIssueResource struct {
	ID         string                       `json:"id"`
	Type       string                       `json:"type"`
	Attributes errorTrackingIssueAttributes `json:"attributes"`
}

type errorTrackingIssueAttributes struct {
	ErrorMessage     string                   `json:"error_message"`
	ErrorType        string                   `json:"error_type"`
	Service          string                   `json:"service"`
	FilePath         string                   `json:"file_path"`
	FunctionName     string                   `json:"function_name"`
	Platform         string                   `json:"platform"`
	State            string                   `json:"state"`
	Languages        []string                 `json:"languages"`
	IsCrash          bool                     `json:"is_crash"`
	FirstSeen        int64                    `json:"first_seen"`
	LastSeen         int64                    `json:"last_seen"`
	FirstSeenVersion string                   `json:"first_seen_version"`
	LastSeenVersion  string                   `json:"last_seen_version"`
	Regression       *errorTrackingRegression `json:"regression"`
}

type errorTrackingRegression struct {
	RegressedAt        string `json:"regressed_at"`
	RegressedAtVersion string `json:"regressed_at_version"`
	ResolvedAt         string `json:"resolved_at"`
}

type errorTrackingIssueResponse struct {
	Data     errorTrackingIssueResource      `json:"data"`
	Included []errorTrackingIncludedResource `json:"included"`
}

type errorTrackingIncludedResource struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes json.RawMessage `json:"attributes"`
}

// SearchErrorTrackingIssues searches open and acknowledged Error Tracking
// issues from about the last 30 days. Empty query means "*". Results are the
// newest created issues, first-seen time descending. Limit is capped at 100.
func (c *Client) SearchErrorTrackingIssues(query string, limit int) ([]ErrorTrackingIssue, error) {
	if limit <= 0 {
		limit = maxErrorTrackingSearchLimit
	}
	if limit > maxErrorTrackingSearchLimit {
		limit = maxErrorTrackingSearchLimit
	}

	query = strings.TrimSpace(query)
	if query == "" {
		query = "*"
	}

	now := time.Now()
	payload := errorTrackingSearchRequest{
		Data: errorTrackingSearchRequestData{
			Type: "search_request",
			Attributes: errorTrackingSearchRequestAttributes{
				Query:   query,
				From:    now.Add(-errorTrackingSearchWindow).UnixMilli(),
				To:      now.UnixMilli(),
				Persona: "ALL",
				States:  []string{"OPEN", "ACKNOWLEDGED"},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling search request: %v", err)
	}

	requestURL := fmt.Sprintf("%s/api/v2/error-tracking/issues/search?include=%s", c.BaseURL, url.QueryEscape("issue"))
	responseBody, err := c.execRequest(http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, mapErrorTrackingError(err)
	}

	var response errorTrackingSearchResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing search response: %v", err)
	}

	byID := make(map[string]ErrorTrackingIssue, len(response.Included))
	for _, included := range response.Included {
		if !strings.EqualFold(included.Type, "issue") || strings.TrimSpace(included.ID) == "" {
			continue
		}
		byID[included.ID] = errorTrackingIssueFromResource(included)
	}

	issues := make([]ErrorTrackingIssue, 0, len(response.Data))
	seen := map[string]bool{}
	for _, result := range response.Data {
		issueID := strings.TrimSpace(result.Relationships.Issue.Data.ID)
		if issueID == "" {
			issueID = strings.TrimSpace(result.ID)
		}
		if issueID == "" || seen[issueID] {
			continue
		}
		seen[issueID] = true

		if issue, ok := byID[issueID]; ok {
			issues = append(issues, issue)
		} else {
			issues = append(issues, ErrorTrackingIssue{ID: issueID})
		}
	}

	if len(issues) == 0 {
		return issues, nil
	}
	return NewestCreatedErrorTrackingIssues(issues, limit), nil
}

// GetErrorTrackingIssue loads one Error Tracking issue by id.
func (c *Client) GetErrorTrackingIssue(issueID string) (*ErrorTrackingIssue, error) {
	issueID = strings.TrimSpace(issueID)
	if issueID == "" {
		return nil, fmt.Errorf("issue id is required")
	}

	requestURL := fmt.Sprintf(
		"%s/api/v2/error-tracking/issues/%s?include=%s",
		c.BaseURL,
		url.PathEscape(issueID),
		url.QueryEscape("assignee,case,team_owners"),
	)
	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, mapErrorTrackingError(err)
	}

	var response errorTrackingIssueResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing issue response: %v", err)
	}
	if strings.TrimSpace(response.Data.ID) == "" {
		return nil, fmt.Errorf("issue not found")
	}

	issue := errorTrackingIssueFromResource(response.Data)
	applyErrorTrackingIncluded(&issue, response.Included)
	issue.URL = c.IssueURL(issue.ID)
	return &issue, nil
}

func errorTrackingIssueFromResource(resource errorTrackingIssueResource) ErrorTrackingIssue {
	attributes := resource.Attributes
	issue := ErrorTrackingIssue{
		ID:               strings.TrimSpace(resource.ID),
		ErrorType:        strings.TrimSpace(attributes.ErrorType),
		ErrorMessage:     strings.TrimSpace(attributes.ErrorMessage),
		Service:          strings.TrimSpace(attributes.Service),
		FilePath:         strings.TrimSpace(attributes.FilePath),
		FunctionName:     strings.TrimSpace(attributes.FunctionName),
		Platform:         strings.TrimSpace(attributes.Platform),
		State:            strings.TrimSpace(attributes.State),
		Languages:        compactStrings(attributes.Languages),
		IsCrash:          attributes.IsCrash,
		FirstSeen:        unixMilliTime(attributes.FirstSeen),
		LastSeen:         unixMilliTime(attributes.LastSeen),
		FirstSeenVersion: strings.TrimSpace(attributes.FirstSeenVersion),
		LastSeenVersion:  strings.TrimSpace(attributes.LastSeenVersion),
	}
	if attributes.Regression != nil {
		issue.Regression = &ErrorTrackingRegression{
			RegressedAt:        parseDatadogTime(attributes.Regression.RegressedAt),
			RegressedAtVersion: strings.TrimSpace(attributes.Regression.RegressedAtVersion),
			ResolvedAt:         parseDatadogTime(attributes.Regression.ResolvedAt),
		}
	}
	return issue
}

func compactStrings(values []string) []string {
	compact := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		compact = append(compact, value)
	}
	return compact
}

func unixMilliTime(millis int64) time.Time {
	if millis <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(millis).UTC()
}

func parseDatadogTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func mapErrorTrackingError(err error) error {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return ErrErrorTrackingForbidden
	}
	return err
}
