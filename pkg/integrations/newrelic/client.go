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
	UserAPIKey    string
	LicenseKey    string
	NerdGraphURL  string
	MetricBaseURL string
	http          core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	userAPIKey := ""
	if raw, err := ctx.GetConfig("userApiKey"); err == nil {
		userAPIKey = strings.TrimSpace(string(raw))
	}

	licenseKey := ""
	if raw, err := ctx.GetConfig("licenseKey"); err == nil {
		licenseKey = strings.TrimSpace(string(raw))
	}

	if userAPIKey == "" && licenseKey == "" {
		return nil, fmt.Errorf("at least one API key is required: provide a User API Key and/or a License Key")
	}

	site, err := ctx.GetConfig("site")
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	siteStr := strings.TrimSpace(string(site))

	var nerdGraphURL, metricBaseURL string
	if siteStr == "EU" {
		nerdGraphURL = nerdGraphAPIBaseEU
		metricBaseURL = metricsAPIBaseEU
	} else {
		nerdGraphURL = nerdGraphAPIBaseUS
		metricBaseURL = metricsAPIBaseUS
	}

	return &Client{
		UserAPIKey:    userAPIKey,
		LicenseKey:    licenseKey,
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

type ReportMetricResult struct {
	StatusCode int    `json:"statusCode"`
	Status     string `json:"status"`
}

func (c *Client) ReportMetric(ctx context.Context, batch []MetricBatch) (*ReportMetricResult, error) {
	url := c.MetricBaseURL

	bodyBytes, err := json.Marshal(batch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metric batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create metric request: %w", err)
	}

	// Use License Key with X-License-Key header; fall back to User API Key
	if c.LicenseKey != "" {
		req.Header.Set("X-License-Key", c.LicenseKey)
	} else {
		req.Header.Set("Api-Key", c.UserAPIKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to report metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, parseErrorResponse(url, body, resp.StatusCode)
	}

	return &ReportMetricResult{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
	}, nil
}

// doNerdGraphRequest handles the shared boilerplate for all GraphQL calls to Newrelic
func (c *Client) doNerdGraphRequest(ctx context.Context, query string, variables map[string]any, outData any) error {
	gqlRequest := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(gqlRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.NerdGraphURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create NerdGraph request: %w", err)
	}

	req.Header.Set("Api-Key", c.UserAPIKey)
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

	// Marshal the map back to JSON and Unmarshal into the specific struct 'outData'
	// This is the cleanest way to map a map[string]any to a specific struct
	dataBytes, err := json.Marshal(gqlResponse.Data)
	if err != nil {
		return fmt.Errorf("failed to re-marshal data: %w", err)
	}

	return json.Unmarshal(dataBytes, outData)
}

func (c *Client) ValidateAPIKey(ctx context.Context) error {
	query := `{ actor { user { name email } } }`
	var out any // We don't actually need the data for validation, just the error check
	return c.doNerdGraphRequest(ctx, query, nil, &out)
}

// ListAccounts fetches the list of accounts the API key has access to
func (c *Client) ListAccounts(ctx context.Context) ([]Account, error) {
	query := `{ actor { accounts { id name } } }`
	var response struct {
		Actor struct {
			Accounts []Account `json:"accounts"`
		} `json:"actor"`
	}

	if err := c.doNerdGraphRequest(ctx, query, nil, &response); err != nil {
		return nil, err
	}
	return response.Actor.Accounts, nil
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
	Results       []map[string]interface{} `json:"results"`
	TotalResult   map[string]interface{}   `json:"totalResult,omitempty"`
	Metadata      *NRQLMetadata            `json:"metadata,omitempty"`
	QueryProgress *QueryProgress           `json:"queryProgress,omitempty"`
}

type QueryProgress struct {
	QueryId          string `json:"queryId"`
	Completed        bool   `json:"completed"`
	RetryAfter       int    `json:"retryAfter"`
	RetryDeadline    int64  `json:"retryDeadline"`
	ResultExpiration int64  `json:"resultExpiration"`
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

// RunNRQLQuery executes an async NRQL query via NerdGraph with a fixed 10s timeout.
// If the query completes within 10s, results are returned directly.
// Otherwise, QueryProgress is populated with a queryId for polling.
func (c *Client) RunNRQLQuery(ctx context.Context, accountID int64, query string) (*NRQLQueryResponse, error) {
	graphqlQuery := fmt.Sprintf(`{
        actor {
            account(id: %d) {
                nrql(query: %s, timeout: 10, async: true) {
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
                    queryProgress {
                        queryId
                        completed
                        retryAfter
                        retryDeadline
                        resultExpiration
                    }
                }
            }
        }
    }`, accountID, strconv.Quote(query))

	return c.executeNRQLGraphQL(ctx, graphqlQuery)
}

// PollNRQLQuery polls for the result of an async NRQL query using the queryId.
func (c *Client) PollNRQLQuery(ctx context.Context, accountID int64, queryId string) (*NRQLQueryResponse, error) {
	graphqlQuery := fmt.Sprintf(`{
        actor {
            account(id: %d) {
                nrqlQueryProgress(queryId: %s) {
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
                    queryProgress {
                        queryId
                        completed
                        retryAfter
                        retryDeadline
                        resultExpiration
                    }
                }
            }
        }
    }`, accountID, strconv.Quote(queryId))

	return c.executeNRQLGraphQL(ctx, graphqlQuery)
}

// nrqlGraphQLData is used to deserialize the GraphQL response from doNerdGraphRequest.
type nrqlGraphQLData struct {
	Actor struct {
		Account struct {
			NRQL              *NRQLQueryResponse `json:"nrql,omitempty"`
			NRQLQueryProgress *NRQLQueryResponse `json:"nrqlQueryProgress,omitempty"`
		} `json:"account"`
	} `json:"actor"`
}

// executeNRQLGraphQL is a shared helper for NRQL query and poll requests.
// It delegates HTTP/GraphQL boilerplate to doNerdGraphRequest.
func (c *Client) executeNRQLGraphQL(ctx context.Context, graphqlQuery string) (*NRQLQueryResponse, error) {
	var data nrqlGraphQLData
	if err := c.doNerdGraphRequest(ctx, graphqlQuery, nil, &data); err != nil {
		return nil, err
	}

	// Try nrql first (initial query), then nrqlQueryProgress (poll)
	if data.Actor.Account.NRQL != nil {
		return data.Actor.Account.NRQL, nil
	}
	if data.Actor.Account.NRQLQueryProgress != nil {
		return data.Actor.Account.NRQLQueryProgress, nil
	}

	return nil, fmt.Errorf("invalid GraphQL response: missing nrql or nrqlQueryProgress")
}
