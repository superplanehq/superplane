package elastic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	baseURL   string
	kibanaURL string
	authType  string
	apiKey    string
	username  string
	password  string
	http      core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	serverURL, err := ctx.GetConfig("url")
	if err != nil {
		return nil, fmt.Errorf("error getting url: %v", err)
	}

	authType, err := ctx.GetConfig("authType")
	if err != nil {
		return nil, fmt.Errorf("error getting authType: %v", err)
	}

	c := &Client{
		baseURL:  strings.TrimRight(string(serverURL), "/"),
		authType: string(authType),
		http:     httpCtx,
	}

	if kibanaURL, err := ctx.GetConfig("kibanaUrl"); err == nil {
		c.kibanaURL = strings.TrimRight(string(kibanaURL), "/")
	}

	switch c.authType {
	case "apiKey":
		apiKey, err := ctx.GetConfig("apiKey")
		if err != nil {
			return nil, fmt.Errorf("error getting apiKey: %v", err)
		}
		c.apiKey = string(apiKey)
	case "basic":
		username, err := ctx.GetConfig("username")
		if err != nil {
			return nil, fmt.Errorf("error getting username: %v", err)
		}
		password, err := ctx.GetConfig("password")
		if err != nil {
			return nil, fmt.Errorf("error getting password: %v", err)
		}
		c.username = string(username)
		c.password = string(password)
	}

	return c, nil
}

func (c *Client) setAuthHeaders(req *http.Request) {
	switch c.authType {
	case "apiKey":
		req.Header.Set("Authorization", "ApiKey "+c.apiKey)
	case "basic":
		req.SetBasicAuth(c.username, c.password)
	}
}

func (c *Client) execRequest(method, fullURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(req)

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
		return nil, fmt.Errorf("request failed (%d): %s", res.StatusCode, redactedResponseHint(responseBody))
	}

	return responseBody, nil
}

// KibanaAPIError is returned by execKibanaRequest for non-2xx responses so
// callers can check the status code (e.g. treat 404 as a no-op on delete).
type KibanaAPIError struct {
	StatusCode int
	Body       string
}

func (e *KibanaAPIError) Error() string {
	return fmt.Sprintf("Kibana request failed (%d): %s", e.StatusCode, e.Body)
}

// execKibanaRequest is like execRequest but targets the Kibana URL and adds
// the kbn-xsrf header required by all Kibana write endpoints.
func (c *Client) execKibanaRequest(method, path string, body io.Reader) ([]byte, error) {
	if c.kibanaURL == "" {
		return nil, fmt.Errorf("kibana URL is not configured")
	}

	req, err := http.NewRequest(method, c.kibanaURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("error building Kibana request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("kbn-xsrf", "true")
	c.setAuthHeaders(req)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing Kibana request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading Kibana response body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &KibanaAPIError{StatusCode: res.StatusCode, Body: redactedResponseHint(responseBody)}
	}

	return responseBody, nil
}

// ValidateCredentials checks that the configured URL and credentials are valid
// by performing a GET / against the cluster info endpoint.
func (c *Client) ValidateCredentials() error {
	_, err := c.execRequest(http.MethodGet, c.baseURL+"/", nil)
	return err
}

// ValidateKibana checks that the Kibana URL is reachable and that the
// credentials have permission to manage connectors (required for webhook setup).
func (c *Client) ValidateKibana() error {
	_, err := c.execKibanaRequest(http.MethodGet, "/api/actions/connectors", nil)
	return err
}

// KibanaRule is the relevant subset of a Kibana alerting rule.
type KibanaRule struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type kibanaRulesResponse struct {
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
	Total   int          `json:"total"`
	Data    []KibanaRule `json:"data"`
}

// ListKibanaRules returns all alerting rules from Kibana, paginating as needed.
func (c *Client) ListKibanaRules() ([]KibanaRule, error) {
	const perPage = 100
	const maxPages = 100
	var all []KibanaRule

	for page := 1; page <= maxPages; page++ {
		path := fmt.Sprintf("/api/alerting/rules/_find?per_page=%d&page=%d", perPage, page)
		body, err := c.execKibanaRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var resp kibanaRulesResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("error parsing Kibana rules response: %v", err)
		}

		if len(resp.Data) == 0 {
			if resp.Total == 0 || len(all) >= resp.Total {
				return all, nil
			}
			return nil, fmt.Errorf("received empty Kibana rules page %d before reaching reported total %d", page, resp.Total)
		}

		all = append(all, resp.Data...)
		if len(all) >= resp.Total {
			return all, nil
		}
	}

	return nil, fmt.Errorf("exceeded maximum Kibana rule pages (%d)", maxPages)
}

// KibanaSpace is the relevant subset of a Kibana space.
type KibanaSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListKibanaSpaces returns all spaces from Kibana.
func (c *Client) ListKibanaSpaces() ([]KibanaSpace, error) {
	body, err := c.execKibanaRequest(http.MethodGet, "/api/spaces/space", nil)
	if err != nil {
		return nil, err
	}

	var spaces []KibanaSpace
	if err := json.Unmarshal(body, &spaces); err != nil {
		return nil, fmt.Errorf("error parsing Kibana spaces response: %v", err)
	}

	return spaces, nil
}

// KibanaConnectorResponse is the relevant subset of the Kibana connector API response.
type KibanaConnectorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateKibanaConnector creates a Kibana Webhook connector that POSTs to
// webhookURL and includes the signing secret as the X-Superplane-Secret header.
func (c *Client) CreateKibanaConnector(name, webhookURL, secret string) (*KibanaConnectorResponse, error) {
	payload := map[string]any{
		"connector_type_id": ".webhook",
		"name":              name,
		"config": map[string]any{
			"url":    webhookURL,
			"method": "post",
			"headers": map[string]string{
				"Content-Type":    "application/json",
				SigningHeaderName: secret,
			},
			"hasAuth": false,
		},
		"secrets": map[string]any{},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling connector payload: %v", err)
	}

	responseBody, err := c.execKibanaRequest(http.MethodPost, "/api/actions/connector", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var resp KibanaConnectorResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing Kibana connector response: %v", err)
	}

	return &resp, nil
}

// DeleteKibanaConnector removes a Kibana connector by ID.
// A 404 response is treated as success: the connector is already gone.
func (c *Client) DeleteKibanaConnector(connectorID string) error {
	_, err := c.execKibanaRequest(
		http.MethodDelete,
		fmt.Sprintf("/api/actions/connector/%s", url.PathEscape(connectorID)),
		nil,
	)
	var kibanaErr *KibanaAPIError
	if errors.As(err, &kibanaErr) && kibanaErr.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}

// redactedResponseHint returns a safe hint for inclusion in returned errors
// without exposing raw upstream response bodies.
func redactedResponseHint(b []byte) string {
	if len(b) == 0 {
		return "response body omitted"
	}
	return fmt.Sprintf("response body omitted (%d bytes)", len(b))
}

// IndexInfo holds the minimal fields returned by GET /_cat/indices.
type IndexInfo struct {
	Index string `json:"index"`
}

// ListIndices returns all user-facing indices from the cluster, excluding
// dot-prefixed system indices (e.g. .kibana, .security-*).
func (c *Client) ListIndices() ([]IndexInfo, error) {
	responseBody, err := c.execRequest(http.MethodGet, c.baseURL+"/_cat/indices?format=json&h=index", nil)
	if err != nil {
		return nil, err
	}

	var all []IndexInfo
	if err := json.Unmarshal(responseBody, &all); err != nil {
		return nil, fmt.Errorf("error parsing indices response: %v", err)
	}

	indices := make([]IndexInfo, 0, len(all))
	for _, idx := range all {
		if !strings.HasPrefix(idx.Index, ".") {
			indices = append(indices, idx)
		}
	}

	return indices, nil
}

// IndexDocumentResponse represents the Elasticsearch index/create response.
type IndexDocumentResponse struct {
	ID      string `json:"_id"`
	Index   string `json:"_index"`
	Result  string `json:"result"`
	Version int    `json:"_version"`
	Shards  struct {
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
}

// IndexDocument writes doc to the given index. If documentID is non-empty the
// document is written at that ID (PUT, enabling idempotent writes); otherwise
// Elasticsearch generates an ID (POST).
func (c *Client) IndexDocument(index, documentID string, doc map[string]any) (*IndexDocumentResponse, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("error marshaling document: %v", err)
	}

	var fullURL string
	var method string
	if documentID != "" {
		fullURL = fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, url.PathEscape(index), url.PathEscape(documentID))
		method = http.MethodPut
	} else {
		fullURL = fmt.Sprintf("%s/%s/_doc", c.baseURL, url.PathEscape(index))
		method = http.MethodPost
	}

	responseBody, err := c.execRequest(method, fullURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var resp IndexDocumentResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &resp, nil
}
