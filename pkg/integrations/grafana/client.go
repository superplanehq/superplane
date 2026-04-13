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
	"time"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	maxResponseSize = 2 * 1024 * 1024 // 2MB
)

var createAnnotationRetryDelays = []time.Duration{
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
}

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

// Notification policies are read and written as map[string]json.RawMessage so Grafana
// fields we do not model (mute_time_intervals, matchers, nested route fields, etc.)
// round-trip unchanged at the root and within each route object.

func parseNotificationPolicyRoot(body []byte) (map[string]json.RawMessage, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("error parsing notification policies: %v", err)
	}
	if root == nil {
		root = map[string]json.RawMessage{}
	}
	return root, nil
}

func marshalNotificationPolicyRoot(root map[string]json.RawMessage) ([]byte, error) {
	data, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("error marshaling notification policies: %v", err)
	}
	return data, nil
}

func getChildRoutes(root map[string]json.RawMessage) ([]json.RawMessage, error) {
	raw, ok := root["routes"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var routes []json.RawMessage
	if err := json.Unmarshal(raw, &routes); err != nil {
		return nil, fmt.Errorf("error parsing routes array: %v", err)
	}
	return routes, nil
}

func setChildRoutes(root map[string]json.RawMessage, routes []json.RawMessage) error {
	encoded, err := json.Marshal(routes)
	if err != nil {
		return err
	}
	root["routes"] = encoded
	return nil
}

type routeReceiverField struct {
	Receiver string `json:"receiver"`
}

func routeReceiverName(route json.RawMessage) (string, error) {
	var r routeReceiverField
	if err := json.Unmarshal(route, &r); err != nil {
		return "", err
	}
	return strings.TrimSpace(r.Receiver), nil
}

func removeRoutesForReceiverRaw(routes []json.RawMessage, receiver string) ([]json.RawMessage, error) {
	if len(routes) == 0 {
		return nil, nil
	}
	out := make([]json.RawMessage, 0, len(routes))
	for _, route := range routes {
		name, err := routeReceiverName(route)
		if err != nil {
			return nil, err
		}
		if name != receiver {
			out = append(out, route)
		}
	}
	return out, nil
}

type superplaneNotificationRoute struct {
	Receiver       string     `json:"receiver"`
	Continue       bool       `json:"continue,omitempty"`
	ObjectMatchers [][]string `json:"object_matchers,omitempty"`
}

func (c *Client) getNotificationPolicies() (map[string]json.RawMessage, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/api/v1/provisioning/policies", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error getting notification policies: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana get notification policies", status, responseBody)
	}
	return parseNotificationPolicyRoot(responseBody)
}

func (c *Client) putNotificationPolicies(root map[string]json.RawMessage) error {
	data, err := marshalNotificationPolicyRoot(root)
	if err != nil {
		return err
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

	routes, err := getChildRoutes(root)
	if err != nil {
		return err
	}

	filtered, err := removeRoutesForReceiverRaw(routes, contactPointName)
	if err != nil {
		return err
	}

	route := superplaneNotificationRoute{
		Receiver: contactPointName,
		Continue: true,
	}
	if len(alertNamePredicates) > 0 {
		route.ObjectMatchers = buildAlertNameMatchers(alertNamePredicates)
	}

	routeBytes, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("error marshaling notification route: %v", err)
	}

	// Prepend so our route takes priority over catch-alls.
	newRoutes := append([]json.RawMessage{routeBytes}, filtered...)
	if err := setChildRoutes(root, newRoutes); err != nil {
		return err
	}
	return c.putNotificationPolicies(root)
}

// combinedPositiveAlertNameRegex builds the =~ pattern Grafana uses for object_matchers on
// alertname: full-string match with positive predicates OR'd inside one alternation.
// Must stay aligned with alertLabelNameMatchesPredicates in on_alert_firing.go.
func combinedPositiveAlertNameRegex(predicates []configuration.Predicate) (string, bool) {
	var parts []string
	for _, p := range predicates {
		switch p.Type {
		case configuration.PredicateTypeEquals:
			parts = append(parts, regexp.QuoteMeta(p.Value))
		case configuration.PredicateTypeMatches:
			parts = append(parts, p.Value)
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	// Grafana =~ applies the regex as a full-string match (same as anchoring the alternation).
	return "^(?:" + strings.Join(parts, "|") + ")$", true
}

// buildAlertNameMatchers converts predicates into Grafana object_matchers entries.
// Positive predicates (equals, matches) are combined into one =~ matcher (OR inside the regex);
// negative predicates (notEquals) become separate != matchers. Grafana ANDs matchers together.
func buildAlertNameMatchers(predicates []configuration.Predicate) [][]string {
	var matchers [][]string

	for _, p := range predicates {
		if p.Type == configuration.PredicateTypeNotEquals {
			matchers = append(matchers, []string{"alertname", "!=", p.Value})
		}
	}

	combined, ok := combinedPositiveAlertNameRegex(predicates)
	if ok {
		matchers = append([][]string{{"alertname", "=~", combined}}, matchers...)
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

	routes, err := getChildRoutes(root)
	if err != nil {
		return err
	}

	filtered, err := removeRoutesForReceiverRaw(routes, contactPointName)
	if err != nil {
		return err
	}
	if len(filtered) == len(routes) {
		return nil // nothing to remove
	}

	if err := setChildRoutes(root, filtered); err != nil {
		return err
	}
	return c.putNotificationPolicies(root)
}

type Annotation struct {
	ID           int64    `json:"id"`
	DashboardUID string   `json:"dashboardUID"`
	PanelID      int64    `json:"panelId"`
	Time         int64    `json:"time"`
	TimeEnd      int64    `json:"timeEnd"`
	Text         string   `json:"text"`
	Tags         []string `json:"tags"`
	Type         string   `json:"type"`
}

func (c *Client) CreateAnnotation(text string, tags []string, dashboardUID string, panelID *int64, timeMS, timeEndMS int64) (int64, error) {
	payload := map[string]any{
		"text": text,
	}
	if len(tags) > 0 {
		payload["tags"] = tags
	}
	if strings.TrimSpace(dashboardUID) != "" {
		payload["dashboardUID"] = strings.TrimSpace(dashboardUID)
	}
	if panelID != nil {
		payload["panelId"] = *panelID
	}
	if timeMS > 0 {
		payload["time"] = timeMS
	}
	if timeEndMS > 0 {
		payload["timeEnd"] = timeEndMS
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("error marshaling annotation payload: %v", err)
	}

	for attempt := 0; ; attempt++ {
		responseBody, status, err := c.execRequest(
			http.MethodPost,
			"/api/annotations",
			bytes.NewReader(body),
			"application/json",
		)
		if err != nil {
			return 0, fmt.Errorf("error creating annotation: %v", err)
		}
		if status == http.StatusTooManyRequests && attempt < len(createAnnotationRetryDelays) {
			time.Sleep(createAnnotationRetryDelays[attempt])
			continue
		}
		if status < 200 || status >= 300 {
			return 0, newAPIStatusError("grafana annotation create", status, responseBody)
		}

		var result struct {
			ID      int64  `json:"id"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return 0, fmt.Errorf("error parsing create annotation response: %v", err)
		}
		return result.ID, nil
	}
}

func (c *Client) ListAnnotations(tags []string, dashboardUID string, panelID *int64, from, to, limit int64) ([]Annotation, error) {
	q := url.Values{}
	for _, tag := range tags {
		if strings.TrimSpace(tag) != "" {
			q.Add("tags", strings.TrimSpace(tag))
		}
	}
	if strings.TrimSpace(dashboardUID) != "" {
		q.Set("dashboardUID", strings.TrimSpace(dashboardUID))
	}
	if panelID != nil && *panelID > 0 {
		q.Set("panelId", fmt.Sprintf("%d", *panelID))
	}
	if from > 0 {
		q.Set("from", fmt.Sprintf("%d", from))
	}
	if to > 0 {
		q.Set("to", fmt.Sprintf("%d", to))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}

	path := "/api/annotations"
	if encoded := q.Encode(); encoded != "" {
		path = path + "?" + encoded
	}

	responseBody, status, err := c.execRequest(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, fmt.Errorf("error listing annotations: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana annotation list", status, responseBody)
	}

	var annotations []Annotation
	if err := json.Unmarshal(responseBody, &annotations); err != nil {
		return nil, fmt.Errorf("error parsing annotations response: %v", err)
	}
	return annotations, nil
}

func (c *Client) GetAnnotation(id int64) (Annotation, error) {
	responseBody, status, err := c.execRequest(
		http.MethodGet,
		fmt.Sprintf("/api/annotations/%d", id),
		nil,
		"",
	)
	if err != nil {
		return Annotation{}, fmt.Errorf("error getting annotation: %v", err)
	}
	if status < 200 || status >= 300 {
		return Annotation{}, newAPIStatusError("grafana annotation get", status, responseBody)
	}

	var annotation Annotation
	if err := json.Unmarshal(responseBody, &annotation); err != nil {
		return Annotation{}, fmt.Errorf("error parsing annotation response: %v", err)
	}
	return annotation, nil
}

func (c *Client) DeleteAnnotation(id int64) error {
	responseBody, status, err := c.execRequest(
		http.MethodDelete,
		fmt.Sprintf("/api/annotations/%d", id),
		nil, "",
	)
	if err != nil {
		return fmt.Errorf("error deleting annotation: %v", err)
	}
	if status == http.StatusNotFound || (status >= 200 && status < 300) {
		return nil
	}
	return newAPIStatusError("grafana annotation delete", status, responseBody)
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

// DashboardSearchHit matches Grafana GET /api/search entries for type=dash-db.
type DashboardSearchHit struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
}

type DashboardPanel struct {
	ID    int64
	Title string
}

// SearchDashboards lists dashboards via the folder/dashboard search API (dashboard UID + title).
func (c *Client) SearchDashboards() ([]DashboardSearchHit, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/api/search?type=dash-db&limit=5000", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error searching dashboards: %v", err)
	}

	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana dashboard search", status, responseBody)
	}

	var hits []DashboardSearchHit
	if err := json.Unmarshal(responseBody, &hits); err != nil {
		return nil, fmt.Errorf("error parsing dashboard search response: %v", err)
	}

	return hits, nil
}

// GetDashboardTitle loads a dashboard by UID and returns its title (GET /api/dashboards/uid/:uid).
func (c *Client) GetDashboardTitle(uid string) (string, error) {
	responseBody, status, err := c.execRequest(
		http.MethodGet,
		fmt.Sprintf("/api/dashboards/uid/%s", url.PathEscape(strings.TrimSpace(uid))),
		nil,
		"",
	)
	if err != nil {
		return "", fmt.Errorf("error getting dashboard: %v", err)
	}
	if status < 200 || status >= 300 {
		return "", newAPIStatusError("grafana dashboard get", status, responseBody)
	}

	var response struct {
		Dashboard struct {
			Title string `json:"title"`
		} `json:"dashboard"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("error parsing dashboard response: %v", err)
	}
	return strings.TrimSpace(response.Dashboard.Title), nil
}

func (c *Client) ListDashboardPanels(uid string) ([]DashboardPanel, error) {
	responseBody, status, err := c.execRequest(
		http.MethodGet,
		fmt.Sprintf("/api/dashboards/uid/%s", url.PathEscape(strings.TrimSpace(uid))),
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("error getting dashboard: %v", err)
	}

	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana dashboard get", status, responseBody)
	}

	var response struct {
		Dashboard map[string]any `json:"dashboard"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing dashboard response: %v", err)
	}

	return extractDashboardPanels(response.Dashboard), nil
}

func extractDashboardPanels(dashboard map[string]any) []DashboardPanel {
	var panels []DashboardPanel
	if dashboard == nil {
		return panels
	}

	rootPanels, _ := dashboard["panels"].([]any)
	collectDashboardPanels(rootPanels, &panels)
	return panels
}

func collectDashboardPanels(values []any, destination *[]DashboardPanel) {
	for _, value := range values {
		panel, ok := value.(map[string]any)
		if !ok {
			continue
		}

		if nestedPanels, ok := panel["panels"].([]any); ok {
			collectDashboardPanels(nestedPanels, destination)
		}

		panelType, _ := panel["type"].(string)
		if strings.EqualFold(strings.TrimSpace(panelType), "row") {
			continue
		}

		idValue, ok := panel["id"]
		if !ok {
			continue
		}

		idFloat, ok := idValue.(float64)
		if !ok {
			continue
		}

		id := int64(idFloat)
		if id <= 0 {
			continue
		}

		title, _ := panel["title"].(string)
		*destination = append(*destination, DashboardPanel{
			ID:    id,
			Title: title,
		})
	}
}
