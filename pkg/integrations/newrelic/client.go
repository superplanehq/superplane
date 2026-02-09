package newrelic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	APIKey  string
	BaseURL string
	HTTP    core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	key := string(apiKey)
	if key == "" {
		return nil, fmt.Errorf("API key is required")
	}

	site, err := ctx.GetConfig("site")
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	baseURL := restAPIBaseUS
	if string(site) == "EU" {
		baseURL = restAPIBaseEU
	}

	return &Client{
		APIKey:  key,
		BaseURL: baseURL,
		HTTP:    httpCtx,
	}, nil
}

func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	url := c.BaseURL + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseErrorResponse(responseBody, resp.StatusCode)
	}

	return responseBody, nil
}

func (c *Client) ValidateAPIKey() error {
	_, err := c.ListAccounts()
	return err
}

type ListAccountsResponse struct {
	Accounts []Account `json:"accounts"`
}

func (c *Client) ListAccounts() ([]Account, error) {
	responseBody, err := c.doRequest(http.MethodGet, "/accounts.json", nil)
	if err != nil {
		return nil, err
	}

	var response ListAccountsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to decode accounts response: %w", err)
	}

	return response.Accounts, nil
}
