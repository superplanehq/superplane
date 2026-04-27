package grafana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	syntheticMonitoringDataSourceType = "synthetic-monitoring-datasource"
	syntheticsMetricsLookback         = "24h"
	syntheticsPluginPath              = "/a/grafana-synthetic-monitoring-app/checks"
)

type SyntheticsClient struct {
	DataSourceUID        string
	MetricsDataSourceUID string
	GrafanaClient        *Client
}

type SyntheticCheckLabel struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SyntheticCheckBasicAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type SyntheticCheckHeaderMatch struct {
	Header       string `json:"header,omitempty"`
	Regexp       string `json:"regexp,omitempty"`
	AllowMissing bool   `json:"allowMissing,omitempty"`
}

type SyntheticCheckHTTPSettings struct {
	Method                       string                      `json:"method"`
	Headers                      []string                    `json:"headers,omitempty"`
	Body                         string                      `json:"body,omitempty"`
	IPVersion                    string                      `json:"ipVersion,omitempty"`
	TLSConfig                    *SyntheticCheckTLSConfig    `json:"tlsConfig,omitempty"`
	NoFollowRedirects            bool                        `json:"noFollowRedirects,omitempty"`
	Compression                  string                      `json:"compression,omitempty"`
	BearerToken                  string                      `json:"bearerToken,omitempty"`
	BasicAuth                    *SyntheticCheckBasicAuth    `json:"basicAuth,omitempty"`
	FailIfSSL                    bool                        `json:"failIfSSL,omitempty"`
	FailIfNotSSL                 bool                        `json:"failIfNotSSL,omitempty"`
	ValidStatusCodes             []int                       `json:"validStatusCodes,omitempty"`
	FailIfBodyMatchesRegexp      []string                    `json:"failIfBodyMatchesRegexp,omitempty"`
	FailIfBodyNotMatchesRegexp   []string                    `json:"failIfBodyNotMatchesRegexp,omitempty"`
	FailIfHeaderMatchesRegexp    []SyntheticCheckHeaderMatch `json:"failIfHeaderMatchesRegexp,omitempty"`
	FailIfHeaderNotMatchesRegexp []SyntheticCheckHeaderMatch `json:"failIfHeaderNotMatchesRegexp,omitempty"`
}

type SyntheticCheckSettings struct {
	HTTP *SyntheticCheckHTTPSettings `json:"http,omitempty"`
}

type SyntheticCheck struct {
	ID               int64                  `json:"id,omitempty"`
	TenantID         int64                  `json:"tenantId,omitempty"`
	Job              string                 `json:"job"`
	Target           string                 `json:"target"`
	Frequency        int64                  `json:"frequency"`
	Timeout          int64                  `json:"timeout"`
	Enabled          bool                   `json:"enabled"`
	AlertSensitivity string                 `json:"alertSensitivity,omitempty"`
	BasicMetricsOnly bool                   `json:"basicMetricsOnly"`
	Labels           []SyntheticCheckLabel  `json:"labels,omitempty"`
	Probes           []int64                `json:"probes,omitempty"`
	Created          float64                `json:"created,omitempty"`
	Modified         float64                `json:"modified,omitempty"`
	Settings         SyntheticCheckSettings `json:"settings"`
	Alerts           []SyntheticCheckAlert  `json:"alerts,omitempty"`
}

type SyntheticCheckTLSConfig struct {
	CACert             string `json:"caCert,omitempty"`
	ClientCert         string `json:"clientCert,omitempty"`
	ClientKey          string `json:"clientKey,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
	ServerName         string `json:"serverName,omitempty"`
}

type SyntheticCheckAlert struct {
	Name       string `json:"name"`
	Threshold  int64  `json:"threshold"`
	Period     string `json:"period,omitempty"`
	RunbookURL string `json:"runbookUrl,omitempty"`
}

type SyntheticProbe struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
	Public bool   `json:"public,omitempty"`
	Online bool   `json:"online,omitempty"`
}

type SyntheticCheckMetrics struct {
	// LastOutcome is the derived probe reachability label: "Up", "Partial", or "Down"
	// (see probeAvgToOutcome). It matches getHttpSyntheticCheck output channel routing.
	LastOutcome              *string  `json:"lastOutcome,omitempty"`
	UptimePercent24h         *float64 `json:"uptimePercent24h,omitempty"`
	ReachabilityPercent24h   *float64 `json:"reachabilityPercent24h,omitempty"`
	SuccessRuns24h           *float64 `json:"successRuns24h,omitempty"`
	FailureRuns24h           *float64 `json:"failureRuns24h,omitempty"`
	TotalRuns24h             *float64 `json:"totalRuns24h,omitempty"`
	AverageLatencySeconds24h *float64 `json:"averageLatencySeconds24h,omitempty"`
	SSLEarliestExpiryAt      *string  `json:"sslEarliestExpiryAt,omitempty"`
	SSLEarliestExpiryDays    *float64 `json:"sslEarliestExpiryDays,omitempty"`
	FrequencyMilliseconds    *int64   `json:"frequencyMilliseconds,omitempty"`
	LastExecutionAt          *string  `json:"lastExecutionAt,omitempty"`
}

func (c SyntheticCheck) IDString() string {
	if c.ID <= 0 {
		return ""
	}
	return strconv.FormatInt(c.ID, 10)
}

func (p SyntheticProbe) IDString() string {
	if p.ID <= 0 {
		return ""
	}
	return strconv.FormatInt(p.ID, 10)
}

func NewSyntheticsClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*SyntheticsClient, error) {
	grafanaClient, err := NewClient(httpCtx, ctx, true)
	if err != nil {
		return nil, err
	}

	dataSourceUID, metricsDataSourceUID, err := resolveSyntheticMonitoringDataSources(grafanaClient)
	if err != nil {
		return nil, err
	}

	return &SyntheticsClient{
		DataSourceUID:        dataSourceUID,
		MetricsDataSourceUID: metricsDataSourceUID,
		GrafanaClient:        grafanaClient,
	}, nil
}

func (c *SyntheticsClient) buildProxyPath(path string) string {
	return fmt.Sprintf("/api/datasources/proxy/uid/%s/%s", url.PathEscape(strings.TrimSpace(c.DataSourceUID)), strings.TrimPrefix(path, "/"))
}

func (c *SyntheticsClient) execRequest(method, path string, body io.Reader, contentType string) ([]byte, int, error) {
	if c.GrafanaClient == nil || strings.TrimSpace(c.DataSourceUID) == "" {
		return nil, 0, fmt.Errorf("grafana synthetic monitoring datasource is not configured")
	}

	return c.GrafanaClient.execRequest(method, c.buildProxyPath(path), body, contentType)
}

func (c *SyntheticsClient) ListChecks() ([]SyntheticCheck, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/sm/check/list?includeAlerts=true", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error listing synthetic checks: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana synthetic check list", status, responseBody)
	}

	var checks []SyntheticCheck
	if err := json.Unmarshal(responseBody, &checks); err != nil {
		return nil, fmt.Errorf("error parsing synthetic checks response: %v", err)
	}

	return checks, nil
}

func (c *SyntheticsClient) GetCheck(id string) (*SyntheticCheck, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("synthetic check id is required")
	}

	responseBody, status, err := c.execRequest(http.MethodGet, fmt.Sprintf("/sm/check/%s", url.PathEscape(id)), nil, "")
	if err != nil {
		return nil, err
	}

	if status >= 200 && status < 300 {
		var check SyntheticCheck
		if err := json.Unmarshal(responseBody, &check); err != nil {
			return nil, fmt.Errorf("error parsing synthetic check response: %v", err)
		}
		return &check, nil
	}

	// Some proxy setups only expose list; fall back so integrations keep working.
	if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
		return c.getCheckViaList(id)
	}

	return nil, newAPIStatusError("grafana synthetic check get", status, responseBody)
}

func (c *SyntheticsClient) getCheckViaList(id string) (*SyntheticCheck, error) {
	checks, err := c.ListChecks()
	if err != nil {
		return nil, err
	}

	for i := range checks {
		if checks[i].IDString() == id {
			return &checks[i], nil
		}
	}

	return nil, fmt.Errorf("synthetic check %q not found", id)
}

func (c *SyntheticsClient) CreateCheck(check SyntheticCheck) (*SyntheticCheck, error) {
	body, err := json.Marshal(check)
	if err != nil {
		return nil, fmt.Errorf("error marshaling synthetic check payload: %v", err)
	}

	responseBody, status, err := c.execRequest(http.MethodPost, "/sm/check/add", bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("error creating synthetic check: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana synthetic check create", status, responseBody)
	}

	var created SyntheticCheck
	if err := json.Unmarshal(responseBody, &created); err != nil {
		return nil, fmt.Errorf("error parsing synthetic check response: %v", err)
	}

	return &created, nil
}

func (c *SyntheticsClient) UpdateCheck(check SyntheticCheck) (*SyntheticCheck, error) {
	if check.ID <= 0 {
		return nil, fmt.Errorf("synthetic check id is required for update")
	}
	if check.TenantID <= 0 {
		return nil, fmt.Errorf("synthetic check tenantId is required for update")
	}

	body, err := json.Marshal(check)
	if err != nil {
		return nil, fmt.Errorf("error marshaling synthetic check payload: %v", err)
	}

	responseBody, status, err := c.execRequest(http.MethodPost, "/sm/check/update", bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("error updating synthetic check: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana synthetic check update", status, responseBody)
	}

	var updated SyntheticCheck
	if err := json.Unmarshal(responseBody, &updated); err != nil {
		return nil, fmt.Errorf("error parsing synthetic check response: %v", err)
	}

	return &updated, nil
}

func (c *SyntheticsClient) DeleteCheck(id string) (map[string]any, error) {
	responseBody, status, err := c.execRequest(http.MethodDelete, fmt.Sprintf("/sm/check/delete/%s", url.PathEscape(id)), nil, "")
	if err != nil {
		return nil, fmt.Errorf("error deleting synthetic check: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana synthetic check delete", status, responseBody)
	}

	var response map[string]any
	if len(responseBody) == 0 {
		return map[string]any{"deleted": true, "syntheticCheck": id}, nil
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing synthetic check delete response: %v", err)
	}

	return response, nil
}

func (c *SyntheticsClient) ListProbes() ([]SyntheticProbe, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, "/sm/probe/list", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error listing synthetic probes: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana synthetic probe list", status, responseBody)
	}

	var probes []SyntheticProbe
	if err := json.Unmarshal(responseBody, &probes); err != nil {
		return nil, fmt.Errorf("error parsing synthetic probes response: %v", err)
	}

	return probes, nil
}

func (c *SyntheticsClient) ListCheckAlerts(id string) ([]SyntheticCheckAlert, error) {
	responseBody, status, err := c.execRequest(http.MethodGet, fmt.Sprintf("/sm/check/%s/alerts", url.PathEscape(id)), nil, "")
	if err != nil {
		return nil, fmt.Errorf("error listing synthetic check alerts: %v", err)
	}
	if status < 200 || status >= 300 {
		return nil, newAPIStatusError("grafana synthetic check alerts list", status, responseBody)
	}

	var response struct {
		Alerts []SyntheticCheckAlert `json:"alerts"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing synthetic check alerts response: %v", err)
	}

	return response.Alerts, nil
}

func (c *SyntheticsClient) UpdateCheckAlerts(id string, alerts []SyntheticCheckAlert) error {
	if alerts == nil {
		alerts = []SyntheticCheckAlert{}
	}

	body, err := json.Marshal(map[string]any{"alerts": alerts})
	if err != nil {
		return fmt.Errorf("error marshaling synthetic check alerts payload: %v", err)
	}

	responseBody, status, err := c.execRequest(http.MethodPut, fmt.Sprintf("/sm/check/%s/alerts", url.PathEscape(id)), bytes.NewReader(body), "application/json")
	if err != nil {
		return fmt.Errorf("error updating synthetic check alerts: %v", err)
	}
	if status < 200 || status >= 300 {
		return newAPIStatusError("grafana synthetic check alerts update", status, responseBody)
	}

	return nil
}

func buildSyntheticCheckWebURL(integration core.IntegrationContext, checkID int64) string {
	if checkID <= 0 {
		return ""
	}

	baseURL, err := readBaseURL(integration)
	if err != nil {
		return ""
	}

	return strings.TrimRight(baseURL, "/") + syntheticsPluginPath + "/" + strconv.FormatInt(checkID, 10)
}

func fetchSyntheticCheckMetrics(ctx core.ExecutionContext, check *SyntheticCheck, metricsDataSourceUID string) *SyntheticCheckMetrics {
	metrics := &SyntheticCheckMetrics{}
	if ctx.HTTP == nil || ctx.Integration == nil || check == nil || strings.TrimSpace(check.Job) == "" || strings.TrimSpace(check.Target) == "" {
		return metrics
	}

	if check.Frequency > 0 {
		metrics.FrequencyMilliseconds = &check.Frequency
	}

	metricsDataSourceUID = strings.TrimSpace(metricsDataSourceUID)
	if metricsDataSourceUID == "" {
		return metrics
	}

	grafanaClient, err := NewClient(ctx.HTTP, ctx.Integration, false)
	if err != nil {
		return metrics
	}

	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	var rawSuccessRuns *float64
	var rawTotalRuns *float64

	successRuns, err := querySyntheticMetricValue(grafanaClient, metricsDataSourceUID, buildSuccessRunsQuery(check), from, now)
	if err == nil {
		rawSuccessRuns = successRuns
		metrics.SuccessRuns24h = normalizeSyntheticRunCount(successRuns)
	}

	totalRuns, err := querySyntheticMetricValue(grafanaClient, metricsDataSourceUID, buildTotalRunsQuery(check), from, now)
	if err == nil {
		rawTotalRuns = totalRuns
		metrics.TotalRuns24h = normalizeSyntheticRunCount(totalRuns)
	}

	if rawSuccessRuns != nil && rawTotalRuns != nil {
		failures := *rawTotalRuns - *rawSuccessRuns
		if failures < 0 {
			failures = 0
		}
		metrics.FailureRuns24h = normalizeSyntheticRunCount(&failures)

		if *rawTotalRuns > 0 {
			reachability := (*rawSuccessRuns / *rawTotalRuns) * 100
			metrics.ReachabilityPercent24h = &reachability
		}
	}

	avgLatency, err := querySyntheticMetricValue(grafanaClient, metricsDataSourceUID, buildAverageLatencyQuery(check), from, now)
	if err == nil {
		metrics.AverageLatencySeconds24h = avgLatency
	}

	uptime, err := querySyntheticMetricValue(grafanaClient, metricsDataSourceUID, buildUptimeQuery(check), from, now)
	if err == nil && uptime != nil {
		uptimePercent := *uptime * 100
		metrics.UptimePercent24h = &uptimePercent
	}

	// Use a 2h window matching the last_over_time lookback so that newly-created
	// checks with only a few minutes of probe data are not excluded by the step
	// grid Grafana applies to a 24h range query.
	probeAvg, err := querySyntheticMetricValue(grafanaClient, metricsDataSourceUID, buildCurrentOutcomeQuery(check), now.Add(-2*time.Hour), now)
	if err == nil && probeAvg != nil {
		outcome := probeAvgToOutcome(*probeAvg)
		metrics.LastOutcome = &outcome
	}

	lastExecution, err := querySyntheticMetricValue(grafanaClient, metricsDataSourceUID, buildLastExecutionQuery(check), from, now)
	if err == nil && lastExecution != nil && *lastExecution > 0 {
		executedAt := time.Unix(int64(*lastExecution), 0).UTC().Format(time.RFC3339)
		metrics.LastExecutionAt = &executedAt
	}

	sslExpiry, err := querySyntheticMetricValue(grafanaClient, metricsDataSourceUID, buildSSLEarliestExpiryQuery(check), from, now)
	if err == nil && sslExpiry != nil && *sslExpiry > 0 {
		expiryAt := time.Unix(int64(*sslExpiry), 0).UTC()
		expiryAtString := expiryAt.Format(time.RFC3339)
		expiryDays := expiryAt.Sub(now).Hours() / 24
		metrics.SSLEarliestExpiryAt = &expiryAtString
		metrics.SSLEarliestExpiryDays = &expiryDays
	}

	return metrics
}

func normalizeSyntheticRunCount(value *float64) *float64 {
	if value == nil {
		return nil
	}

	rounded := math.Round(*value)
	if rounded < 0 {
		rounded = 0
	}

	return &rounded
}

// resolveSyntheticMonitoringDataSources returns the Synthetic Monitoring plugin
// datasource UID (for /sm/* proxy calls) and the nested Prometheus metrics UID
// from jsonData.metrics (for /api/ds/query), using a single ListDataSources call.
func resolveSyntheticMonitoringDataSources(client *Client) (pluginUID string, metricsUID string, err error) {
	dataSources, err := client.ListDataSources()
	if err != nil {
		return "", "", err
	}

	for _, dataSource := range dataSources {
		if strings.TrimSpace(dataSource.Type) != syntheticMonitoringDataSourceType {
			continue
		}
		if pluginUID == "" && strings.TrimSpace(dataSource.UID) != "" {
			pluginUID = strings.TrimSpace(dataSource.UID)
		}
		if dataSource.JSONData.Metrics != nil && strings.TrimSpace(dataSource.JSONData.Metrics.UID) != "" {
			metricsUID = strings.TrimSpace(dataSource.JSONData.Metrics.UID)
		}
	}

	if pluginUID == "" {
		return "", "", fmt.Errorf("synthetic monitoring datasource not found")
	}
	return pluginUID, metricsUID, nil
}

func querySyntheticMetricValue(client *Client, dataSourceUID, expr string, from, to time.Time) (*float64, error) {
	request := grafanaQueryRequest{
		Queries: []grafanaQuery{
			{
				RefID:      "A",
				Datasource: map[string]string{"uid": strings.TrimSpace(dataSourceUID)},
				Expr:       expr,
				Query:      expr,
			},
		},
		From: strconv.FormatInt(from.UnixMilli(), 10),
		To:   strconv.FormatInt(to.UnixMilli(), 10),
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling synthetic metric query: %v", err)
	}

	responseBody, status, err := client.execRequest(http.MethodPost, "/api/ds/query", bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("grafana metrics query failed with status %d: %s", status, string(responseBody))
	}

	return extractFirstNumberFromDataSourceQuery(responseBody)
}

func extractFirstNumberFromDataSourceQuery(responseBody []byte) (*float64, error) {
	var response struct {
		Results map[string]struct {
			Frames []struct {
				Data struct {
					Values []any `json:"values"`
				} `json:"data"`
			} `json:"frames"`
		} `json:"results"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing grafana metrics response: %v", err)
	}

	for _, result := range response.Results {
		for _, frame := range result.Frames {
			if len(frame.Data.Values) < 2 {
				continue
			}

			if value, ok := extractLastNumericValue(frame.Data.Values[1]); ok {
				return &value, nil
			}
		}
	}

	return nil, fmt.Errorf("no numeric result found in grafana metrics response")
}

func extractLastNumericValue(raw any) (float64, bool) {
	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return 0, false
	}

	for i := len(values) - 1; i >= 0; i-- {
		switch typed := values[i].(type) {
		case float64:
			return typed, true
		case int:
			return float64(typed), true
		case int64:
			return float64(typed), true
		case json.Number:
			value, err := typed.Float64()
			if err == nil {
				return value, true
			}
		case string:
			value, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
			if err == nil {
				return value, true
			}
		}
	}

	return 0, false
}

// probeAvgToOutcome maps the average probe_success value across all probe
// locations to an outcome string matching Grafana's own per-probe status model.
func probeAvgToOutcome(avg float64) string {
	switch {
	case avg >= 1.0:
		return "Up"
	case avg <= 0:
		return "Down"
	default:
		return "Partial"
	}
}

func buildSuccessRunsQuery(check *SyntheticCheck) string {
	return fmt.Sprintf(
		`sum(increase(probe_all_success_sum{job=%q, instance=%q}[%s]))`,
		check.Job,
		check.Target,
		syntheticsMetricsLookback,
	)
}

func buildTotalRunsQuery(check *SyntheticCheck) string {
	return fmt.Sprintf(
		`sum(increase(probe_all_success_count{job=%q, instance=%q}[%s]))`,
		check.Job,
		check.Target,
		syntheticsMetricsLookback,
	)
}

func buildAverageLatencyQuery(check *SyntheticCheck) string {
	return fmt.Sprintf(
		`sum((rate(probe_all_duration_seconds_sum{probe=~".*", instance=%q, job=%q}[%s]) OR rate(probe_duration_seconds_sum{probe=~".*", instance=%q, job=%q}[%s]))) / sum((rate(probe_all_duration_seconds_count{probe=~".*", instance=%q, job=%q}[%s]) OR rate(probe_duration_seconds_count{probe=~".*", instance=%q, job=%q}[%s])))`,
		check.Target,
		check.Job,
		syntheticsMetricsLookback,
		check.Target,
		check.Job,
		syntheticsMetricsLookback,
		check.Target,
		check.Job,
		syntheticsMetricsLookback,
		check.Target,
		check.Job,
		syntheticsMetricsLookback,
	)
}

func buildUptimeQuery(check *SyntheticCheck) string {
	frequencySeconds := syntheticCheckFrequencySeconds(check)
	return fmt.Sprintf(
		`avg_over_time((max by () (max_over_time(probe_success{job=%q, instance=%q}[%ds])))[%s:%ds])`,
		check.Job,
		check.Target,
		frequencySeconds,
		syntheticsMetricsLookback,
		frequencySeconds,
	)
}

// buildCurrentOutcomeQuery returns a PromQL query that computes the fraction of
// probe locations currently passing, using the same probe_success gauge that
// Grafana's own UI uses to show per-probe success/failure status.
//
// The result is an average across all probe locations:
//
//	1.0       → all probes passing  → Up
//	0 < x < 1 → some passing, some failing → Partial
//	0.0       → all probes failing  → Down
func buildCurrentOutcomeQuery(check *SyntheticCheck) string {
	return fmt.Sprintf(
		`avg(last_over_time(probe_success{job=%q, instance=%q}[2h]))`,
		check.Job,
		check.Target,
	)
}

func buildLastExecutionQuery(check *SyntheticCheck) string {
	return fmt.Sprintf(`max(timestamp(probe_success{job=%q, instance=%q}))`, check.Job, check.Target)
}

func buildSSLEarliestExpiryQuery(check *SyntheticCheck) string {
	return fmt.Sprintf(
		`min(last_over_time(probe_ssl_earliest_cert_expiry{job=%q, instance=%q}[%s]))`,
		check.Job,
		check.Target,
		syntheticsMetricsLookback,
	)
}

func syntheticCheckFrequencySeconds(check *SyntheticCheck) int64 {
	if check == nil || check.Frequency <= 0 {
		return 60
	}

	frequencySeconds := check.Frequency / 1000
	if frequencySeconds <= 0 {
		return 1
	}

	return frequencySeconds
}
