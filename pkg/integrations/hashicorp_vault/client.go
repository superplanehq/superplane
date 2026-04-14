package hashicorp_vault

import (
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL   string
	Token     string
	Namespace string
	http      core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	baseURL, err := ctx.GetConfig("baseURL")
	if err != nil {
		return nil, fmt.Errorf("failed to get baseURL: %w", err)
	}

	token, err := ctx.GetConfig("token")
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	namespace, _ := ctx.GetConfig("namespace")

	return &Client{
		BaseURL:   string(baseURL),
		Token:     string(token),
		Namespace: string(namespace),
		http:      httpCtx,
	}, nil
}

func (c *Client) LookupSelf() error {
	_, err := c.execRequest(http.MethodGet, "/v1/auth/token/lookup-self", nil)
	return err
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("X-Vault-Token", c.Token)
	req.Header.Set("Content-Type", "application/json")
	if c.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.Namespace)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request got %d: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}
