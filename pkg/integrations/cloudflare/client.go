package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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

type CloudflareResponseInfo struct {
	Code             int    `json:"code,omitempty"`
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url,omitempty"`
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

type Monitor struct {
	ID              string              `json:"id,omitempty"`
	AllowInsecure   *bool               `json:"allow_insecure,omitempty"`
	ConsecutiveDown *int                `json:"consecutive_down,omitempty"`
	ConsecutiveUp   *int                `json:"consecutive_up,omitempty"`
	CreatedOn       string              `json:"created_on,omitempty"`
	Description     string              `json:"description,omitempty"`
	ExpectedBody    string              `json:"expected_body,omitempty"`
	ExpectedCodes   string              `json:"expected_codes,omitempty"`
	FollowRedirects *bool               `json:"follow_redirects,omitempty"`
	Header          map[string][]string `json:"header,omitempty"`
	Interval        *int                `json:"interval,omitempty"`
	Method          string              `json:"method,omitempty"`
	ModifiedOn      string              `json:"modified_on,omitempty"`
	Path            string              `json:"path,omitempty"`
	Port            *int                `json:"port,omitempty"`
	ProbeZone       string              `json:"probe_zone,omitempty"`
	Retries         *int                `json:"retries,omitempty"`
	Timeout         *int                `json:"timeout,omitempty"`
	Type            string              `json:"type,omitempty"`
}

type CreateMonitorRequest struct {
	AllowInsecure   *bool               `json:"allow_insecure,omitempty"`
	ConsecutiveDown *int                `json:"consecutive_down,omitempty"`
	ConsecutiveUp   *int                `json:"consecutive_up,omitempty"`
	Description     string              `json:"description,omitempty"`
	ExpectedBody    string              `json:"expected_body,omitempty"`
	ExpectedCodes   string              `json:"expected_codes,omitempty"`
	FollowRedirects *bool               `json:"follow_redirects,omitempty"`
	Header          map[string][]string `json:"header,omitempty"`
	Interval        *int                `json:"interval,omitempty"`
	Method          string              `json:"method,omitempty"`
	Path            string              `json:"path,omitempty"`
	Port            *int                `json:"port,omitempty"`
	ProbeZone       string              `json:"probe_zone,omitempty"`
	Retries         *int                `json:"retries,omitempty"`
	Timeout         *int                `json:"timeout,omitempty"`
	Type            string              `json:"type"`
}

type DeleteMonitorResponse struct {
	ID string `json:"id,omitempty"`
}

type MonitorReference struct {
	ReferenceType string `json:"reference_type,omitempty"`
	ResourceID    string `json:"resource_id,omitempty"`
	ResourceName  string `json:"resource_name,omitempty"`
	ResourceType  string `json:"resource_type,omitempty"`
}

type Pool struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Monitor string `json:"monitor,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type AlertingWebhookDestination struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type CreateAlertingWebhookDestinationRequest struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

type CreateNotificationPolicyRequest struct {
	AlertType   string                       `json:"alert_type"`
	Enabled     bool                         `json:"enabled"`
	Mechanisms  NotificationPolicyMechanisms `json:"mechanisms"`
	Name        string                       `json:"name"`
	Description string                       `json:"description,omitempty"`
	Filters     NotificationPolicyFilters    `json:"filters,omitempty"`
}

type NotificationPolicyMechanisms struct {
	Email     []NotificationMechanism `json:"email,omitempty"`
	PagerDuty []NotificationMechanism `json:"pagerduty,omitempty"`
	Webhooks  []NotificationMechanism `json:"webhooks,omitempty"`
}

type NotificationMechanism struct {
	ID string `json:"id,omitempty"`
}

type NotificationPolicyFilters struct {
	PoolID      []string `json:"pool_id,omitempty"`
	NewHealth   []string `json:"new_health,omitempty"`
	EventSource []string `json:"event_source,omitempty"`
}

type NotificationPolicyResponse struct {
	ID string `json:"id,omitempty"`
}

func accountIDForIntegration(ctx core.IntegrationContext) (string, error) {
	accountID, err := ctx.GetConfig("accountId")
	if err != nil {
		return "", fmt.Errorf("accountId is required")
	}

	value := strings.TrimSpace(string(accountID))
	if value == "" {
		return "", fmt.Errorf("accountId is required")
	}

	return value, nil
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

func (c *Client) ListMonitors(accountID string) ([]Monitor, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/monitors", c.BaseURL, accountID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool      `json:"success"`
		Result  []Monitor `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

func (c *Client) GetMonitor(accountID, monitorID string) (*Monitor, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/monitors/%s", c.BaseURL, accountID, monitorID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool    `json:"success"`
		Result  Monitor `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) CreateMonitor(accountID string, req CreateMonitorRequest) (*Monitor, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/monitors", c.BaseURL, accountID)

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
		Result  Monitor `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) DeleteMonitor(accountID, monitorID string) (*DeleteMonitorResponse, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/monitors/%s", c.BaseURL, accountID, monitorID)
	responseBody, err := c.execRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool                  `json:"success"`
		Result  DeleteMonitorResponse `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) ListMonitorReferences(accountID, monitorID string) ([]MonitorReference, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/monitors/%s/references", c.BaseURL, accountID, monitorID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool               `json:"success"`
		Result  []MonitorReference `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

func (c *Client) ListPools(accountID string) ([]Pool, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/pools", c.BaseURL, accountID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool   `json:"success"`
		Result  []Pool `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

func (c *Client) PatchPoolMonitor(accountID, poolID, monitorID string) (*Pool, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/pools/%s", c.BaseURL, accountID, poolID)
	body, err := json.Marshal(map[string]string{"monitor": monitorID})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool `json:"success"`
		Result  Pool `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) CreateAlertingWebhookDestination(
	accountID string,
	req CreateAlertingWebhookDestinationRequest,
) (*AlertingWebhookDestination, error) {
	url := fmt.Sprintf("%s/accounts/%s/alerting/v3/destinations/webhooks", c.BaseURL, accountID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool                       `json:"success"`
		Result  AlertingWebhookDestination `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) DeleteAlertingWebhookDestination(accountID, webhookID string) error {
	url := fmt.Sprintf("%s/accounts/%s/alerting/v3/destinations/webhooks/%s", c.BaseURL, accountID, webhookID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

func (c *Client) CreateNotificationPolicy(
	accountID string,
	req CreateNotificationPolicyRequest,
) (*NotificationPolicyResponse, error) {
	url := fmt.Sprintf("%s/accounts/%s/alerting/v3/policies", c.BaseURL, accountID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool                       `json:"success"`
		Result  NotificationPolicyResponse `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) DeleteNotificationPolicy(accountID, policyID string) error {
	url := fmt.Sprintf("%s/accounts/%s/alerting/v3/policies/%s", c.BaseURL, accountID, policyID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
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
