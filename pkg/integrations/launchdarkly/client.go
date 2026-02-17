package launchdarkly

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultAPIBaseURL = "https://app.launchdarkly.com/api/v2"

type Client struct {
	Token      string
	APIBaseURL string
	http       core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	token, err := integration.GetConfig("apiAccessToken")
	if err != nil {
		return nil, fmt.Errorf("missing apiAccessToken: %w", err)
	}

	baseURLRaw, _ := integration.GetConfig("apiBaseUrl")
	baseURL := resolveAPIBaseURL(string(baseURLRaw))

	return &Client{
		Token:      strings.TrimSpace(string(token)),
		APIBaseURL: baseURL,
		http:       httpClient,
	}, nil
}

func resolveAPIBaseURL(configured string) string {
	v := strings.TrimSpace(configured)
	if v == "" {
		return defaultAPIBaseURL
	}

	v = strings.TrimRight(v, "/")
	if strings.HasSuffix(v, "/api/v2") {
		return v
	}

	return v + "/api/v2"
}

func (c *Client) VerifyCredentials() error {
	_, body, err := c.execRequestRaw(http.MethodGet, c.APIBaseURL+"/projects?limit=1")
	if err != nil {
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("invalid API response: %w", err)
	}

	return nil
}

func (c *Client) GetFlag(projectKey, flagKey string) (int, map[string]any, []byte, error) {
	requestURL := fmt.Sprintf(
		"%s/flags/%s/%s",
		c.APIBaseURL,
		url.PathEscape(projectKey),
		url.PathEscape(flagKey),
	)
	statusCode, body, err := c.execRequestRaw(http.MethodGet, requestURL)
	if err != nil {
		return statusCode, nil, body, err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return statusCode, nil, body, fmt.Errorf("failed to parse response: %w", err)
	}

	return statusCode, payload, body, nil
}

func (c *Client) DeleteFlag(projectKey, flagKey string) (int, []byte, error) {
	requestURL := fmt.Sprintf(
		"%s/flags/%s/%s",
		c.APIBaseURL,
		url.PathEscape(projectKey),
		url.PathEscape(flagKey),
	)
	statusCode, body, err := c.execRequestRaw(http.MethodDelete, requestURL)
	if err != nil {
		return statusCode, body, err
	}

	return statusCode, body, nil
}

func (c *Client) execRequestRaw(method, url string) (int, []byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.Token)

	res, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res.StatusCode, body, nil
	}

	if res.StatusCode == http.StatusUnauthorized {
		return res.StatusCode, body, fmt.Errorf("LaunchDarkly credentials are invalid or expired")
	}

	if res.StatusCode == http.StatusForbidden {
		return res.StatusCode, body, fmt.Errorf("LaunchDarkly token does not have required permissions")
	}

	return res.StatusCode, body, fmt.Errorf("request failed (%d): %s", res.StatusCode, string(body))
}
