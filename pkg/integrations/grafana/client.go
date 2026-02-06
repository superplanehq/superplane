package grafana

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	maxResponseSize = 2 * 1024 * 1024 // 2MB
)

type Client struct {
	BaseURL  string
	APIToken string
	http     core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext, requireToken bool) (*Client, error) {
	baseURL, err := readBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	apiToken, err := readAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	if requireToken && apiToken == "" {
		return nil, fmt.Errorf("apiToken is required")
	}

	return &Client{
		BaseURL:  baseURL,
		APIToken: apiToken,
		http:     httpCtx,
	}, nil
}

func readBaseURL(ctx core.IntegrationContext) (string, error) {
	baseURLConfig, err := ctx.GetConfig("baseURL")
	if err != nil {
		return "", fmt.Errorf("error reading baseURL: %v", err)
	}

	if baseURLConfig == nil {
		return "", fmt.Errorf("baseURL is required")
	}

	baseURL := strings.TrimSpace(string(baseURLConfig))
	if baseURL == "" {
		return "", fmt.Errorf("baseURL is required")
	}

	if _, err := url.Parse(baseURL); err != nil {
		return "", fmt.Errorf("invalid baseURL: %v", err)
	}

	return strings.TrimSuffix(baseURL, "/"), nil
}

func readAPIToken(ctx core.IntegrationContext) (string, error) {
	apiTokenConfig, err := ctx.GetConfig("apiToken")
	if err != nil {
		return "", fmt.Errorf("error reading apiToken: %v", err)
	}

	if apiTokenConfig == nil {
		return "", nil
	}

	return strings.TrimSpace(string(apiTokenConfig)), nil
}

func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.BaseURL, "/"), strings.TrimPrefix(path, "/"))
}

func (c *Client) execRequest(method, path string, body io.Reader, contentType string) ([]byte, int, error) {
	req, err := http.NewRequest(method, c.buildURL(path), body)
	if err != nil {
		return nil, 0, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.APIToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	limitedReader := io.LimitReader(res.Body, maxResponseSize)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("error reading body: %v", err)
	}

	if len(responseBody) >= maxResponseSize {
		return nil, res.StatusCode, fmt.Errorf("response too large: exceeds maximum size of %d bytes", maxResponseSize)
	}

	return responseBody, res.StatusCode, nil
}
