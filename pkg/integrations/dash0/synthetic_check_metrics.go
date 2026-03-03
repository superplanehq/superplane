package dash0

import (
	"fmt"
	"math"
	"strconv"
)

// SyntheticCheckMetrics contains operational metrics for a synthetic check.
type SyntheticCheckMetrics struct {
	HealthyRuns24h  *int     `json:"healthyRuns24h"`
	CriticalRuns24h *int     `json:"criticalRuns24h"`
	TotalRuns24h    *int     `json:"totalRuns24h"`
	AvgDuration24h  *float64 `json:"avgDuration24hMs"` // Milliseconds
	HealthyRuns7d   *int     `json:"healthyRuns7d"`
	CriticalRuns7d  *int     `json:"criticalRuns7d"`
	TotalRuns7d     *int     `json:"totalRuns7d"`
	AvgDuration7d   *float64 `json:"avgDuration7dMs"` // Milliseconds
}

// FetchSyntheticCheckMetrics queries the Dash0 Prometheus API for operational metrics
// of a synthetic check identified by its dash0_check_id.
func FetchSyntheticCheckMetrics(client *Client, dataset, checkID string) *SyntheticCheckMetrics {
	metrics := &SyntheticCheckMetrics{}

	totalRuns24h := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s"}[24h]))`,
		checkID,
	))
	if totalRuns24h != nil {
		v := int(math.Round(*totalRuns24h))
		metrics.TotalRuns24h = &v
	}

	healthy24h := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s", dash0_synthetic_check_outcome="Healthy"}[24h]))`,
		checkID,
	))
	if healthy24h != nil {
		v := int(math.Round(*healthy24h))
		metrics.HealthyRuns24h = &v
	}

	critical24h := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s", dash0_synthetic_check_outcome="Critical"}[24h]))`,
		checkID,
	))
	if critical24h != nil {
		v := int(math.Round(*critical24h))
		metrics.CriticalRuns24h = &v
	}

	totalDuration24h := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.http.total.duration", dash0_check_id="%s"}[24h]))`,
		checkID,
	))
	if totalDuration24h != nil && totalRuns24h != nil && *totalRuns24h > 0 {
		avgMs := math.Round(*totalDuration24h / *totalRuns24h * 1000)
		metrics.AvgDuration24h = &avgMs
	}

	totalRuns7d := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s"}[7d]))`,
		checkID,
	))
	if totalRuns7d != nil {
		v := int(math.Round(*totalRuns7d))
		metrics.TotalRuns7d = &v
	}

	healthy7d := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s", dash0_synthetic_check_outcome="Healthy"}[7d]))`,
		checkID,
	))
	if healthy7d != nil {
		v := int(math.Round(*healthy7d))
		metrics.HealthyRuns7d = &v
	}

	critical7d := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s", dash0_synthetic_check_outcome="Critical"}[7d]))`,
		checkID,
	))
	if critical7d != nil {
		v := int(math.Round(*critical7d))
		metrics.CriticalRuns7d = &v
	}

	totalDuration7d := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.http.total.duration", dash0_check_id="%s"}[7d]))`,
		checkID,
	))
	if totalDuration7d != nil && totalRuns7d != nil && *totalRuns7d > 0 {
		avgMs := math.Round(*totalDuration7d / *totalRuns7d * 1000)
		metrics.AvgDuration7d = &avgMs
	}

	return metrics
}

// queryInstantScalar executes a Prometheus instant query and returns the scalar result value.
func queryInstantScalar(client *Client, dataset, query string) *float64 {
	result, err := client.ExecutePrometheusInstantQuery(query, dataset)
	if err != nil {
		return nil
	}

	data, ok := result["data"].(PrometheusResponseData)
	if !ok || len(data.Result) == 0 {
		return nil
	}

	if len(data.Result[0].Value) < 2 {
		return nil
	}

	valStr, ok := data.Result[0].Value[1].(string)
	if !ok {
		return nil
	}

	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return nil
	}

	return &val
}
