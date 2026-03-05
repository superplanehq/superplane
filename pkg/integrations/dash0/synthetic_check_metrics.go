package dash0

import (
	"fmt"
	"math"
	"strconv"

	"github.com/superplanehq/superplane/pkg/core"
)

// SyntheticCheckMetrics contains operational metrics for a synthetic check.
type SyntheticCheckMetrics struct {
	HealthyRuns24h  int     `json:"healthyRuns24h"`
	CriticalRuns24h int     `json:"criticalRuns24h"`
	TotalRuns24h    int     `json:"totalRuns24h"`
	AvgDuration24h  float64 `json:"avgDuration24hMs"` // Milliseconds
	HealthyRuns7d   int     `json:"healthyRuns7d"`
	CriticalRuns7d  int     `json:"criticalRuns7d"`
	TotalRuns7d     int     `json:"totalRuns7d"`
	AvgDuration7d   float64 `json:"avgDuration7dMs"` // Milliseconds
	LastOutcome     string  `json:"lastOutcome"`     // Most recent run outcome: Healthy or Critical
}

// FetchSyntheticCheckMetrics queries the Dash0 Prometheus API for operational metrics
// of a synthetic check identified by its dash0_check_id.
func FetchSyntheticCheckMetrics(ctx core.ExecutionContext, client *Client, dataset, checkID string) *SyntheticCheckMetrics {
	metrics := &SyntheticCheckMetrics{}

	totalRuns24h := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s"}[24h]))`,
		checkID,
	))
	if totalRuns24h != nil {
		metrics.TotalRuns24h = int(math.Round(*totalRuns24h))
	}

	healthy24h := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s", dash0_synthetic_check_outcome="Healthy"}[24h]))`,
		checkID,
	))
	if healthy24h != nil {
		metrics.HealthyRuns24h = int(math.Round(*healthy24h))
	}

	critical24h := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s", dash0_synthetic_check_outcome="Critical"}[24h]))`,
		checkID,
	))
	if critical24h != nil {
		metrics.CriticalRuns24h = int(math.Round(*critical24h))
	}

	totalDuration24h := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.http.total.duration", dash0_check_id="%s"}[24h]))`,
		checkID,
	))
	if totalDuration24h != nil && totalRuns24h != nil && *totalRuns24h > 0 {
		metrics.AvgDuration24h = math.Round(*totalDuration24h / *totalRuns24h * 1000)
	}

	totalRuns7d := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s"}[7d]))`,
		checkID,
	))
	if totalRuns7d != nil {
		metrics.TotalRuns7d = int(math.Round(*totalRuns7d))
	}

	healthy7d := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s", dash0_synthetic_check_outcome="Healthy"}[7d]))`,
		checkID,
	))
	if healthy7d != nil {
		metrics.HealthyRuns7d = int(math.Round(*healthy7d))
	}

	critical7d := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s", dash0_synthetic_check_outcome="Critical"}[7d]))`,
		checkID,
	))
	if critical7d != nil {
		metrics.CriticalRuns7d = int(math.Round(*critical7d))
	}

	totalDuration7d := queryInstantScalar(client, dataset, fmt.Sprintf(
		`sum(increase({otel_metric_name="dash0.synthetic_check.http.total.duration", dash0_check_id="%s"}[7d]))`,
		checkID,
	))
	if totalDuration7d != nil && totalRuns7d != nil && *totalRuns7d > 0 {
		metrics.AvgDuration7d = math.Round(*totalDuration7d / *totalRuns7d * 1000)
	}

	// Fetch the most recent run outcome.
	metrics.LastOutcome = fetchLastSyntheticCheckOutcome(ctx, client, dataset, checkID)

	return metrics
}

func fetchLastSyntheticCheckOutcome(ctx core.ExecutionContext, client *Client, dataset, checkID string) string {
	query := fmt.Sprintf(
		`topk(1, max by (dash0_synthetic_check_outcome) (timestamp({otel_metric_name="dash0.synthetic_check.runs", dash0_check_id="%s"})))`,
		checkID,
	)

	result, err := client.ExecutePrometheusInstantQuery(query, dataset)
	if err != nil {
		return ""
	}

	data, ok := result["data"].(PrometheusResponseData)
	if !ok || len(data.Result) == 0 {
		return ""
	}
	outcome := data.Result[0].Metric["dash0_synthetic_check_outcome"]
	if outcome != "Healthy" && outcome != "Critical" {
		return ""
	}

	return outcome
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
