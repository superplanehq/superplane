package dash0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	Token   string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.AppInstallationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting api token: %v", err)
	}

	// Default to empty - user must provide their organization-specific URL
	// Dash0 Cloud uses organization-specific URLs like https://your-org.dash0.com
	baseURL := ""
	baseURLConfig, err := ctx.GetConfig("baseURL")
	if err == nil && baseURLConfig != nil && len(baseURLConfig) > 0 {
		baseURL = strings.TrimSuffix(string(baseURLConfig), "/")
	}
	
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is required for Dash0 Cloud. Find your API URL in Dash0 dashboard under Organization Settings > Endpoints Reference")
	}

	return &Client{
		Token:   string(apiToken),
		BaseURL: baseURL,
		http:    http,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

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
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   map[string]any `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message   string                 `json:"message"`
	Locations []GraphQLErrorLocation `json:"locations,omitempty"`
	Path      []any                  `json:"path,omitempty"`
}

type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func (c *Client) ExecuteGraphQL(query string, variables map[string]any) (map[string]any, error) {
	request := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/graphql", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response GraphQLResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	// Check for GraphQL errors
	if len(response.Errors) > 0 {
		errorMessages := make([]string, len(response.Errors))
		for i, err := range response.Errors {
			errorMessages[i] = err.Message
		}
		return nil, fmt.Errorf("graphql errors: %s", strings.Join(errorMessages, "; "))
	}

	return response.Data, nil
}
