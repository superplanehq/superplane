package grafana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	maxResponseSize = 2 * 1024 * 1024 // 2MB
)

type Client struct {
	BaseURL  string
	APIToken string
	http     core.HTTPContext
}

type contactPoint struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type DataSource struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type apiStatusError struct {
	Operation    string
	StatusCode   int
	ResponseBody string
}

func (e *apiStatusError) Error() string {
	return fmt.Sprintf("%s failed with status %d: %s", e.Operation, e.StatusCode, e.ResponseBody)
}

func newAPIStatusError(operation string, status int, responseBody []byte) error {
	return &apiStatusError{
		Operation:    operation,
		StatusCode:   status,
		ResponseBody: string(responseBody),
	}
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext, requireToken bool) (*Client, error) {
	baseURL, err := readBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	apiToken, err := readAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	if requireToken && apiToken == "" {
		return nil, fmt.Errorf("apiToken is required")
	}

	return &Client{
		BaseURL:  baseURL,
		APIToken: apiToken,
		http:     httpCtx,
	}, nil
}

func readBaseURL(ctx core.IntegrationContext) (string, error) {
	baseURLConfig, err := ctx.GetConfig("baseURL")
	if err != nil {
		return "", fmt.Errorf("error reading baseURL: %v", err)
	}

	if baseURLConfig == nil {
		return "", fmt.Errorf("baseURL is required")
	}

	baseURLRaw := strings.TrimSpace(string(baseURLConfig))
	if baseURLRaw == "" {
		return "", fmt.Errorf("baseURL is required")
	}

	parsed, err := url.Parse(baseURLRaw)
	if err != nil {
		return "", fmt.Errorf("invalid baseURL: %v", err)
	}

	// url.Parse accepts relative URLs (e.g. "grafana.local"), which will fail later in http.NewRequest.
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid baseURL: must include scheme and host (e.g. https://grafana.example.com)")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid baseURL: unsupported scheme %q (expected http or https)", parsed.Scheme)
	}

	return strings.TrimSuffix(baseURLRaw, "/"), nil
}

func readAPIToken(ctx core.IntegrationContext) (string, error) {
	type optionalConfigReader interface {
		GetOptionalConfig(name string) ([]byte, error)
	}

	var (
		apiTokenConfig []byte
		err            error
	)

	if optionalCtx, ok := ctx.(optionalConfigReader); ok {
		apiTokenConfig, err = optionalCtx.GetOptionalConfig("apiToken")
	} else {
		apiTokenConfig, err = ctx.GetConfig("apiToken")
		if err != nil && strings.Contains(err.Error(), "config apiToken not found") {
			return "", nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("error reading apiToken: %v", err)
	}

	if apiTokenConfig == nil {
		return "", nil
	}

	return strings.TrimSpace(string(apiTokenConfig)), nil
}

func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.BaseURL, "/"), strings.TrimPrefix(path, "/"))
}

func (c *Client) execRequest(method, path string, body io.Reader, contentType string) ([]byte, int, error) {
	return c.execRequestWithHeaders(method, path, body, contentType, nil)
}

func (c *Client) execRequestWithHeaders(
	method, path string,
	body io.Reader,
	contentType string,
	headers map[string]string,
) ([]byte, int, error) {
	req, err := http.NewRequest(method, c.buildURL(path), body)
	if err != nil {
		return nil, 0, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.APIToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	// Read one byte beyond the max to detect overflow without rejecting an exact-limit response.
	limitedReader := io.LimitReader(res.Body, int64(maxResponseSize)+1)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("error reading body: %v", err)
	}

	if len(responseBody) > maxResponseSize {
		return nil, res.StatusCode, fmt.Errorf("response too large: exceeds maximum size of %d bytes", maxResponseSize)
	}

	return responseBody, res.StatusCode, nil
}

func (c *Client) listContactPoints() ([]contactPoint, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/api/v1/provisioning/contact-points", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error listing contact points: %v", err)
	}

	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana contact point list", status, responseBody)
	}

	var direct []contactPoint
	if err := json.Unmarshal(responseBody, &direct); err == nil {
		return direct, nil
	}

	wrapped := struct {
		Items json.RawMessage `json:"items"`
	}{}
	if err := json.Unmarshal(responseBody, &wrapped); err == nil {
		if wrapped.Items == nil || bytes.Equal(bytes.TrimSpace(wrapped.Items), []byte("null")) {
			return nil, fmt.Errorf("error parsing contact points response")
		}

		var items []contactPoint
		if err := json.Unmarshal(wrapped.Items, &items); err != nil {
			return nil, fmt.Errorf("error parsing contact points response")
		}

		return items, nil
	}

	return nil, fmt.Errorf("error parsing contact points response")
}

func (c *Client) UpsertWebhookContactPoint(name, webhookURL, bearerToken string) (string, error) {
	points, err := c.listContactPoints()
	if err != nil {
		return "", err
	}

	existingUID := ""
	for _, point := range points {
		if strings.TrimSpace(point.Name) == name {
			existingUID = strings.TrimSpace(point.UID)
			break
		}
	}

	payload := map[string]any{
		"name":                  name,
		"type":                  "webhook",
		"disableResolveMessage": false,
		"settings": map[string]any{
			"url":        webhookURL,
			"httpMethod": "POST",
		},
	}

	if bearerToken != "" {
		settings := payload["settings"].(map[string]any)
		settings["authorization_scheme"] = "Bearer"
		settings["authorization_credentials"] = bearerToken
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling contact point payload: %v", err)
	}

	if existingUID != "" {
		responseBody, status, err := c.execRequestWithHeaders(
			http.MethodPut,
			fmt.Sprintf("/api/v1/provisioning/contact-points/%s", existingUID),
			bytes.NewReader(body),
			"application/json",
			map[string]string{
				"X-Disable-Provenance": "true",
			},
		)
		if err != nil {
			return "", fmt.Errorf("error updating contact point: %v", err)
		}
		if status < 200 || status >= 300 {
			return "", newAPIStatusError("grafana contact point update", status, responseBody)
		}
		return existingUID, nil
	}

	responseBody, status, err := c.execRequestWithHeaders(
		http.MethodPost,
		"/api/v1/provisioning/contact-points",
		bytes.NewReader(body),
		"application/json",
		map[string]string{
			"X-Disable-Provenance": "true",
		},
	)
	if err != nil {
		return "", fmt.Errorf("error creating contact point: %v", err)
	}
	if status < 200 || status >= 300 {
		return "", newAPIStatusError("grafana contact point create", status, responseBody)
	}

	created := contactPoint{}
	if err := json.Unmarshal(responseBody, &created); err == nil && strings.TrimSpace(created.UID) != "" {
		return strings.TrimSpace(created.UID), nil
	}

	refreshedPoints, err := c.listContactPoints()
	if err != nil {
		return "", err
	}

	for _, point := range refreshedPoints {
		if strings.TrimSpace(point.Name) == name && strings.TrimSpace(point.UID) != "" {
			return strings.TrimSpace(point.UID), nil
		}
	}

	return "", fmt.Errorf("contact point created but uid was not returned")
}

func (c *Client) DeleteContactPoint(uid string) error {
	if strings.TrimSpace(uid) == "" {
		return nil
	}

	responseBody, status, err := c.execRequest(http.MethodDelete, fmt.Sprintf("/api/v1/provisioning/contact-points/%s", uid), nil, "")
	if err != nil {
		return fmt.Errorf("error deleting contact point: %v", err)
	}

	if status == http.StatusNotFound {
		return nil
	}

	if status < 200 || status >= 300 {
		return newAPIStatusError("grafana contact point delete", status, responseBody)
	}

	return nil
}

// NotificationPolicyRoute represents one node in Grafana's notification policy tree.
// We only model the fields we read/write; unknown fields are preserved via RawFields.
type NotificationPolicyRoute struct {
	Receiver       string                    `json:"receiver"`
	GroupBy        []string                  `json:"group_by,omitempty"`
	ObjectMatchers [][]string                `json:"object_matchers,omitempty"`
	Continue       bool                      `json:"continue,omitempty"`
	GroupWait      string                    `json:"group_wait,omitempty"`
	GroupInterval  string                    `json:"group_interval,omitempty"`
	RepeatInterval string                    `json:"repeat_interval,omitempty"`
	Routes         []NotificationPolicyRoute `json:"routes,omitempty"`
}

func (c *Client) getNotificationPolicies() (*NotificationPolicyRoute, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/api/v1/provisioning/policies", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error getting notification policies: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana get notification policies", status, responseBody)
	}
	var root NotificationPolicyRoute
	if err := json.Unmarshal(responseBody, &root); err != nil {
		return nil, fmt.Errorf("error parsing notification policies response: %v", err)
	}
	return &root, nil
}

func (c *Client) putNotificationPolicies(root *NotificationPolicyRoute) error {
	data, err := json.Marshal(root)
	if err != nil {
		return fmt.Errorf("error marshaling notification policies: %v", err)
	}
	responseBody, status, err := c.execRequestWithHeaders(
		http.MethodPut, "/api/v1/provisioning/policies",
		bytes.NewReader(data), "application/json",
		map[string]string{"X-Disable-Provenance": "true"},
	)
	if err != nil {
		return fmt.Errorf("error updating notification policies: %v", err)
	}
	if status < 200 || status >= 300 {
		return newAPIStatusError("grafana put notification policies", status, responseBody)
	}
	return nil
}

// UpsertNotificationPolicyRoute ensures a child route for contactPointName exists at the
// root of the policy tree. If alertNamePredicates is non-empty, object_matchers are built
// from the predicates: positive predicates (equals, matches) are combined into a single
// =~ regex OR pattern; negative predicates (notEquals) become individual != matchers.
// The route has continue=true so other routes still fire.
func (c *Client) UpsertNotificationPolicyRoute(contactPointName string, alertNamePredicates []configuration.Predicate) error {
	root, err := c.getNotificationPolicies()
	if err != nil {
		return err
	}

	// Remove any existing route for this contact point.
	root.Routes = removeRoutesForReceiver(root.Routes, contactPointName)

	route := NotificationPolicyRoute{
		Receiver: contactPointName,
		Continue: true,
	}

	if len(alertNamePredicates) > 0 {
		route.ObjectMatchers = buildAlertNameMatchers(alertNamePredicates)
	}

	// Prepend so our route takes priority over catch-alls.
	root.Routes = append([]NotificationPolicyRoute{route}, root.Routes...)
	return c.putNotificationPolicies(root)
}

// buildAlertNameMatchers converts predicates into Grafana object_matchers entries.
// Positive predicates (equals, matches) are combined into a single =~ regex alternative.
// Negative predicates (notEquals) become individual != matchers.
func buildAlertNameMatchers(predicates []configuration.Predicate) [][]string {
	var positivePatterns []string
	var matchers [][]string

	for _, p := range predicates {
		switch p.Type {
		case configuration.PredicateTypeEquals:
			positivePatterns = append(positivePatterns, regexp.QuoteMeta(p.Value))
		case configuration.PredicateTypeMatches:
			positivePatterns = append(positivePatterns, p.Value)
		case configuration.PredicateTypeNotEquals:
			matchers = append(matchers, []string{"alertname", "!=", p.Value})
		}
	}

	if len(positivePatterns) > 0 {
		matchers = append([][]string{{"alertname", "=~", strings.Join(positivePatterns, "|")}}, matchers...)
	}

	return matchers
}

// RemoveNotificationPolicyRoute removes any child route for contactPointName from the
// root of the policy tree. No-op if no such route exists.
func (c *Client) RemoveNotificationPolicyRoute(contactPointName string) error {
	root, err := c.getNotificationPolicies()
	if err != nil {
		return err
	}

	filtered := removeRoutesForReceiver(root.Routes, contactPointName)
	if len(filtered) == len(root.Routes) {
		return nil // nothing to remove
	}

	root.Routes = filtered
	return c.putNotificationPolicies(root)
}

func removeRoutesForReceiver(routes []NotificationPolicyRoute, receiver string) []NotificationPolicyRoute {
	result := make([]NotificationPolicyRoute, 0, len(routes))
	for _, r := range routes {
		if strings.TrimSpace(r.Receiver) != receiver {
			result = append(result, r)
		}
	}
	return result
}

func (c *Client) ListDataSources() ([]DataSource, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/api/datasources", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error listing data sources: %v", err)
	}

	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana data source list", status, responseBody)
	}

	var sources []DataSource
	if err := json.Unmarshal(responseBody, &sources); err != nil {
		return nil, fmt.Errorf("error parsing data sources response: %v", err)
	}

	return sources, nil
}
