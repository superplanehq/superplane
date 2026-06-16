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

func metricByType(metricType string) (metricOption, bool) {
	for _, m := range instanceMetricOptions {
		if m.Value == metricType {
			return m, true
		}
	}
	return metricOption{}, false
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

// --- policy-level vocabulary (combiner, severity, alert strategy, labels) ---

var combinerOptions = []configuration.FieldOption{
	{Label: "Any condition is met (OR)", Value: "OR"},
	{Label: "All conditions are met (AND)", Value: "AND"},
	{Label: "All conditions, same resource (AND_WITH_MATCHING_RESOURCE)", Value: "AND_WITH_MATCHING_RESOURCE"},
}

func isValidCombiner(combiner string) bool {
	switch combiner {
	case "OR", "AND", "AND_WITH_MATCHING_RESOURCE":
		return true
	}
	return false
}

var severityOptions = []configuration.FieldOption{
	{Label: "Critical", Value: "CRITICAL"},
	{Label: "Error", Value: "ERROR"},
	{Label: "Warning", Value: "WARNING"},
}

func isValidSeverity(severity string) bool {
	switch severity {
	case "", "CRITICAL", "ERROR", "WARNING":
		return true
	}
	return false
}

var autoCloseOptions = []configuration.FieldOption{
	{Label: "30 minutes", Value: "1800s"},
	{Label: "1 hour", Value: "3600s"},
	{Label: "1 day", Value: "86400s"},
	{Label: "7 days", Value: "604800s"},
}

var notificationRateLimitOptions = []configuration.FieldOption{
	{Label: "Every 5 minutes", Value: "300s"},
	{Label: "Every 30 minutes", Value: "1800s"},
	{Label: "Every 1 hour", Value: "3600s"},
}

// buildAlertStrategy assembles the optional alertStrategy block, returning nil
// when neither auto-close nor a notification rate limit is configured.
func buildAlertStrategy(autoClose, rateLimit string) map[string]any {
	strategy := map[string]any{}
	if autoClose != "" {
		strategy["autoClose"] = autoClose
	}
	if rateLimit != "" {
		strategy["notificationRateLimit"] = map[string]any{"period": rateLimit}
	}
	if len(strategy) == 0 {
		return nil
	}
	return strategy
}

// KeyValueSpec is one user label (key/value pair).
type KeyValueSpec struct {
	Key   string `mapstructure:"key"`
	Value string `mapstructure:"value"`
}

func buildUserLabels(pairs []KeyValueSpec) map[string]string {
	labels := map[string]string{}
	for _, p := range pairs {
		key := strings.TrimSpace(p.Key)
		if key != "" {
			labels[key] = p.Value
		}
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

// buildDocumentation assembles the optional documentation block.
func buildDocumentation(content, subject string) map[string]any {
	content = strings.TrimSpace(content)
	subject = strings.TrimSpace(subject)
	if content == "" && subject == "" {
		return nil
	}
	doc := map[string]any{"mimeType": "text/markdown"}
	if content != "" {
		doc["content"] = content
	}
	if subject != "" {
		doc["subject"] = subject
	}
	return doc
}

// alertPolicy models the subset of the Cloud Monitoring AlertPolicy resource we
// read back after create/get/update.
type alertPolicy struct {
	Name                 string   `json:"name"`
	DisplayName          string   `json:"displayName"`
	Combiner             string   `json:"combiner"`
	Severity             string   `json:"severity"`
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
		ConditionPrometheusQueryLanguage *struct {
			Query              string `json:"query"`
			Duration           string `json:"duration"`
			EvaluationInterval string `json:"evaluationInterval"`
		} `json:"conditionPrometheusQueryLanguage"`
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
	if p.Severity != "" {
		payload["severity"] = p.Severity
	}
	if len(p.NotificationChannels) > 0 {
		payload["notificationChannels"] = p.NotificationChannels
	}
	if len(p.Conditions) > 0 {
		switch first := p.Conditions[0]; {
		case first.ConditionThreshold != nil:
			c := first.ConditionThreshold
			payload["conditionType"] = conditionKindThreshold
			payload["filter"] = c.Filter
			payload["comparison"] = c.Comparison
			payload["thresholdValue"] = c.ThresholdValue
			payload["duration"] = c.Duration
		case first.ConditionPrometheusQueryLanguage != nil:
			c := first.ConditionPrometheusQueryLanguage
			payload["conditionType"] = conditionKindPromQL
			payload["query"] = c.Query
			if c.Duration != "" {
				payload["duration"] = c.Duration
			}
			if c.EvaluationInterval != "" {
				payload["evaluationInterval"] = c.EvaluationInterval
			}
		}
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
