package monitoring

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const (
	roleHintWrite = "roles/monitoring.editor (or roles/monitoring.alertPolicyEditor)"
	roleHintRead  = "roles/monitoring.viewer"
)

// apiErrorMessage formats an API error for the execution state, appending an IAM
// hint on 403 since a missing role is the most common cause of monitoring write
// failures (reads can succeed while writes are denied).
func apiErrorMessage(action, roleHint string, err error) string {
	var apiErr *gcpcommon.GCPAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("%s: %v — ensure the integration's service account has the %s IAM role", action, err, roleHint)
	}
	return fmt.Sprintf("%s: %v", action, err)
}

const (
	comparisonGT = "COMPARISON_GT"
	comparisonLT = "COMPARISON_LT"

	// alignmentPeriod is the per-series alignment window for the condition's
	// aggregation. 60s matches the native resolution of Compute Engine metrics.
	alignmentPeriod = "60s"
)

// metricOption couples a user-facing label with the Cloud Monitoring metric type
// and the aligner appropriate for it (gauge metrics use ALIGN_MEAN; delta/cumulative
// counters use ALIGN_RATE to express a per-second rate).
type metricOption struct {
	Label   string
	Value   string
	Aligner string
}

var instanceMetricOptions = []metricOption{
	{Label: "CPU utilization (fraction 0–1)", Value: "compute.googleapis.com/instance/cpu/utilization", Aligner: "ALIGN_MEAN"},
	{Label: "Sent network traffic (bytes/s)", Value: "compute.googleapis.com/instance/network/sent_bytes_count", Aligner: "ALIGN_RATE"},
	{Label: "Received network traffic (bytes/s)", Value: "compute.googleapis.com/instance/network/received_bytes_count", Aligner: "ALIGN_RATE"},
	{Label: "Disk read (bytes/s)", Value: "compute.googleapis.com/instance/disk/read_bytes_count", Aligner: "ALIGN_RATE"},
	{Label: "Disk write (bytes/s)", Value: "compute.googleapis.com/instance/disk/write_bytes_count", Aligner: "ALIGN_RATE"},
}

func metricFieldOptions() []configuration.FieldOption {
	opts := make([]configuration.FieldOption, 0, len(instanceMetricOptions))
	for _, m := range instanceMetricOptions {
		opts = append(opts, configuration.FieldOption{Label: m.Label, Value: m.Value})
	}
	return opts
}

// metricValues returns every supported metric type, used to drive the
// "show/require these fields when a metric is chosen" conditions on the Update
// component.
func metricValues() []string {
	values := make([]string, 0, len(instanceMetricOptions))
	for _, m := range instanceMetricOptions {
		values = append(values, m.Value)
	}
	return values
}

func metricByType(metricType string) (metricOption, bool) {
	for _, m := range instanceMetricOptions {
		if m.Value == metricType {
			return m, true
		}
	}
	return metricOption{}, false
}

var comparisonOptions = []configuration.FieldOption{
	{Label: "Above threshold", Value: comparisonGT},
	{Label: "Below threshold", Value: comparisonLT},
}

func isValidComparison(comparison string) bool {
	return comparison == comparisonGT || comparison == comparisonLT
}

var durationOptions = []configuration.FieldOption{
	{Label: "Immediately", Value: "0s"},
	{Label: "1 minute", Value: "60s"},
	{Label: "5 minutes", Value: "300s"},
	{Label: "10 minutes", Value: "600s"},
	{Label: "30 minutes", Value: "1800s"},
}

func isValidDuration(duration string) bool {
	for _, opt := range durationOptions {
		if opt.Value == duration {
			return true
		}
	}
	return false
}

// instanceMetricFilter builds the Cloud Monitoring filter that scopes the
// condition to a single instance metric across all GCE instances in the project.
func instanceMetricFilter(metricType string) string {
	return fmt.Sprintf(`metric.type="%s" AND resource.type="gce_instance"`, metricType)
}

// buildThresholdCondition assembles a single conditionThreshold for the policy.
func buildThresholdCondition(metricType, comparison string, threshold float64, duration string) map[string]any {
	aligner := "ALIGN_MEAN"
	if m, ok := metricByType(metricType); ok {
		aligner = m.Aligner
	}
	return map[string]any{
		"displayName": conditionDisplayName(metricType, comparison, threshold),
		"conditionThreshold": map[string]any{
			"filter":         instanceMetricFilter(metricType),
			"comparison":     comparison,
			"thresholdValue": threshold,
			"duration":       duration,
			"trigger":        map[string]any{"count": 1},
			"aggregations": []any{
				map[string]any{
					"alignmentPeriod":  alignmentPeriod,
					"perSeriesAligner": aligner,
				},
			},
		},
	}
}

func conditionDisplayName(metricType, comparison string, threshold float64) string {
	op := ">"
	if comparison == comparisonLT {
		op = "<"
	}
	return fmt.Sprintf("%s %s %g", lastSegment(metricType), op, threshold)
}

// alertPolicy models the subset of the Cloud Monitoring AlertPolicy resource we
// read back after create/get/update.
type alertPolicy struct {
	Name                 string   `json:"name"`
	DisplayName          string   `json:"displayName"`
	Combiner             string   `json:"combiner"`
	Enabled              *bool    `json:"enabled"`
	NotificationChannels []string `json:"notificationChannels"`
	Conditions           []struct {
		DisplayName        string `json:"displayName"`
		ConditionThreshold *struct {
			Filter         string  `json:"filter"`
			Comparison     string  `json:"comparison"`
			ThresholdValue float64 `json:"thresholdValue"`
			Duration       string  `json:"duration"`
		} `json:"conditionThreshold"`
	} `json:"conditions"`
}

// policyPayload normalizes an AlertPolicy into the component output payload.
func policyPayload(p *alertPolicy) map[string]any {
	payload := map[string]any{
		"name":            p.Name,
		"id":              lastSegment(p.Name),
		"displayName":     p.DisplayName,
		"combiner":        p.Combiner,
		"conditionsCount": len(p.Conditions),
	}
	if p.Enabled != nil {
		payload["enabled"] = *p.Enabled
	}
	if len(p.NotificationChannels) > 0 {
		payload["notificationChannels"] = p.NotificationChannels
	}
	if len(p.Conditions) > 0 && p.Conditions[0].ConditionThreshold != nil {
		c := p.Conditions[0].ConditionThreshold
		payload["filter"] = c.Filter
		payload["comparison"] = c.Comparison
		payload["thresholdValue"] = c.ThresholdValue
		payload["duration"] = c.Duration
	}
	return payload
}

// parsePolicyName extracts (project, name) from an alert policy value. The value
// is the resource name `projects/<project>/alertPolicies/<id>`, optionally as a
// full monitoring.googleapis.com URL. The relative name is returned so callers
// can append it to the monitoring base URL.
func parsePolicyName(value string) (project, name string, err error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return "", "", errors.New("alert policy is required")
	}
	idx := strings.Index(s, "projects/")
	if idx < 0 {
		return "", "", fmt.Errorf("alert policy %q must be a resource name like projects/<project>/alertPolicies/<id>", value)
	}
	rel := s[idx:]
	if q := strings.IndexAny(rel, "?#"); q >= 0 {
		rel = rel[:q]
	}
	rel = strings.TrimRight(rel, "/")
	parts := strings.Split(rel, "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "alertPolicies" || parts[1] == "" || parts[3] == "" {
		return "", "", fmt.Errorf("alert policy %q is not a valid alert policy name", value)
	}
	return parts[1], rel, nil
}

func lastSegment(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if i := strings.LastIndex(value, "/"); i >= 0 {
		return value[i+1:]
	}
	return value
}
