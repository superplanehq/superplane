package dash0

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SyntheticCheckMetrics holds key metrics for a synthetic check (uptime, duration, fails, status, etc.).
type SyntheticCheckMetrics struct {
	Uptime24hPct    *float64 `json:"uptime24hPct,omitempty"`
	Uptime7dPct     *float64 `json:"uptime7dPct,omitempty"`
	AvgDuration7dMs *float64 `json:"avgDuration7dMs,omitempty"`
	Fails7d         *int     `json:"fails7d,omitempty"`
	LastCheckAt     *string  `json:"lastCheckAt,omitempty"` // ISO or relative e.g. "5m ago"
	DownFor7dSec    *float64 `json:"downFor7dSec,omitempty"`
	// Status is the current check state from Prometheus (e.g. "Clear", "Degraded", "Critical").
	Status *string `json:"status,omitempty"`
}

// GetSyntheticCheckMetrics queries Prometheus for metrics of a synthetic check.
// Returns nil if queries fail or metrics are not available.
// Dash0 exposes native Prometheus metrics (__name__) with underscores, e.g. dash0_synthetic_check_runs_total,
// dash0_synthetic_check_http_total_duration_seconds_*, and labels dash0_check_id, dash0_synthetic_check_outcome.
func (c *Client) GetSyntheticCheckMetrics(checkID string, dataset string) (*SyntheticCheckMetrics, error) {
	if checkID == "" {
		return nil, nil
	}
	// Escape for PromQL: backslash and double-quote
	escaped := strings.ReplaceAll(checkID, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)

	// Dash0 Prometheus uses __name__ (metric names with underscores) and dash0_check_id (confirmed via API discovery).
	labelFilter := fmt.Sprintf(`dash0_check_type="synthetic.http", dash0_check_id="%s"`, escaped)
	out := c.tryFetchMetrics(dataset, labelFilter)
	if out == nil {
		return nil, nil
	}
	// Return nil only when no metric field was populated (including Status).
	if out.Uptime24hPct == nil && out.Uptime7dPct == nil && out.AvgDuration7dMs == nil &&
		out.Fails7d == nil && out.LastCheckAt == nil && out.DownFor7dSec == nil && out.Status == nil {
		return nil, nil
	}
	return out, nil
}

func (c *Client) tryFetchMetrics(dataset, labelFilter string) *SyntheticCheckMetrics {
	// Dash0 exposes native Prometheus metrics: __name__="dash0_synthetic_check_runs_total", etc.
	runsSelector := fmt.Sprintf(`{__name__="dash0_synthetic_check_runs_total", %s}`, labelFilter)
	q24 := fmt.Sprintf(`sum(increase(%s[24h]))`, runsSelector)
	q7 := fmt.Sprintf(`sum(increase(%s[7d]))`, runsSelector)
	// Failures: runs with outcome Critical or Degraded (success is typically Pass/Clear).
	failsSelector := fmt.Sprintf(`{__name__="dash0_synthetic_check_runs_total", %s, dash0_synthetic_check_outcome=~"Critical|Degraded"}`, labelFilter)
	qFails := fmt.Sprintf(`sum(increase(%s[7d])) or vector(0)`, failsSelector)
	qFails24 := fmt.Sprintf(`sum(increase(%s[24h])) or vector(0)`, failsSelector)
	// Duration: dash0_synthetic_check_http_total_duration_seconds_* (values in seconds; *1000 for ms).
	durSumSelector := fmt.Sprintf(`{__name__="dash0_synthetic_check_http_total_duration_seconds_sum", %s}`, labelFilter)
	durCountSelector := fmt.Sprintf(`{__name__="dash0_synthetic_check_http_total_duration_seconds_count", %s}`, labelFilter)
	qAvgDurNum := fmt.Sprintf(`sum(increase(%s[7d]))`, durSumSelector)
	qAvgDurDen := fmt.Sprintf(`sum(increase(%s[7d]))`, durCountSelector)

	metrics := &SyntheticCheckMetrics{}

	total24, _ := c.scalarFromInstantQuery(q24, dataset)
	total7, _ := c.scalarFromInstantQuery(q7, dataset)
	fails7, _ := c.scalarFromInstantQuery(qFails, dataset)
	fails24, _ := c.scalarFromInstantQuery(qFails24, dataset)
	avgDurSum, _ := c.scalarFromInstantQuery(qAvgDurNum, dataset)
	avgDurCount, _ := c.scalarFromInstantQuery(qAvgDurDen, dataset)
	var avgDur *float64
	if avgDurSum != nil && avgDurCount != nil && *avgDurCount > 0 {
		ms := (*avgDurSum / *avgDurCount) * 1000 // seconds -> ms
		avgDur = &ms
	}

	if total24 != nil && *total24 > 0 {
		success24 := *total24
		if fails24 != nil && *fails24 >= 0 {
			success24 = *total24 - *fails24
		}
		pct := 100 * success24 / *total24
		metrics.Uptime24hPct = &pct
	}
	if total7 != nil && *total7 > 0 {
		success7 := *total7
		if fails7 != nil && *fails7 >= 0 {
			success7 = *total7 - *fails7
		}
		pct := 100 * success7 / *total7
		metrics.Uptime7dPct = &pct
	}

	if fails7 != nil && *fails7 >= 0 {
		f := int(*fails7)
		metrics.Fails7d = &f
	}

	if avgDur != nil && *avgDur >= 0 {
		metrics.AvgDuration7dMs = avgDur
	}

	// Down for 7d: approximate as fails * interval (e.g. 1m = 60s per fail)
	if metrics.Fails7d != nil && *metrics.Fails7d > 0 {
		downSec := float64(*metrics.Fails7d) * 60 // assume 1m interval
		metrics.DownFor7dSec = &downSec
	}

	// Last check: query last run timestamp if available
	lastTs := c.lastTimestampFromInstantQuery(fmt.Sprintf(`max(%s)`, runsSelector), dataset)
	if lastTs != "" {
		metrics.LastCheckAt = &lastTs
	}

	// Current status: Dash0 exposes dash0_check_status (gauge 0=clear, 1=degraded, 2=critical).
	statusSelector := fmt.Sprintf(`{__name__="dash0_check_status", %s}`, labelFilter)
	if v, ok := c.scalarFromInstantQuery(fmt.Sprintf(`max(%s)`, statusSelector), dataset); ok && v != nil {
		s := statusFromGauge(*v)
		if s != "" {
			metrics.Status = &s
		}
	}

	// If no metrics were populated, return nil so caller can try another label or skip
	if metrics.Uptime24hPct == nil && metrics.Uptime7dPct == nil && metrics.AvgDuration7dMs == nil &&
		metrics.Fails7d == nil && metrics.LastCheckAt == nil && metrics.DownFor7dSec == nil && metrics.Status == nil {
		return nil
	}
	return metrics
}

// statusFromGauge maps Dash0 check status gauge value to label (0=Clear, 1=Degraded, 2=Critical).
func statusFromGauge(v float64) string {
	switch int(v) {
	case 0:
		return "Clear"
	case 1:
		return "Degraded"
	case 2:
		return "Critical"
	default:
		return ""
	}
}

// scalarFromInstantQuery runs query and returns the first scalar value, or (nil, false) on error/empty.
func (c *Client) scalarFromInstantQuery(promQL, dataset string) (*float64, bool) {
	res, err := c.ExecutePrometheusInstantQuery(promQL, dataset)
	if err != nil {
		return nil, false
	}
	data, _ := res["data"].(map[string]any)
	if data == nil {
		return nil, false
	}
	results, _ := data["result"].([]any)
	if len(results) == 0 {
		return nil, false
	}
	first, _ := results[0].(map[string]any)
	if first == nil {
		return nil, false
	}
	val, _ := first["value"].([]any)
	if len(val) < 2 {
		return nil, false
	}
	switch v := val[1].(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, false
		}
		return &f, true
	case float64:
		return &v, true
	}
	return nil, false
}

// lastTimestampFromInstantQuery returns a human-friendly "X m ago" or ISO string for the last sample.
func (c *Client) lastTimestampFromInstantQuery(promQL, dataset string) string {
	res, err := c.ExecutePrometheusInstantQuery(promQL, dataset)
	if err != nil {
		return ""
	}
	data, _ := res["data"].(map[string]any)
	if data == nil {
		return ""
	}
	results, _ := data["result"].([]any)
	if len(results) == 0 {
		return ""
	}
	first, _ := results[0].(map[string]any)
	if first == nil {
		return ""
	}
	val, _ := first["value"].([]any)
	if len(val) < 1 {
		return ""
	}
	var ts float64
	switch t := val[0].(type) {
	case float64:
		ts = t
	case string:
		ts, _ = strconv.ParseFloat(t, 64)
	default:
		return ""
	}
	then := time.Unix(int64(ts), 0)
	ago := time.Since(then)
	if ago < time.Minute {
		return "just now"
	}
	if ago < time.Hour {
		return fmt.Sprintf("%.0f m ago", ago.Minutes())
	}
	if ago < 24*time.Hour {
		return fmt.Sprintf("%.0f h ago", ago.Hours())
	}
	return then.Format(time.RFC3339)
}
