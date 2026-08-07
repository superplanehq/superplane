package jenkins

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL  string
	Username string
	APIToken string
	http     core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	baseURL, err := ctx.GetConfig("baseUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting baseUrl: %w", err)
	}

	username, err := ctx.GetConfig("username")
	if err != nil {
		return nil, fmt.Errorf("error getting username: %w", err)
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting apiToken: %w", err)
	}

	return &Client{
		BaseURL:  strings.TrimRight(string(baseURL), "/"),
		Username: string(username),
		APIToken: string(apiToken),
		http:     http,
	}, nil
}

func (c *Client) execRequest(method, path string, body io.Reader) (int, []byte, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, path), body)
	if err != nil {
		return 0, nil, fmt.Errorf("error building request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.APIToken)

	res, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("error reading body: %w", err)
	}

	return res.StatusCode, responseBody, nil
}

// Verify checks that the configured Base URL and credentials are valid.
func (c *Client) Verify() error {
	statusCode, body, err := c.execRequest(http.MethodGet, "/api/json", nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return statusError(statusCode, body)
	}

	return nil
}

const maxErrorBodyLen = 200

// statusError builds a user-facing error for a non-2xx Jenkins response.
// Jenkins renders most errors as Jetty's default HTML error page, which is
// noisy and unhelpful to show as-is, so known status codes get a clean
// message and anything else has its body truncated/sanitized.
func statusError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed (401): check your Jenkins username and API token")
	case http.StatusForbidden:
		return fmt.Errorf("access denied (403): check your Jenkins username and API token permissions")
	case http.StatusNotFound:
		return fmt.Errorf("not found (404): check your Jenkins Base URL")
	default:
		return fmt.Errorf("request got %d: %s", statusCode, sanitizeErrorBody(body))
	}
}

func sanitizeErrorBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "(empty response)"
	}

	if strings.HasPrefix(text, "<") {
		return "(non-JSON response)"
	}

	if len(text) > maxErrorBodyLen {
		return text[:maxErrorBodyLen] + "..."
	}

	return text
}
