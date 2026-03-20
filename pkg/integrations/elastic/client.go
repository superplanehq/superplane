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

	authType := "apiKey"
	if configuredAuthType, err := ctx.GetConfig("authType"); err == nil && string(configuredAuthType) != "" {
		authType = string(configuredAuthType)
	}

	c := &Client{
		baseURL:  strings.TrimRight(string(serverURL), "/"),
		authType: authType,
		http:     httpCtx,
	}

	switch authType {
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
	default:
		return nil, fmt.Errorf("unsupported authType %q", authType)
	}

	if kibanaURL, err := ctx.GetConfig("kibanaUrl"); err == nil {
		c.kibanaURL = strings.TrimRight(string(kibanaURL), "/")
	}

	return c, nil
}

func (c *Client) setAuthHeaders(req *http.Request) {
	if c.authType == "basic" {
		req.SetBasicAuth(c.username, c.password)
		return
	}

	req.Header.Set("Authorization", "ApiKey "+c.apiKey)
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
		body := redactedResponseHint(responseBody)
		if res.StatusCode >= 400 && res.StatusCode < 500 {
			body = string(responseBody)
		}
		return nil, &KibanaAPIError{StatusCode: res.StatusCode, Body: body}
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

type KibanaRuleActionFrequency struct {
	NotifyWhen string  `json:"notify_when,omitempty"`
	Summary    bool    `json:"summary"`
	Throttle   *string `json:"throttle"`
}

type KibanaRuleAction struct {
	ID                      string                     `json:"id"`
	Group                   string                     `json:"group"`
	Params                  map[string]any             `json:"params"`
	Frequency               *KibanaRuleActionFrequency `json:"frequency,omitempty"`
	UseAlertDataForTemplate bool                       `json:"use_alert_data_for_template,omitempty"`
	UUID                    string                     `json:"uuid,omitempty"`
	AlertsFilter            map[string]any             `json:"alerts_filter,omitempty"`
}

type KibanaRuleSchedule struct {
	Interval string `json:"interval"`
}

type KibanaRuleFlapping struct {
	LookBackWindow        int `json:"look_back_window"`
	StatusChangeThreshold int `json:"status_change_threshold"`
}

type KibanaRuleAlertDelay struct {
	Active int `json:"active"`
}

type KibanaRuleDetails struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Consumer   string                `json:"consumer"`
	Params     map[string]any        `json:"params"`
	RuleTypeID string                `json:"rule_type_id"`
	Schedule   KibanaRuleSchedule    `json:"schedule"`
	Tags       []string              `json:"tags"`
	Actions    []KibanaRuleAction    `json:"actions"`
	AlertDelay *KibanaRuleAlertDelay `json:"alert_delay,omitempty"`
}

type updateKibanaRuleRequest struct {
	Name       string                `json:"name"`
	Params     map[string]any        `json:"params"`
	Schedule   KibanaRuleSchedule    `json:"schedule"`
	Tags       []string              `json:"tags"`
	Actions    []KibanaRuleAction    `json:"actions"`
	AlertDelay *KibanaRuleAlertDelay `json:"alert_delay,omitempty"`
}

type KibanaRuleType struct {
	ID                   string `json:"id"`
	DefaultActionGroupID string `json:"default_action_group_id"`
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

func (c *Client) GetKibanaRule(ruleID string) (*KibanaRuleDetails, error) {
	body, err := c.execKibanaRequest(http.MethodGet, fmt.Sprintf("/api/alerting/rule/%s", url.PathEscape(ruleID)), nil)
	if err != nil {
		return nil, err
	}

	var rule KibanaRuleDetails
	if err := json.Unmarshal(body, &rule); err != nil {
		return nil, fmt.Errorf("error parsing Kibana rule response: %v", err)
	}

	return &rule, nil
}

func (c *Client) EnsureKibanaRuleHasConnector(ruleID, connectorID string) error {
	return c.updateKibanaRuleWithRetry(ruleID, func(rule *KibanaRuleDetails) error {
		for _, action := range rule.Actions {
			if action.ID == connectorID {
				return nil
			}
		}

		actionGroupID, err := c.GetKibanaRuleDefaultActionGroupID(rule.RuleTypeID)
		if err != nil {
			return err
		}

		rule.Actions = append(rule.Actions, superPlaneKibanaRuleAction(connectorID, actionGroupID))
		return c.updateKibanaRule(ruleID, rule)
	})
}

func (c *Client) RemoveKibanaRuleConnector(ruleID, connectorID string) error {
	return c.updateKibanaRuleWithRetry(ruleID, func(rule *KibanaRuleDetails) error {
		filtered := make([]KibanaRuleAction, 0, len(rule.Actions))
		for _, action := range rule.Actions {
			if action.ID != connectorID {
				filtered = append(filtered, action)
			}
		}

		if len(filtered) == len(rule.Actions) {
			return nil
		}

		rule.Actions = filtered
		return c.updateKibanaRule(ruleID, rule)
	})
}

func (c *Client) updateKibanaRule(ruleID string, rule *KibanaRuleDetails) error {
	params := rule.Params
	if params == nil {
		params = map[string]any{}
	}

	tags := rule.Tags
	if tags == nil {
		tags = []string{}
	}

	actions := rule.Actions
	if actions == nil {
		actions = []KibanaRuleAction{}
	}

	payload := updateKibanaRuleRequest{
		Name:       rule.Name,
		Params:     params,
		Schedule:   rule.Schedule,
		Tags:       tags,
		Actions:    actions,
		AlertDelay: rule.AlertDelay,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling Kibana rule payload: %v", err)
	}

	_, err = c.execKibanaRequest(
		http.MethodPut,
		fmt.Sprintf("/api/alerting/rule/%s", url.PathEscape(ruleID)),
		bytes.NewReader(data),
	)
	return err
}

func (c *Client) updateKibanaRuleWithRetry(ruleID string, update func(*KibanaRuleDetails) error) error {
	for attempt := 0; attempt < 2; attempt++ {
		rule, err := c.GetKibanaRule(ruleID)
		if err != nil {
			var kibanaErr *KibanaAPIError
			if errors.As(err, &kibanaErr) && kibanaErr.StatusCode == http.StatusNotFound {
				return nil
			}
			return err
		}

		err = update(rule)
		if err == nil {
			return nil
		}

		var kibanaErr *KibanaAPIError
		if !(errors.As(err, &kibanaErr) && kibanaErr.StatusCode == http.StatusConflict && attempt == 0) {
			return err
		}
	}

	return nil
}

func (c *Client) GetKibanaRuleDefaultActionGroupID(ruleTypeID string) (string, error) {
	body, err := c.execKibanaRequest(http.MethodGet, "/api/alerting/rule_types", nil)
	if err != nil {
		return "", err
	}

	var ruleTypes []KibanaRuleType
	if err := json.Unmarshal(body, &ruleTypes); err != nil {
		return "", fmt.Errorf("error parsing Kibana rule types response: %v", err)
	}

	for _, ruleType := range ruleTypes {
		if ruleType.ID == ruleTypeID {
			if ruleType.DefaultActionGroupID == "" {
				return "default", nil
			}
			return ruleType.DefaultActionGroupID, nil
		}
	}

	return "default", nil
}

func superPlaneKibanaRuleAction(connectorID, actionGroupID string) KibanaRuleAction {
	if actionGroupID == "" {
		actionGroupID = "default"
	}

	return KibanaRuleAction{
		ID:    connectorID,
		Group: actionGroupID,
		Params: map[string]any{
			"body": kibanaAlertWebhookActionBody,
		},
		Frequency: &KibanaRuleActionFrequency{
			NotifyWhen: "onActionGroupChange",
			Summary:    false,
			Throttle:   nil,
		},
	}
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
	ID                string `json:"id"`
	Name              string `json:"name"`
	ConnectorTypeID   string `json:"connector_type_id"`
	ReferencedByCount int    `json:"referenced_by_count"`
	Config            struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
	} `json:"config"`
}

func (c *Client) ListKibanaConnectors() ([]KibanaConnectorResponse, error) {
	responseBody, err := c.execKibanaRequest(http.MethodGet, "/api/actions/connectors", nil)
	if err != nil {
		return nil, err
	}

	var connectors []KibanaConnectorResponse
	if err := json.Unmarshal(responseBody, &connectors); err != nil {
		return nil, fmt.Errorf("error parsing Kibana connectors response: %v", err)
	}

	return connectors, nil
}

func (c *Client) FindKibanaWebhookConnector(webhookURL string) (*KibanaConnectorResponse, error) {
	connectors, err := c.ListKibanaConnectors()
	if err != nil {
		return nil, err
	}

	for _, connector := range connectors {
		if connector.ConnectorTypeID != ".webhook" {
			continue
		}
		if connector.Config.URL == webhookURL {
			return &connector, nil
		}
	}

	return nil, nil
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

// GetDocumentResponse is returned by GET /{index}/_doc/{id}.
type GetDocumentResponse struct {
	ID      string         `json:"_id"`
	Index   string         `json:"_index"`
	Version int            `json:"_version"`
	Found   bool           `json:"found"`
	Source  map[string]any `json:"_source"`
}

// GetDocument retrieves a document by index and document ID.
func (c *Client) GetDocument(index, documentID string) (*GetDocumentResponse, error) {
	fullURL := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, url.PathEscape(index), url.PathEscape(documentID))
	responseBody, err := c.execRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	var resp GetDocumentResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing get document response: %v", err)
	}

	return &resp, nil
}

// UpdateDocument applies a partial update to an existing document.
// Uses POST /{index}/_update/{id} with body {"doc": fields}.
// Reuses IndexDocumentResponse since the response shape is identical.
func (c *Client) UpdateDocument(index, documentID string, fields map[string]any) (*IndexDocumentResponse, error) {
	payload := map[string]any{"doc": fields}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling update payload: %v", err)
	}

	fullURL := fmt.Sprintf("%s/%s/_update/%s", c.baseURL, url.PathEscape(index), url.PathEscape(documentID))
	responseBody, err := c.execRequest(http.MethodPost, fullURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var resp IndexDocumentResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing update document response: %v", err)
	}

	return &resp, nil
}

// SearchHit represents a single document result from an Elasticsearch search.
type SearchHit struct {
	ID     string         `json:"_id"`
	Index  string         `json:"_index"`
	Source map[string]any `json:"_source"`
}

// ListDocuments returns up to 100 documents from an index for use in resource pickers.
func (c *Client) ListDocuments(index string) ([]SearchHit, error) {
	query := map[string]any{
		"query":   map[string]any{"match_all": map[string]any{}},
		"_source": false,
		"size":    100,
	}

	data, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("error marshaling list documents query: %v", err)
	}

	fullURL := fmt.Sprintf("%s/%s/_search", c.baseURL, url.PathEscape(index))
	responseBody, err := c.execRequest(http.MethodPost, fullURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var result struct {
		Hits struct {
			Hits []SearchHit `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing list documents response: %v", err)
	}

	return result.Hits.Hits, nil
}

// TimestampValue extracts the @timestamp value from the source as a string.
// Returns "" if the field is absent or not a string.
func (h *SearchHit) TimestampValue() string {
	if h.Source == nil {
		return ""
	}
	if v, ok := h.Source[onDocumentIndexedTimeField]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SearchDocumentsAfter queries an index for documents where @timestamp is
// strictly greater than afterTimestamp, sorted ascending.
func (c *Client) SearchDocumentsAfter(index, afterTimestamp string, size int) ([]SearchHit, error) {
	query := map[string]any{
		"query": map[string]any{
			"range": map[string]any{
				onDocumentIndexedTimeField: map[string]any{
					"gt": afterTimestamp,
				},
			},
		},
		"sort": []any{
			map[string]any{onDocumentIndexedTimeField: map[string]any{"order": "asc"}},
		},
		"size": size,
	}

	data, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("error marshaling search query: %v", err)
	}

	fullURL := fmt.Sprintf("%s/%s/_search", c.baseURL, url.PathEscape(index))
	responseBody, err := c.execRequest(http.MethodPost, fullURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var result struct {
		Hits struct {
			Hits []SearchHit `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing search response: %v", err)
	}

	return result.Hits.Hits, nil
}

// KibanaRuleResponse is the relevant subset of the Kibana alerting rule API response.
type KibanaRuleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateKibanaQueryRule creates a Kibana Elasticsearch query rule that fires
// connectorID whenever new documents appear in index within a 1-minute window.
func (c *Client) CreateKibanaQueryRule(index, connectorID, routeKey string) (*KibanaRuleResponse, error) {
	actionBody, err := json.Marshal(map[string]any{
		"eventType": "document_indexed",
		"index":     index,
		"routeKey":  routeKey,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling action body: %v", err)
	}

	esQuery, err := json.Marshal(map[string]any{"query": map[string]any{"match_all": map[string]any{}}})
	if err != nil {
		return nil, fmt.Errorf("error marshaling esQuery: %v", err)
	}

	payload := map[string]any{
		"name":         "SuperPlane \u2022 " + index,
		"rule_type_id": ".es-query",
		"consumer":     "alerts",
		"schedule":     map[string]any{"interval": "1m"},
		"params": map[string]any{
			"index":                      []string{index},
			"timeField":                  onDocumentIndexedTimeField,
			"esQuery":                    string(esQuery),
			"size":                       100,
			"threshold":                  []int{0},
			"thresholdComparator":        ">",
			"timeWindowSize":             1,
			"timeWindowUnit":             "m",
			"excludeHitsFromPreviousRun": true,
		},
		"actions": []any{
			map[string]any{
				"id":    connectorID,
				"group": "query matched",
				"params": map[string]any{
					"body": string(actionBody),
				},
				"frequency": map[string]any{
					"notify_when": "onActiveAlert",
					"throttle":    nil,
					"summary":     false,
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling rule payload: %v", err)
	}

	responseBody, err := c.execKibanaRequest(http.MethodPost, "/api/alerting/rule", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var resp KibanaRuleResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing rule response: %v", err)
	}

	return &resp, nil
}

// DeleteKibanaRule removes a Kibana alerting rule by ID.
// A 404 response is treated as success: the rule is already gone.
func (c *Client) DeleteKibanaRule(ruleID string) error {
	_, err := c.execKibanaRequest(
		http.MethodDelete,
		fmt.Sprintf("/api/alerting/rule/%s", url.PathEscape(ruleID)),
		nil,
	)
	var kibanaErr *KibanaAPIError
	if errors.As(err, &kibanaErr) && kibanaErr.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}
