package newrelic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	APIKey       string
	BaseURL      string
	NerdGraphURL string
	http         core.HTTPContext
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
	nerdGraphURL := nerdGraphAPIBaseUS

	if string(site) == "EU" {
		baseURL = restAPIBaseEU
		nerdGraphURL = nerdGraphAPIBaseEU
	}

	return &Client{
		APIKey:       key,
		BaseURL:      baseURL,
		NerdGraphURL: nerdGraphURL,
		http:         httpCtx,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// New Relic REST API v2 uses 'Api-Key' header
	// NerdGraph (GraphQL) also accepts 'Api-Key'
	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
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
	url := fmt.Sprintf("%s/accounts.json", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response ListAccountsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to decode accounts response: %w", err)
	}

	return response.Accounts, nil
}
