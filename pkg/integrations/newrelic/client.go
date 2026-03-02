package newrelic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const MaxResponseSize = 1 * 1024 * 1024 // 1MB

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

	nerdGraphURL, metricAPIURL := urlsForRegion(string(region))

	return &Client{
		AccountID:    string(accountID),
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
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, errBody)
	}

	return responseBody, nil
}
