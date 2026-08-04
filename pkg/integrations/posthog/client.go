package posthog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// DefaultHost is PostHog's US cloud region. EU cloud is https://eu.posthog.com,
// and self-hosted instances use their own address.
const DefaultHost = "https://us.posthog.com"

// WebhookTemplateID is the PostHog CDP template SuperPlane provisions to deliver
// events. It POSTs a templated JSON body to a URL, which is all a trigger needs.
const WebhookTemplateID = "template-webhook"

// HogFunctionTypeDestination is the function type that runs against live events.
const HogFunctionTypeDestination = "destination"

// maxEventDefinitions bounds the event name dropdown. PostHog projects can
// accumulate thousands of definitions, and a picker that long is unusable
// anyway, so paging stops here instead of walking the whole taxonomy.
const maxEventDefinitions = 1000

// pageSize is the per-request limit used for paginated PostHog list endpoints.
const pageSize = 200

// Project is a PostHog project, which owns events, hog functions, and queries.
type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type projectListResponse struct {
	Next    *string   `json:"next"`
	Results []Project `json:"results"`
}

// EventDefinition is an event name PostHog has seen in a project. It backs the
// event picker on the trigger.
type EventDefinition struct {
	Name string `json:"name"`
}

type eventDefinitionListResponse struct {
	Next    *string           `json:"next"`
	Results []EventDefinition `json:"results"`
}

// HogFunctionInput is one value for a hog function input, as PostHog expects it:
// every input is wrapped in an object rather than assigned directly.
type HogFunctionInput struct {
	Value any `json:"value"`
}

// HogFunctionEventFilter restricts a hog function to a single event name.
type HogFunctionEventFilter struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// HogFunctionFilters controls which events reach the function. An empty Events
// slice means every event in the project matches.
type HogFunctionFilters struct {
	Events             []HogFunctionEventFilter `json:"events"`
	FilterTestAccounts bool                     `json:"filter_test_accounts"`
}

// CreateHogFunctionRequest is the body for creating a destination from a template.
type CreateHogFunctionRequest struct {
	Type        string                      `json:"type"`
	TemplateID  string                      `json:"template_id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	Enabled     bool                        `json:"enabled"`
	Inputs      map[string]HogFunctionInput `json:"inputs"`
	Filters     HogFunctionFilters          `json:"filters"`
}

// HogFunction is the created destination. The ID is needed to delete it later.
type HogFunction struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// QueryResponse is the result of a HogQL query. Results are positional rows,
// which Columns maps back onto names.
type QueryResponse struct {
	Results [][]any  `json:"results"`
	Columns []string `json:"columns"`
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type Client struct {
	Token string
	Host  string
	http  core.HTTPContext
}

func NewClient(httpContext core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error getting personal API key: %w", err)
	}

	token := strings.TrimSpace(string(apiKey))
	if token == "" {
		return nil, fmt.Errorf("personal API key is required")
	}

	//
	// Host is a defaulted field, so a connection saved before it existed simply
	// falls back to the US cloud region rather than failing to build a client.
	//
	host, _ := ctx.GetConfig("host")

	return &Client{
		Token: token,
		Host:  NormalizeHost(string(host)),
		http:  httpContext,
	}, nil
}

// NormalizeHost drops a trailing slash so paths concatenate predictably, and
// falls back to the US cloud region when no host is configured.
func NormalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return DefaultHost
	}

	return strings.TrimRight(host, "/")
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.Host+path, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return responseBody, nil
}

// ListProjects returns every project the personal API key can read.
func (c *Client) ListProjects() ([]Project, error) {
	all := []Project{}
	for offset := 0; ; offset += pageSize {
		path := fmt.Sprintf("/api/projects/?limit=%d&offset=%d", pageSize, offset)
		responseBody, err := c.execRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var response projectListResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing projects response: %w", err)
		}

		all = append(all, response.Results...)
		if response.Next == nil || len(response.Results) == 0 {
			break
		}
	}

	return all, nil
}

// ListEventDefinitions returns the event names seen in a project, capped at
// maxEventDefinitions.
func (c *Client) ListEventDefinitions(projectID string) ([]EventDefinition, error) {
	all := []EventDefinition{}
	for offset := 0; offset < maxEventDefinitions; offset += pageSize {
		path := fmt.Sprintf("/api/projects/%s/event_definitions/?limit=%d&offset=%d", url.PathEscape(projectID), pageSize, offset)
		responseBody, err := c.execRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var response eventDefinitionListResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing event definitions response: %w", err)
		}

		all = append(all, response.Results...)
		if response.Next == nil || len(response.Results) == 0 {
			break
		}
	}

	return all, nil
}

// CreateHogFunction creates a destination in a project.
func (c *Client) CreateHogFunction(projectID string, req CreateHogFunctionRequest) (*HogFunction, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error encoding request: %w", err)
	}

	path := fmt.Sprintf("/api/projects/%s/hog_functions/", url.PathEscape(projectID))
	responseBody, err := c.execRequest(http.MethodPost, path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var result HogFunction
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing hog function response: %w", err)
	}

	return &result, nil
}

// DeleteHogFunction removes a destination from a project.
func (c *Client) DeleteHogFunction(projectID, id string) error {
	path := fmt.Sprintf("/api/projects/%s/hog_functions/%s/", url.PathEscape(projectID), url.PathEscape(id))
	_, err := c.execRequest(http.MethodDelete, path, nil)
	return err
}

// QueryRequest is a HogQL query plus the values PostHog should substitute into
// its placeholders. Passing values separately keeps names the user picked out of
// the query text, so they cannot change what the query does.
type QueryRequest struct {
	Query  string
	Values map[string]any
}

// Query runs a HogQL query against a project and returns the raw response.
func (c *Client) Query(projectID string, request QueryRequest) (*QueryResponse, error) {
	query := map[string]any{
		"kind":  "HogQLQuery",
		"query": request.Query,
	}

	if len(request.Values) > 0 {
		query["values"] = request.Values
	}

	body := map[string]any{"query": query}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error encoding request: %w", err)
	}

	path := fmt.Sprintf("/api/projects/%s/query/", url.PathEscape(projectID))
	responseBody, err := c.execRequest(http.MethodPost, path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var result QueryResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing query response: %w", err)
	}

	return &result, nil
}

// RowsToMaps turns positional query results into name-keyed rows, so workflow
// expressions can read `row.event` instead of `row[0]`. Columns PostHog did not
// name fall back to their position.
func RowsToMaps(response *QueryResponse) []any {
	rows := make([]any, 0, len(response.Results))
	for _, result := range response.Results {
		row := map[string]any{}
		for i, value := range result {
			row[columnName(response.Columns, i)] = value
		}
		rows = append(rows, row)
	}

	return rows
}

func columnName(columns []string, i int) string {
	if i < len(columns) && columns[i] != "" {
		return columns[i]
	}

	return "column_" + strconv.Itoa(i)
}
