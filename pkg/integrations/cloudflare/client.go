package cloudflare

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
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

type PurgeCacheRequest struct {
	PurgeEverything bool     `json:"purge_everything,omitempty"`
	Files           []string `json:"files,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Hosts           []string `json:"hosts,omitempty"`
	Prefixes        []string `json:"prefixes,omitempty"`
}

type PurgeCacheResult struct {
	ID string `json:"id,omitempty"`
}

type CertificatePack struct {
	ID                   string   `json:"id,omitempty"`
	CertificateAuthority string   `json:"certificate_authority,omitempty"`
	Hosts                []string `json:"hosts,omitempty"`
	Status               string   `json:"status,omitempty"`
	Type                 string   `json:"type,omitempty"`
	ValidationMethod     string   `json:"validation_method,omitempty"`
	ValidityDays         int      `json:"validity_days,omitempty"`
}

type OrderCertificatePackRequest struct {
	CertificateAuthority string   `json:"certificate_authority"`
	Hosts                []string `json:"hosts"`
	Type                 string   `json:"type"`
	ValidationMethod     string   `json:"validation_method"`
	ValidityDays         *int     `json:"validity_days,omitempty"`
	CloudflareBranding   *bool    `json:"cloudflare_branding,omitempty"`
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
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// Enabled must not use omitempty: disabled pools must still serialize enabled:false.
	Enabled bool `json:"enabled"`
	// MinimumOrigins must not use omitempty: 0 is a valid value and must serialize as minimum_origins:0.
	MinimumOrigins int             `json:"minimum_origins"`
	Monitor        string          `json:"monitor,omitempty"`
	Origins        []Origin        `json:"origins,omitempty"`
	LoadShedding   *LoadShedding   `json:"load_shedding,omitempty"`
	OriginSteering *OriginSteering `json:"origin_steering,omitempty"`
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
	TunnelID    []string `json:"tunnel_id,omitempty"`
	NewStatus   []string `json:"new_status,omitempty"`
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

func (c *Client) UpdateMonitor(accountID, monitorID string, req CreateMonitorRequest) (*Monitor, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/monitors/%s", c.BaseURL, accountID, monitorID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, url, bytes.NewReader(body))
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
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
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
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
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
		return newCloudflareAPIError(res.StatusCode, responseBody)
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
		return newCloudflareAPIError(http.StatusOK, responseBody)
	}

	return nil
}

// KVKey represents a single key in a Workers KV namespace
type KVKey struct {
	Name string `json:"name"`
}

// ListKVNamespaces returns all Workers KV namespaces for an account
func (c *Client) ListKVNamespaces(accountID string) ([]KVNamespace, error) {
	var all []KVNamespace
	cursor := ""

	for {
		pageURL := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces?per_page=100", c.BaseURL, accountID)
		if cursor != "" {
			pageURL += "&cursor=" + url.QueryEscape(cursor)
		}

		responseBody, err := c.execRequest(http.MethodGet, pageURL, nil)
		if err != nil {
			return nil, err
		}

		var response struct {
			Success    bool          `json:"success"`
			Result     []KVNamespace `json:"result"`
			ResultInfo struct {
				Cursor string `json:"cursor"`
			} `json:"result_info"`
		}

		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		if !response.Success {
			return nil, newCloudflareAPIError(http.StatusOK, responseBody)
		}

		all = append(all, response.Result...)

		if response.ResultInfo.Cursor == "" {
			break
		}

		cursor = response.ResultInfo.Cursor
	}

	return all, nil
}

// ListKVKeys returns all keys in a Workers KV namespace
func (c *Client) ListKVKeys(accountID, namespaceID string) ([]KVKey, error) {
	var all []KVKey
	cursor := ""

	for {
		pageURL := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/keys?limit=1000", c.BaseURL, accountID, namespaceID)
		if cursor != "" {
			pageURL += "&cursor=" + url.QueryEscape(cursor)
		}

		responseBody, err := c.execRequest(http.MethodGet, pageURL, nil)
		if err != nil {
			return nil, err
		}

		var response struct {
			Success    bool    `json:"success"`
			Result     []KVKey `json:"result"`
			ResultInfo struct {
				Cursor string `json:"cursor"`
			} `json:"result_info"`
		}

		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		if !response.Success {
			return nil, newCloudflareAPIError(http.StatusOK, responseBody)
		}

		all = append(all, response.Result...)

		if response.ResultInfo.Cursor == "" {
			break
		}

		cursor = response.ResultInfo.Cursor
	}

	return all, nil
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
		return newCloudflareAPIError(http.StatusOK, responseBody)
	}

	return nil
}

// ---- Pool types ----

// Coordinates holds geographic coordinates used for proximity steering
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Origin represents a single origin server in a pool
type Origin struct {
	Name        string       `json:"name"`
	Address     string       `json:"address"`
	Enabled     bool         `json:"enabled"`
	Weight      float64      `json:"weight"`
	Port        int          `json:"port,omitempty"`
	Coordinates *Coordinates `json:"coordinates,omitempty"`
}

// OriginSteering configures how origins within a pool are selected
type OriginSteering struct {
	// Policy is one of: "random", "hash", "least_outstanding_requests", "least_connections"
	Policy string `json:"policy,omitempty"`
}

// LoadShedding configures load shedding behaviour for a pool
type LoadShedding struct {
	DefaultPercent float64 `json:"default_percent"`
	DefaultPolicy  string  `json:"default_policy"`
	SessionPercent float64 `json:"session_percent"`
	SessionPolicy  string  `json:"session_policy"`
}

// GetPool retrieves an origin pool by ID for a given account
func (c *Client) GetPool(accountID, poolID string) (*Pool, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/pools/%s", c.BaseURL, accountID, poolID)

	responseBody, err := c.execRequest(http.MethodGet, url, nil)
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

// DeletePool deletes an origin pool by ID for a given account
func (c *Client) DeletePool(accountID, poolID string) error {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/pools/%s", c.BaseURL, accountID, poolID)
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

// CreatePoolRequest is the payload for creating a pool
type CreatePoolRequest struct {
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	Enabled        bool            `json:"enabled"`
	MinimumOrigins *int            `json:"minimum_origins,omitempty"`
	Monitor        string          `json:"monitor,omitempty"`
	Origins        []Origin        `json:"origins"`
	LoadShedding   *LoadShedding   `json:"load_shedding,omitempty"`
	OriginSteering *OriginSteering `json:"origin_steering,omitempty"`
}

// CreatePool creates a new origin pool under a Cloudflare account
func (c *Client) CreatePool(accountID string, req CreatePoolRequest) (*Pool, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/pools", c.BaseURL, accountID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
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

// UpdatePoolRequest is the payload for updating an origin pool
type UpdatePoolRequest struct {
	Name           string          `json:"name,omitempty"`
	Description    string          `json:"description,omitempty"`
	Enabled        *bool           `json:"enabled,omitempty"`
	MinimumOrigins *int            `json:"minimum_origins,omitempty"`
	Monitor        string          `json:"monitor,omitempty"`
	Origins        []Origin        `json:"origins,omitempty"`
	LoadShedding   *LoadShedding   `json:"load_shedding,omitempty"`
	OriginSteering *OriginSteering `json:"origin_steering,omitempty"`
}

// UpdatePool updates an existing origin pool under a Cloudflare account
func (c *Client) UpdatePool(accountID, poolID string, req UpdatePoolRequest) (*Pool, error) {
	url := fmt.Sprintf("%s/accounts/%s/load_balancers/pools/%s", c.BaseURL, accountID, poolID)

	body, err := json.Marshal(req)
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

func (c *Client) PurgeCache(zoneID string, req PurgeCacheRequest) (*PurgeCacheResult, error) {
	url := fmt.Sprintf("%s/zones/%s/purge_cache", c.BaseURL, zoneID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool             `json:"success"`
		Result  PurgeCacheResult `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) OrderCertificatePack(zoneID string, req OrderCertificatePackRequest) (*CertificatePack, error) {
	url := fmt.Sprintf("%s/zones/%s/ssl/certificate_packs/order", c.BaseURL, zoneID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool            `json:"success"`
		Result  CertificatePack `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

func (c *Client) DeleteCertificatePack(zoneID, packID string) error {
	url := fmt.Sprintf("%s/zones/%s/ssl/certificate_packs/%s", c.BaseURL, zoneID, packID)

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

func (c *Client) ListCertificatePacks(zoneID string) ([]CertificatePack, error) {
	url := fmt.Sprintf("%s/zones/%s/ssl/certificate_packs", c.BaseURL, zoneID)

	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool              `json:"success"`
		Result  []CertificatePack `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

// ---- Load Balancer types ----

// RandomSteering holds per-pool weights used by the random steering policy
type RandomSteering struct {
	DefaultWeight float64            `json:"default_weight,omitempty"`
	PoolWeights   map[string]float64 `json:"pool_weights,omitempty"`
}

// SessionAffinityAttributes holds extra session affinity options
type SessionAffinityAttributes struct {
	SameSite     string `json:"samesite,omitempty"`
	Secure       string `json:"secure,omitempty"`
	ZeroDowntime string `json:"zero_downtime_failover,omitempty"`
}

// LBRuleOverrides defines the overrides applied when an LB rule condition is met
type LBRuleOverrides struct {
	SteeringPolicy     string          `json:"steering_policy,omitempty"`
	FallbackPool       string          `json:"fallback_pool,omitempty"`
	DefaultPools       []string        `json:"default_pools,omitempty"`
	SessionAffinity    string          `json:"session_affinity,omitempty"`
	SessionAffinityTTL *int            `json:"session_affinity_ttl,omitempty"`
	RandomSteering     *RandomSteering `json:"random_steering,omitempty"`
}

// LBRule defines a conditional override rule on a load balancer
type LBRule struct {
	Name      string          `json:"name"`
	Condition string          `json:"condition"`
	Disabled  bool            `json:"disabled,omitempty"`
	Priority  int             `json:"priority,omitempty"`
	Overrides LBRuleOverrides `json:"overrides"`
}

// LoadBalancer represents a Cloudflare Load Balancer
type LoadBalancer struct {
	ID                        string                     `json:"id"`
	Name                      string                     `json:"name"`
	Description               string                     `json:"description,omitempty"`
	Enabled                   *bool                      `json:"enabled,omitempty"`
	Proxied                   bool                       `json:"proxied"`
	TTL                       int                        `json:"ttl,omitempty"`
	FallbackPool              string                     `json:"fallback_pool"`
	DefaultPools              []string                   `json:"default_pools"`
	SteeringPolicy            string                     `json:"steering_policy,omitempty"`
	SessionAffinity           string                     `json:"session_affinity,omitempty"`
	SessionAffinityTTL        *int                       `json:"session_affinity_ttl,omitempty"`
	SessionAffinityAttributes *SessionAffinityAttributes `json:"session_affinity_attributes,omitempty"`
	RandomSteering            *RandomSteering            `json:"random_steering,omitempty"`
	Networks                  []string                   `json:"networks,omitempty"`
	Rules                     []LBRule                   `json:"rules,omitempty"`
	Monitor                   string                     `json:"monitor,omitempty"`
}

// CreateLoadBalancerRequest is the payload for creating a load balancer
type CreateLoadBalancerRequest struct {
	Name                      string                     `json:"name"`
	Description               string                     `json:"description,omitempty"`
	Enabled                   *bool                      `json:"enabled,omitempty"`
	Proxied                   bool                       `json:"proxied"`
	TTL                       int                        `json:"ttl,omitempty"`
	FallbackPool              string                     `json:"fallback_pool"`
	DefaultPools              []string                   `json:"default_pools"`
	SteeringPolicy            string                     `json:"steering_policy,omitempty"`
	SessionAffinity           string                     `json:"session_affinity,omitempty"`
	SessionAffinityTTL        *int                       `json:"session_affinity_ttl,omitempty"`
	SessionAffinityAttributes *SessionAffinityAttributes `json:"session_affinity_attributes,omitempty"`
	RandomSteering            *RandomSteering            `json:"random_steering,omitempty"`
	Networks                  []string                   `json:"networks,omitempty"`
	Rules                     []LBRule                   `json:"rules,omitempty"`
	Monitor                   string                     `json:"monitor,omitempty"`
}

// UpdateLoadBalancerRequest is the payload for patching a load balancer
type UpdateLoadBalancerRequest struct {
	Name                      string                     `json:"name,omitempty"`
	Description               string                     `json:"description,omitempty"`
	Enabled                   *bool                      `json:"enabled,omitempty"`
	SteeringPolicy            string                     `json:"steering_policy,omitempty"`
	SessionAffinity           string                     `json:"session_affinity,omitempty"`
	SessionAffinityTTL        *int                       `json:"session_affinity_ttl,omitempty"`
	SessionAffinityAttributes *SessionAffinityAttributes `json:"session_affinity_attributes,omitempty"`
	RandomSteering            *RandomSteering            `json:"random_steering,omitempty"`
	FallbackPool              string                     `json:"fallback_pool,omitempty"`
	DefaultPools              []string                   `json:"default_pools,omitempty"`
}

// ListLoadBalancers lists all load balancers for a given zone
func (c *Client) ListLoadBalancers(zoneID string) ([]LoadBalancer, error) {
	url := fmt.Sprintf("%s/zones/%s/load_balancers", c.BaseURL, zoneID)

	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool           `json:"success"`
		Result  []LoadBalancer `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return response.Result, nil
}

// GetLoadBalancer retrieves a load balancer by ID for a given zone
func (c *Client) GetLoadBalancer(zoneID, lbID string) (*LoadBalancer, error) {
	url := fmt.Sprintf("%s/zones/%s/load_balancers/%s", c.BaseURL, zoneID, lbID)

	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool         `json:"success"`
		Result  LoadBalancer `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

// CreateLoadBalancer creates a new load balancer under a zone
func (c *Client) CreateLoadBalancer(zoneID string, req CreateLoadBalancerRequest) (*LoadBalancer, error) {
	url := fmt.Sprintf("%s/zones/%s/load_balancers", c.BaseURL, zoneID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool         `json:"success"`
		Result  LoadBalancer `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

// UpdateLoadBalancer patches an existing load balancer
func (c *Client) UpdateLoadBalancer(zoneID, lbID string, req UpdateLoadBalancerRequest) (*LoadBalancer, error) {
	url := fmt.Sprintf("%s/zones/%s/load_balancers/%s", c.BaseURL, zoneID, lbID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool         `json:"success"`
		Result  LoadBalancer `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &response.Result, nil
}

// DeleteLoadBalancer deletes a load balancer by ID from a zone
func (c *Client) DeleteLoadBalancer(zoneID, lbID string) error {
	url := fmt.Sprintf("%s/zones/%s/load_balancers/%s", c.BaseURL, zoneID, lbID)
	responseBody, err := c.execRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	var response struct {
		Success bool              `json:"success"`
		Errors  []CloudflareError `json:"errors"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return &CloudflareAPIError{
			StatusCode: http.StatusOK,
			Errors:     response.Errors,
			Body:       responseBody,
		}
	}

	return nil
}

// WorkerScriptSummary is one script from GET .../workers/scripts (ID is the script name used in upload and route URLs).
type WorkerScriptSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListWorkerScripts lists uploaded Worker scripts for an account.
func (c *Client) ListWorkerScripts(accountID string) ([]WorkerScriptSummary, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/workers/scripts", c.BaseURL, accountID)
	responseBody, err := c.execRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool                  `json:"success"`
		Result  []WorkerScriptSummary `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}
	return response.Result, nil
}

const workerModuleFileName = "worker.js"

// Cloudflare's upload API treats ES module entrypoints correctly when the script part uses a
// module JavaScript media type. mime/multipart.Writer.CreateFormFile defaults to application/octet-stream,
// which can surface as validation error 10021 ("Main module must be an ES module").
const workerModulePartContentType = "application/javascript+module"

// When compatibility_date is omitted, the API falls back to a very old date; set a modern default so
// module Workers validate consistently.
const defaultWorkerUploadCompatibilityDate = "2024-01-15"

// CreateWorkerResource provisions a Worker via POST .../workers/workers before the first script upload.
func (c *Client) CreateWorkerResource(accountID string, body map[string]any) (map[string]any, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/workers/workers", c.BaseURL, accountID)

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling create worker request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, rawURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool           `json:"success"`
		Result  map[string]any `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}
	return response.Result, nil
}

// UploadWorkerScriptVersion uploads a new Worker version using multipart/form-data (POST .../versions).
func (c *Client) UploadWorkerScriptVersion(accountID, scriptName string, metadata map[string]any, moduleContent string) (string, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	if _, ok := metadata["main_module"]; !ok {
		metadata["main_module"] = workerModuleFileName
	}
	if v, ok := metadata["compatibility_date"]; !ok || strings.TrimSpace(fmt.Sprint(v)) == "" {
		metadata["compatibility_date"] = defaultWorkerUploadCompatibilityDate
	}

	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("error marshaling worker metadata: %w", err)
	}

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	if err := writer.WriteField("metadata", string(metaBytes)); err != nil {
		return "", fmt.Errorf("error writing metadata field: %w", err)
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, workerModuleFileName, workerModuleFileName))
	partHeader.Set("Content-Type", workerModulePartContentType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return "", fmt.Errorf("error creating script form part: %w", err)
	}
	if _, err := part.Write([]byte(moduleContent)); err != nil {
		return "", fmt.Errorf("error writing script content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("error closing multipart writer: %w", err)
	}

	rawURL := fmt.Sprintf("%s/accounts/%s/workers/scripts/%s/versions", c.BaseURL, accountID, url.PathEscape(scriptName))
	req, err := http.NewRequest(http.MethodPost, rawURL, buf)
	if err != nil {
		return "", fmt.Errorf("error building request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", newCloudflareAPIError(res.StatusCode, responseBody)
	}

	var response struct {
		Success bool `json:"success"`
		Result  struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return "", newCloudflareAPIError(http.StatusOK, responseBody)
	}
	if response.Result.ID == "" {
		return "", fmt.Errorf("upload response missing version id")
	}
	return response.Result.ID, nil
}

type workerDeploymentRequest struct {
	Strategy    string                    `json:"strategy"`
	Versions    []workerDeploymentVersion `json:"versions"`
	Annotations map[string]string         `json:"annotations,omitempty"`
}

type workerDeploymentVersion struct {
	Percentage int    `json:"percentage"`
	VersionID  string `json:"version_id"`
}

// CreateWorkerDeployment creates a deployment so the given version serves traffic.
func (c *Client) CreateWorkerDeployment(accountID, scriptName, versionID string, annotations map[string]string) (map[string]any, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/workers/scripts/%s/deployments", c.BaseURL, accountID, url.PathEscape(scriptName))

	reqBody := workerDeploymentRequest{
		Strategy: "percentage",
		Versions: []workerDeploymentVersion{
			{Percentage: 100, VersionID: versionID},
		},
		Annotations: annotations,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling deployment request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool           `json:"success"`
		Result  map[string]any `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}
	return response.Result, nil
}

// GetWorkerSettings returns script settings (bindings, compatibility, etc.).
func (c *Client) GetWorkerSettings(accountID, scriptName string) (map[string]any, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/workers/scripts/%s/settings", c.BaseURL, accountID, url.PathEscape(scriptName))
	responseBody, err := c.execRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool           `json:"success"`
		Result  map[string]any `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}
	return response.Result, nil
}

// ListWorkerDeployments returns deployments for a script (newest first per API).
func (c *Client) ListWorkerDeployments(accountID, scriptName string) ([]map[string]any, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/workers/scripts/%s/deployments", c.BaseURL, accountID, url.PathEscape(scriptName))
	responseBody, err := c.execRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool `json:"success"`
		Result  struct {
			Deployments []map[string]any `json:"deployments"`
		} `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}
	return response.Result.Deployments, nil
}

// DeleteWorkerScript deletes a Worker script from the account.
func (c *Client) DeleteWorkerScript(accountID, scriptName string, force bool) error {
	rawURL := fmt.Sprintf("%s/accounts/%s/workers/scripts/%s", c.BaseURL, accountID, url.PathEscape(scriptName))
	if force {
		rawURL += "?force=true"
	}
	responseBody, err := c.execRequest(http.MethodDelete, rawURL, nil)
	if err != nil {
		return err
	}

	var response struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return newCloudflareAPIError(http.StatusOK, responseBody)
	}
	return nil
}

// WorkerRoute is a zone route mapping a pattern to a Worker script name.
type WorkerRoute struct {
	ID      string `json:"id"`
	Pattern string `json:"pattern"`
	Script  string `json:"script,omitempty"`
}

type workerRouteRequest struct {
	Pattern string `json:"pattern"`
	Script  string `json:"script,omitempty"`
}

// CreateWorkerRoute creates a zone route for a Worker.
func (c *Client) CreateWorkerRoute(zoneID, pattern, script string) (*WorkerRoute, error) {
	rawURL := fmt.Sprintf("%s/zones/%s/workers/routes", c.BaseURL, zoneID)

	body, err := json.Marshal(workerRouteRequest{Pattern: pattern, Script: script})
	if err != nil {
		return nil, fmt.Errorf("error marshaling route request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool        `json:"success"`
		Result  WorkerRoute `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}
	return &response.Result, nil
}

// UpdateWorkerRoute updates an existing zone route.
func (c *Client) UpdateWorkerRoute(zoneID, routeID, pattern, script string) (*WorkerRoute, error) {
	rawURL := fmt.Sprintf("%s/zones/%s/workers/routes/%s", c.BaseURL, zoneID, routeID)

	body, err := json.Marshal(workerRouteRequest{Pattern: pattern, Script: script})
	if err != nil {
		return nil, fmt.Errorf("error marshaling route request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool        `json:"success"`
		Result  WorkerRoute `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}
	return &response.Result, nil
}
