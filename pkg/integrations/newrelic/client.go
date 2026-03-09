package newrelic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const MaxResponseSize = 1 * 1024 * 1024 // 1MB

// APIError represents an HTTP error response from New Relic.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request got %d code: %s", e.StatusCode, e.Body)
}

type Client struct {
	AccountID    string
	UserAPIKey   string
	LicenseKey   string
	NerdGraphURL string
	MetricAPIURL string
	http         core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	accountID, err := ctx.GetConfig("accountId")
	if err != nil {
		return nil, fmt.Errorf("error getting accountId: %v", err)
	}

	region, err := ctx.GetConfig("region")
	if err != nil {
		return nil, fmt.Errorf("error getting region: %v", err)
	}

	userAPIKey, err := ctx.GetConfig("userApiKey")
	if err != nil {
		return nil, fmt.Errorf("error getting userApiKey: %v", err)
	}

	licenseKey, err := ctx.GetConfig("licenseKey")
	if err != nil {
		return nil, fmt.Errorf("error getting licenseKey: %v", err)
	}

	accountIDStr := string(accountID)
	if !accountIDRegexp.MatchString(accountIDStr) {
		return nil, fmt.Errorf("accountId must be numeric, got %q", accountIDStr)
	}

	nerdGraphURL, metricAPIURL := urlsForRegion(string(region))

	return &Client{
		AccountID:    accountIDStr,
		UserAPIKey:   string(userAPIKey),
		LicenseKey:   string(licenseKey),
		NerdGraphURL: nerdGraphURL,
		MetricAPIURL: metricAPIURL,
		http:         http,
	}, nil
}

func urlsForRegion(region string) (string, string) {
	if region == "EU" {
		return "https://api.eu.newrelic.com/graphql", "https://metric-api.eu.newrelic.com/metric/v1"
	}

	return "https://api.newrelic.com/graphql", "https://metric-api.newrelic.com/metric/v1"
}

// ValidateCredentials verifies that the User API Key is valid
// by running a simple NerdGraph query.
func (c *Client) ValidateCredentials(ctx context.Context) error {
	query := `{ "query": "{ actor { user { name } } }" }`
	body, err := c.nerdGraphRequest(ctx, []byte(query))
	if err != nil {
		return err
	}

	var res struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("error parsing validation response: %v", err)
	}

	if len(res.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", res.Errors[0].Message)
	}

	return nil
}

// NerdGraphQuery executes a NerdGraph (GraphQL) query and returns the raw response.
func (c *Client) NerdGraphQuery(ctx context.Context, query string) ([]byte, error) {
	payload := map[string]string{"query": query}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling query: %v", err)
	}

	return c.nerdGraphRequest(ctx, body)
}

// ReportMetric sends metric data to the New Relic Metric API.
func (c *Client) ReportMetric(ctx context.Context, payload []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.MetricAPIURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", c.LicenseKey)

	return c.execRequest(req)
}

// CreateNotificationDestination creates a webhook destination in New Relic via NerdGraph.
func (c *Client) CreateNotificationDestination(ctx context.Context, webhookURL string, secret string) (string, error) {
	name := fmt.Sprintf("SuperPlane-%d", time.Now().UnixMilli())
	authBlock := ""
	if secret != "" {
		authBlock = fmt.Sprintf(`, auth: {type: TOKEN, token: {prefix: "Bearer", token: %s}}`,
			quoteGraphQL(secret))
	}
	query := fmt.Sprintf(`mutation {
		aiNotificationsCreateDestination(accountId: %s, destination: {
			name: %s,
			type: WEBHOOK,
			properties: [{key: "url", value: %s}]%s
		}) {
			destination { id }
			error {
				... on AiNotificationsResponseError { description }
				... on AiNotificationsDataValidationError { details }
				... on AiNotificationsSuggestionError { description details }
			}
		}
	}`, c.AccountID, quoteGraphQL(name), quoteGraphQL(webhookURL), authBlock)

	body, err := c.NerdGraphQuery(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to create notification destination: %w", err)
	}

	var res struct {
		Data struct {
			Result struct {
				Destination struct {
					ID string `json:"id"`
				} `json:"destination"`
				Error *struct {
					Description string `json:"description"`
					Details     string `json:"details"`
				} `json:"error"`
			} `json:"aiNotificationsCreateDestination"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("failed to parse destination response: %w", err)
	}

	if len(res.Errors) > 0 {
		return "", fmt.Errorf("GraphQL error creating destination: %s", res.Errors[0].Message)
	}

	if res.Data.Result.Error != nil {
		errMsg := res.Data.Result.Error.Description
		if errMsg == "" {
			errMsg = res.Data.Result.Error.Details
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return "", fmt.Errorf("failed to create destination: %s", errMsg)
	}

	if res.Data.Result.Destination.ID == "" {
		return "", fmt.Errorf("destination created but no ID returned")
	}

	return res.Data.Result.Destination.ID, nil
}

// CreateNotificationChannel creates a webhook notification channel in New Relic via NerdGraph.
func (c *Client) CreateNotificationChannel(ctx context.Context, destinationID string, payloadTemplate string) (string, error) {
	name := fmt.Sprintf("SuperPlane-%d", time.Now().UnixMilli())
	query := fmt.Sprintf(`mutation {
		aiNotificationsCreateChannel(accountId: %s, channel: {
			name: %s,
			type: WEBHOOK,
			product: IINT,
			destinationId: %s,
			properties: [{key: "payload", value: %s}]
		}) {
			channel { id }
			error {
				... on AiNotificationsResponseError { description }
				... on AiNotificationsDataValidationError { details }
				... on AiNotificationsSuggestionError { description details }
			}
		}
	}`, c.AccountID, quoteGraphQL(name), quoteGraphQL(destinationID), quoteGraphQL(payloadTemplate))

	body, err := c.NerdGraphQuery(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to create notification channel: %w", err)
	}

	var res struct {
		Data struct {
			Result struct {
				Channel struct {
					ID string `json:"id"`
				} `json:"channel"`
				Error *struct {
					Description string `json:"description"`
					Details     string `json:"details"`
				} `json:"error"`
			} `json:"aiNotificationsCreateChannel"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("failed to parse channel response: %w", err)
	}

	if len(res.Errors) > 0 {
		return "", fmt.Errorf("GraphQL error creating channel: %s", res.Errors[0].Message)
	}

	if res.Data.Result.Error != nil {
		errMsg := res.Data.Result.Error.Description
		if errMsg == "" {
			errMsg = res.Data.Result.Error.Details
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return "", fmt.Errorf("failed to create channel: %s", errMsg)
	}

	if res.Data.Result.Channel.ID == "" {
		return "", fmt.Errorf("channel created but no ID returned")
	}

	return res.Data.Result.Channel.ID, nil
}

// DeleteNotificationChannel deletes a notification channel in New Relic via NerdGraph.
func (c *Client) DeleteNotificationChannel(ctx context.Context, channelID string) error {
	query := fmt.Sprintf(`mutation {
		aiNotificationsDeleteChannel(accountId: %s, channelId: %s) {
			ids
			error { details }
		}
	}`, c.AccountID, quoteGraphQL(channelID))

	body, err := c.NerdGraphQuery(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}

	var res struct {
		Data struct {
			Result struct {
				IDs   []string `json:"ids"`
				Error *struct {
					Details string `json:"details"`
				} `json:"error"`
			} `json:"aiNotificationsDeleteChannel"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("failed to parse delete channel response: %w", err)
	}

	if len(res.Errors) > 0 {
		return fmt.Errorf("GraphQL error deleting channel: %s", res.Errors[0].Message)
	}

	if res.Data.Result.Error != nil && res.Data.Result.Error.Details != "" {
		return fmt.Errorf("failed to delete channel: %s", res.Data.Result.Error.Details)
	}

	return nil
}

// DeleteNotificationDestination deletes a notification destination in New Relic via NerdGraph.
func (c *Client) DeleteNotificationDestination(ctx context.Context, destinationID string) error {
	query := fmt.Sprintf(`mutation {
		aiNotificationsDeleteDestination(accountId: %s, destinationId: %s) {
			ids
			error { details }
		}
	}`, c.AccountID, quoteGraphQL(destinationID))

	body, err := c.NerdGraphQuery(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete notification destination: %w", err)
	}

	var res struct {
		Data struct {
			Result struct {
				IDs   []string `json:"ids"`
				Error *struct {
					Details string `json:"details"`
				} `json:"error"`
			} `json:"aiNotificationsDeleteDestination"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("failed to parse delete destination response: %w", err)
	}

	if len(res.Errors) > 0 {
		return fmt.Errorf("GraphQL error deleting destination: %s", res.Errors[0].Message)
	}

	if res.Data.Result.Error != nil && res.Data.Result.Error.Details != "" {
		return fmt.Errorf("failed to delete destination: %s", res.Data.Result.Error.Details)
	}

	return nil
}

type NerdGraphNRQLResponse struct {
	Data struct {
		Actor struct {
			Account struct {
				NRQL struct {
					Results []any `json:"results"`
				} `json:"nrql"`
			} `json:"account"`
		} `json:"actor"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// RunNRQLQuery executes a NRQL query via NerdGraph and returns the result rows.
func (c *Client) RunNRQLQuery(ctx context.Context, query string, timeout int) ([]any, error) {
	graphQLQuery := fmt.Sprintf(
		`{ actor { account(id: %s) { nrql(query: %s, timeout: %d) { results } } } }`,
		c.AccountID,
		quoteGraphQL(query),
		timeout,
	)
	body, err := c.NerdGraphQuery(ctx, graphQLQuery)
	if err != nil {
		return nil, err
	}
	var response NerdGraphNRQLResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", response.Errors[0].Message)
	}
	results := response.Data.Actor.Account.NRQL.Results
	if results == nil {
		results = []any{}
	}
	return results, nil
}

func quoteGraphQL(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (c *Client) nerdGraphRequest(ctx context.Context, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.NerdGraphURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", c.UserAPIKey)

	return c.execRequest(req)
}

func (c *Client) execRequest(req *http.Request) ([]byte, error) {
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	limitedReader := io.LimitReader(res.Body, MaxResponseSize+1)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if len(responseBody) > MaxResponseSize {
		return nil, fmt.Errorf("response too large: exceeds maximum size of %d bytes", MaxResponseSize)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		errBody := string(responseBody)
		if len(errBody) > 256 {
			errBody = errBody[:256] + "... (truncated)"
		}
		return nil, &APIError{StatusCode: res.StatusCode, Body: errBody}
	}

	return responseBody, nil
}
