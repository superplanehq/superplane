package hashicorp_vault

import (
	"encoding/json"
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

// KVSecretMetadata holds version info returned by the KV v2 API.
type KVSecretMetadata struct {
	Version      int    `json:"version"`
	CreatedTime  string `json:"created_time"`
	DeletionTime string `json:"deletion_time"`
	Destroyed    bool   `json:"destroyed"`
}

// KVSecret is the parsed result of a KV v2 GET request.
type KVSecret struct {
	Data     map[string]any   `json:"data"`
	Metadata KVSecretMetadata `json:"metadata"`
}

type kvSecretResponse struct {
	Data struct {
		Data     map[string]any   `json:"data"`
		Metadata KVSecretMetadata `json:"metadata"`
	} `json:"data"`
}

func (c *Client) GetKVSecret(mount, path string) (*KVSecret, error) {
	endpoint := fmt.Sprintf("/v1/%s/data/%s", mount, path)
	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response kvSecretResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &KVSecret{
		Data:     response.Data.Data,
		Metadata: response.Data.Metadata,
	}, nil
}
