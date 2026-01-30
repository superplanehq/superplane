package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
// It returns the Cloudflare API "result" object (when available).
func (c *Client) DeleteDNSRecord(zoneID, recordID string) (map[string]any, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)
	responseBody, err := c.execRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool           `json:"success"`
		Result  map[string]any `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
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
