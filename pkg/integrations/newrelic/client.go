package newrelic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	APIKey        string
	NerdGraphURL  string
	MetricBaseURL string
	http          core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("API key is required: %w", err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return nil, fmt.Errorf("API key is required")
	}

	site, err := ctx.GetConfig("site")
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	var nerdGraphURL, metricBaseURL string
	if string(site) == "EU" {
		nerdGraphURL = nerdGraphAPIBaseEU
		metricBaseURL = metricsAPIBaseEU
	} else {
		nerdGraphURL = nerdGraphAPIBaseUS
		metricBaseURL = metricsAPIBaseUS
	}

	return &Client{
		APIKey:        key,
		NerdGraphURL:  nerdGraphURL,
		MetricBaseURL: metricBaseURL,
		http:          httpCtx,
	}, nil
}

type MetricType string

const (
	MetricTypeGauge   MetricType = "gauge"
	MetricTypeCount   MetricType = "count"
	MetricTypeSummary MetricType = "summary"
)

type Metric struct {
	Name       string         `json:"name"`
	Type       MetricType     `json:"type"`
	Value      any            `json:"value"`
	Timestamp  int64          `json:"timestamp,omitempty"`
	IntervalMs int64          `json:"interval.ms,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type MetricBatch struct {
	Common  *map[string]any `json:"common,omitempty"`
	Metrics []Metric        `json:"metrics"`
}

func IsUserAPIKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, "NRAK-")
}

func (c *Client) ReportMetric(ctx context.Context, batch []MetricBatch) error {
	url := c.MetricBaseURL

	bodyBytes, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal metric batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create metric request: %w", err)
	}

	if IsUserAPIKey(c.APIKey) {
		req.Header.Set("Api-Key", c.APIKey)
	} else {
		req.Header.Set("X-License-Key", c.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to report metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return parseErrorResponse(url, body, resp.StatusCode)
	}

	return nil
}
func (c *Client) ValidateAPIKey(ctx context.Context) error {
	graphqlQuery := `{ actor { user { name email } } }`

	gqlRequest := GraphQLRequest{
		Query: graphqlQuery,
	}

	bodyBytes, err := json.Marshal(gqlRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.NerdGraphURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create NerdGraph request: %w", err)
	}

	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute NerdGraph request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseErrorResponse(c.NerdGraphURL, responseBody, resp.StatusCode)
	}

	var gqlResponse GraphQLResponse
	if err := json.Unmarshal(responseBody, &gqlResponse); err != nil {
		return fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(gqlResponse.Errors) > 0 {
		var errMessages []string
		for _, gqlErr := range gqlResponse.Errors {
			errMessages = append(errMessages, gqlErr.Message)
		}
		return fmt.Errorf("GraphQL errors: %s", strings.Join(errMessages, "; "))
	}

	if gqlResponse.Data == nil {
		return fmt.Errorf("no data returned from identity query")
	}

	return nil
}

// ListAccounts fetches the list of accounts the API key has access to
func (c *Client) ListAccounts(ctx context.Context) ([]Account, error) {
	graphqlQuery := `{ actor { accounts { id name } } }`

	gqlRequest := GraphQLRequest{
		Query: graphqlQuery,
	}

	bodyBytes, err := json.Marshal(gqlRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.NerdGraphURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create NerdGraph request: %w", err)
	}

	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute NerdGraph request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseErrorResponse(c.NerdGraphURL, responseBody, resp.StatusCode)
	}

	var gqlResponse GraphQLResponse
	if err := json.Unmarshal(responseBody, &gqlResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(gqlResponse.Errors) > 0 {
		var errMessages []string
		for _, gqlErr := range gqlResponse.Errors {
			errMessages = append(errMessages, gqlErr.Message)
		}
		return nil, fmt.Errorf("GraphQL errors: %s", strings.Join(errMessages, "; "))
	}

	actor, ok := gqlResponse.Data["actor"].(map[string]interface{})
	if !ok || actor == nil {
		return nil, fmt.Errorf("invalid GraphQL response: missing actor")
	}

	accountsData, ok := actor["accounts"].([]interface{})
	if !ok {
		return []Account{}, nil
	}

	accounts := make([]Account, 0, len(accountsData))
	for _, accData := range accountsData {
		m, ok := accData.(map[string]interface{})
		if !ok {
			continue
		}

		var id int64
		switch v := m["id"].(type) {
		case float64:
			id = int64(v)
		case int64:
			id = v
		case string:
			id, _ = strconv.ParseInt(v, 10, 64)
		}

		name, _ := m["name"].(string)

        // FIXED: Added filter to skip invalid or empty accounts
        if id == 0 || name == "" {
            continue
        }

		accounts = append(accounts, Account{
			ID:   id,
			Name: name,
		})
	}

	return accounts, nil
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   map[string]interface{} `json:"data"`
	Errors []GraphQLError         `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string        `json:"message"`
	Path    []interface{} `json:"path,omitempty"`
}

type NRQLQueryResponse struct {
	Results     []map[string]interface{} `json:"results"`
	TotalResult map[string]interface{}   `json:"totalResult,omitempty"`
	Metadata    *NRQLMetadata            `json:"metadata,omitempty"`
}

type NRQLMetadata struct {
	EventTypes []string    `json:"eventTypes,omitempty"`
	Facets     []string    `json:"facets,omitempty"`
	Messages   []string    `json:"messages,omitempty"`
	TimeWindow *TimeWindow `json:"timeWindow,omitempty"`
}

type TimeWindow struct {
	Begin int64 `json:"begin"`
	End   int64 `json:"end"`
}

func (c *Client) RunNRQLQuery(ctx context.Context, accountID int64, query string, timeout int) (*NRQLQueryResponse, error) {
	graphqlQuery := fmt.Sprintf(`{
        actor {
            account(id: %d) {
                nrql(query: %s, timeout: %d) {
                    results
                    totalResult
                    metadata {
                        eventTypes
                        facets
                        messages
                        timeWindow {
                            begin
                            end
                        }
                    }
                }
            }
        }
    }`, accountID, strconv.Quote(query), timeout)

	gqlRequest := GraphQLRequest{
		Query: graphqlQuery,
	}

	bodyBytes, err := json.Marshal(gqlRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.NerdGraphURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create NerdGraph request: %w", err)
	}

	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute NerdGraph request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseErrorResponse(c.NerdGraphURL, responseBody, resp.StatusCode)
	}

	var gqlResponse GraphQLResponse
	if err := json.Unmarshal(responseBody, &gqlResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(gqlResponse.Errors) > 0 {
		var errMessages []string
		for _, gqlErr := range gqlResponse.Errors {
			errMessages = append(errMessages, gqlErr.Message)
		}
		return nil, fmt.Errorf("GraphQL errors: %s", strings.Join(errMessages, "; "))
	}

	actor, ok := gqlResponse.Data["actor"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid GraphQL response: missing actor")
	}

	account, ok := actor["account"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid GraphQL response: missing account")
	}

	nrqlData, ok := account["nrql"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid GraphQL response: missing nrql")
	}

	nrqlBytes, err := json.Marshal(nrqlData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NRQL data: %w", err)
	}

	var nrqlResponse NRQLQueryResponse
	if err := json.Unmarshal(nrqlBytes, &nrqlResponse); err != nil {
		return nil, fmt.Errorf("failed to decode NRQL response: %w", err)
	}

	return &nrqlResponse, nil
}