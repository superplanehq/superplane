package cloudflare

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.cloudflare.com/client/v4"

type Client struct {
	Token   string
	http    core.HTTPContext
	BaseURL string
}

type CloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CloudflareAPIError struct {
	StatusCode int
	Errors     []CloudflareError
	Body       []byte
}

func (e *CloudflareAPIError) Error() string {
	return fmt.Sprintf("request got %d code: %s", e.StatusCode, string(e.Body))
}

func isCloudflareNotFound(err error) bool {
	apiErr := (*CloudflareAPIError)(nil)
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error finding API token: %v", err)
	}

	return &Client{
		Token:   string(apiToken),
		http:    http,
		BaseURL: baseURL,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	statusCode, responseBody, err := c.execRequestRaw(method, url, body)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, newCloudflareAPIError(statusCode, responseBody)
	}

	return responseBody, nil
}

func (c *Client) execRequestRaw(method, url string, body io.Reader) (int, []byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	res, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, fmt.Errorf("error reading body: %v", err)
	}

	return res.StatusCode, responseBody, nil
}

func newCloudflareAPIError(statusCode int, responseBody []byte) *CloudflareAPIError {
	apiError := &CloudflareAPIError{
		StatusCode: statusCode,
		Body:       responseBody,
	}

	var payload struct {
		Errors []CloudflareError `json:"errors"`
	}

	if err := json.Unmarshal(responseBody, &payload); err == nil {
		apiError.Errors = payload.Errors
	}

	return apiError
}

// Zone represents a Cloudflare zone
type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ListZones retrieves all zones accessible with the API token
func (c *Client) ListZones() ([]Zone, error) {
	url := fmt.Sprintf("%s/zones", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool   `json:"success"`
		Result  []Zone `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

// DeleteDNSRecord deletes a DNS record by its ID within a zone.
// It returns the deleted DNS record (Cloudflare API returns it in result).
func (c *Client) DeleteDNSRecord(zoneID, recordID string) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)
	responseBody, err := c.execRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool      `json:"success"`
		Result  DNSRecord `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

// RedirectRule represents a single redirect rule in a ruleset
type RedirectRule struct {
	ID          string              `json:"id,omitempty"`
	Action      string              `json:"action"`
	Expression  string              `json:"expression"`
	Description string              `json:"description,omitempty"`
	Enabled     bool                `json:"enabled"`
	ActionParam *RedirectActionData `json:"action_parameters,omitempty"`
}

// RedirectActionData contains the redirect configuration
type RedirectActionData struct {
	FromValue *RedirectFromValue `json:"from_value,omitempty"`
}

// RedirectFromValue defines the redirect target
type RedirectFromValue struct {
	StatusCode       int                `json:"status_code"`
	TargetURL        *RedirectTargetURL `json:"target_url,omitempty"`
	PreserveQueryStr bool               `json:"preserve_query_string,omitempty"`
}

// RedirectTargetURL defines the target URL for redirect
type RedirectTargetURL struct {
	Value      string `json:"value,omitempty"`
	Expression string `json:"expression,omitempty"`
}

// Ruleset represents a Cloudflare ruleset
type Ruleset struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Kind        string         `json:"kind"`
	Phase       string         `json:"phase"`
	Rules       []RedirectRule `json:"rules"`
}

// OriginRule represents a single origin rule in a ruleset.
type OriginRule struct {
	ID          string                  `json:"id,omitempty"`
	Action      string                  `json:"action"`
	Expression  string                  `json:"expression"`
	Description string                  `json:"description,omitempty"`
	Enabled     bool                    `json:"enabled"`
	ActionParam *OriginActionParameters `json:"action_parameters,omitempty"`
}

type OriginActionParameters struct {
	HostHeader string         `json:"host_header,omitempty"`
	Origin     *RouteOrigin   `json:"origin,omitempty"`
	SNI        *RouteSNIValue `json:"sni,omitempty"`
}

type RouteOrigin struct {
	Host string `json:"host,omitempty"`
	Port *int   `json:"port,omitempty"`
}

type RouteSNIValue struct {
	Value string `json:"value,omitempty"`
}

type OriginRuleset struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Kind        string       `json:"kind"`
	Phase       string       `json:"phase"`
	Rules       []OriginRule `json:"rules"`
}

type CreateOriginRulesetRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Kind        string       `json:"kind"`
	Phase       string       `json:"phase"`
	Rules       []OriginRule `json:"rules,omitempty"`
}

// ListRedirectRules retrieves all redirect rules for a zone
func (c *Client) ListRedirectRules(zoneID string) ([]RedirectRule, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets/phases/http_request_dynamic_redirect/entrypoint", c.BaseURL, zoneID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool    `json:"success"`
		Result  Ruleset `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Result.Rules, nil
}

// UpdateRedirectRuleRequest is the payload for updating a redirect rule
type UpdateRedirectRuleRequest struct {
	Action      string              `json:"action"`
	Expression  string              `json:"expression"`
	Description string              `json:"description,omitempty"`
	Enabled     bool                `json:"enabled"`
	ActionParam *RedirectActionData `json:"action_parameters,omitempty"`
}

// UpdateRedirectRule updates a specific redirect rule in a zone's ruleset
func (c *Client) UpdateRedirectRule(zoneID, rulesetID, ruleID string, req UpdateRedirectRuleRequest) (*RedirectRule, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets/%s/rules/%s", c.BaseURL, zoneID, rulesetID, ruleID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool    `json:"success"`
		Result  Ruleset `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	// Find the updated rule in the result
	for _, rule := range response.Result.Rules {
		if rule.ID == ruleID {
			return &rule, nil
		}
	}

	return nil, fmt.Errorf("updated rule not found in response")
}

// GetRulesetForPhase gets the ruleset ID for a specific phase in a zone
func (c *Client) GetRulesetForPhase(zoneID, phase string) (*Ruleset, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets/phases/%s/entrypoint", c.BaseURL, zoneID, phase)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool    `json:"success"`
		Result  Ruleset `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) GetOriginRulesetForPhase(zoneID string) (*OriginRuleset, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets/phases/http_request_origin/entrypoint", c.BaseURL, zoneID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool          `json:"success"`
		Result  OriginRuleset `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) CreateOriginRuleset(zoneID string, req CreateOriginRulesetRequest) (*OriginRuleset, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets", c.BaseURL, zoneID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool          `json:"success"`
		Result  OriginRuleset `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) ListOriginRules(zoneID string) ([]OriginRule, error) {
	ruleset, err := c.GetOriginRulesetForPhase(zoneID)
	if err != nil {
		return nil, err
	}

	return ruleset.Rules, nil
}

func (c *Client) CreateOriginRule(zoneID, rulesetID string, req OriginRule) (*OriginRule, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets/%s/rules", c.BaseURL, zoneID, rulesetID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return originRuleFromRulesetResponse(responseBody, "")
}

func (c *Client) UpdateOriginRule(zoneID, rulesetID, ruleID string, req OriginRule) (*OriginRule, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets/%s/rules/%s", c.BaseURL, zoneID, rulesetID, ruleID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return originRuleFromRulesetResponse(responseBody, ruleID)
}

func (c *Client) DeleteOriginRule(zoneID, rulesetID, ruleID string) (*OriginRule, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets/%s/rules/%s", c.BaseURL, zoneID, rulesetID, ruleID)
	responseBody, err := c.execRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &OriginRule{ID: ruleID}, nil
}

func originRuleFromRulesetResponse(responseBody []byte, ruleID string) (*OriginRule, error) {
	var response struct {
		Success bool          `json:"success"`
		Result  OriginRuleset `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	if ruleID != "" {
		for _, rule := range response.Result.Rules {
			if rule.ID == ruleID {
				return &rule, nil
			}
		}

		return nil, fmt.Errorf("origin rule not found in response")
	}

	if len(response.Result.Rules) > 0 {
		return &response.Result.Rules[len(response.Result.Rules)-1], nil
	}

	return nil, fmt.Errorf("origin rule not found in response")
}

// DNSRecord represents a Cloudflare DNS record
type DNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Proxied  bool   `json:"proxied"`
	Priority *int   `json:"priority,omitempty"`
}

// ListDNSRecords retrieves all DNS records for a zone
func (c *Client) ListDNSRecords(zoneID string) ([]DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records", c.BaseURL, zoneID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool        `json:"success"`
		Result  []DNSRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

// GetDNSRecord retrieves a DNS record by ID from a zone
func (c *Client) GetDNSRecord(zoneID, recordID string) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool      `json:"success"`
		Result  DNSRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

// UpdateDNSRecordRequest is the payload for updating a DNS record.
// Cloudflare's Update DNS Record endpoint expects a full record object (type, name, content, ttl, proxied).
type UpdateDNSRecordRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// UpdateDNSRecord updates an existing DNS record in a zone.
func (c *Client) UpdateDNSRecord(zoneID, recordID string, req UpdateDNSRecordRequest) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool      `json:"success"`
		Result  DNSRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

// CreateRedirectRuleRequest is the payload for creating a new redirect rule
type CreateRedirectRuleRequest struct {
	Action      string              `json:"action"`
	Expression  string              `json:"expression"`
	Description string              `json:"description,omitempty"`
	Enabled     bool                `json:"enabled"`
	ActionParam *RedirectActionData `json:"action_parameters,omitempty"`
}

// CreateRedirectRule creates a new redirect rule in a zone's ruleset
func (c *Client) CreateRedirectRule(zoneID, rulesetID string, req CreateRedirectRuleRequest) (*RedirectRule, error) {
	url := fmt.Sprintf("%s/zones/%s/rulesets/%s/rules", c.BaseURL, zoneID, rulesetID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool    `json:"success"`
		Result  Ruleset `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	// Return the last rule (newly created)
	if len(response.Result.Rules) > 0 {
		return &response.Result.Rules[len(response.Result.Rules)-1], nil
	}

	return nil, fmt.Errorf("created rule not found in response")
}

type CreateDNSRecordRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      *int   `json:"ttl,omitempty"`
	Proxied  *bool  `json:"proxied,omitempty"`
	Priority *int   `json:"priority,omitempty"`
}

func (c *Client) CreateDNSRecord(zoneID string, req CreateDNSRecordRequest) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records", c.BaseURL, zoneID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool              `json:"success"`
		Result  DNSRecord         `json:"result"`
		Errors  []CloudflareError `json:"errors"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, &CloudflareAPIError{
			StatusCode: http.StatusOK,
			Errors:     response.Errors,
			Body:       responseBody,
		}
	}

	return &response.Result, nil
}

// ---- Workers KV types ----

// KVNamespace represents a Cloudflare Workers KV namespace
type KVNamespace struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	SupportsURLEncoding *bool  `json:"supports_url_encoding,omitempty"`
}

// CreateKVNamespaceRequest is the payload for creating a KV namespace
type CreateKVNamespaceRequest struct {
	Title string `json:"title"`
}

// CreateKVNamespace creates a new Workers KV namespace under a Cloudflare account
func (c *Client) CreateKVNamespace(accountID string, req CreateKVNamespaceRequest) (*KVNamespace, error) {
	kvURL := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces", c.BaseURL, accountID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, kvURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var kvResponse struct {
		Success bool        `json:"success"`
		Result  KVNamespace `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &kvResponse); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !kvResponse.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &kvResponse.Result, nil
}

// GetKVNamespace retrieves a single Workers KV namespace by ID
func (c *Client) GetKVNamespace(accountID, namespaceID string) (*KVNamespace, error) {
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s", c.BaseURL, accountID, namespaceID)

	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool        `json:"success"`
		Result  KVNamespace `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

// PutKVValue writes a key-value pair to a Workers KV namespace using multipart/form-data
func (c *Client) PutKVValue(accountID, namespaceID, key, value string, expirationTTL *int) error {
	rawURL := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s", c.BaseURL, accountID, namespaceID, url.PathEscape(key))

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	if err := writer.WriteField("value", value); err != nil {
		return fmt.Errorf("error writing value field: %v", err)
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPut, rawURL, buf)
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	if expirationTTL != nil {
		q := req.URL.Query()
		q.Set("expiration_ttl", fmt.Sprintf("%d", *expirationTTL))
		req.URL.RawQuery = q.Encode()
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return newCloudflareAPIError(res.StatusCode, responseBody)
	}

	var response struct {
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("API returned success=false")
	}

	return nil
}

// GetKVValue retrieves the value for a key from a Workers KV namespace.
// The Cloudflare API returns the raw value as the response body (not a JSON envelope).
func (c *Client) GetKVValue(accountID, namespaceID, key string) (string, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s", c.BaseURL, accountID, namespaceID, url.PathEscape(key))

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", newCloudflareAPIError(res.StatusCode, responseBody)
	}

	return string(responseBody), nil
}

// DeleteKVValue deletes a key-value pair from a Workers KV namespace
func (c *Client) DeleteKVValue(accountID, namespaceID, key string) error {
	kvURL := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s", c.BaseURL, accountID, namespaceID, url.PathEscape(key))

	responseBody, err := c.execRequest(http.MethodDelete, kvURL, nil)
	if err != nil {
		return err
	}

	var deleteResponse struct {
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(responseBody, &deleteResponse); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !deleteResponse.Success {
		return fmt.Errorf("API returned success=false")
	}

	return nil
}

// KVKey represents a single key in a Workers KV namespace
type KVKey struct {
	Name string `json:"name"`
}

// ListKVNamespaces returns all Workers KV namespaces for an account
func (c *Client) ListKVNamespaces(accountID string) ([]KVNamespace, error) {
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces?per_page=100", c.BaseURL, accountID)

	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool          `json:"success"`
		Result  []KVNamespace `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

// ListKVKeys returns all keys in a Workers KV namespace
func (c *Client) ListKVKeys(accountID, namespaceID string) ([]KVKey, error) {
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/keys?limit=1000", c.BaseURL, accountID, namespaceID)

	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool    `json:"success"`
		Result  []KVKey `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

// DeleteKVNamespace deletes a Workers KV namespace
func (c *Client) DeleteKVNamespace(accountID, namespaceID string) error {
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s", c.BaseURL, accountID, namespaceID)

	responseBody, err := c.execRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	var response struct {
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("API returned success=false")
	}

	return nil
}
