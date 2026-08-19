package cloudwatch

import (
	"fmt"
	"strings"
	"time"
)

func requireRegion(value string) (string, error) {
	region := strings.TrimSpace(value)
	if region == "" {
		return "", fmt.Errorf("region is required")
	}

	return region, nil
}

func requireAlarmName(value string) (string, error) {
	alarmName := strings.TrimSpace(value)
	if alarmName == "" {
		return "", fmt.Errorf("alarm name is required")
	}

	return alarmName, nil
}

func requireNamespace(value string) (string, error) {
	namespace := strings.TrimSpace(value)
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	return namespace, nil
}

func requireMetricName(value string) (string, error) {
	metricName := strings.TrimSpace(value)
	if metricName == "" {
		return "", fmt.Errorf("metric name is required")
	}

	return metricName, nil
}

func requireStatistic(value string) (string, error) {
	statistic := strings.TrimSpace(value)
	if statistic == "" {
		return "", fmt.Errorf("statistic is required")
	}

	return statistic, nil
}

func requireComparisonOperator(value string) (string, error) {
	comparisonOperator := strings.TrimSpace(value)
	if comparisonOperator == "" {
		return "", fmt.Errorf("comparison operator is required")
	}

	return comparisonOperator, nil
}

func requireTreatMissingData(value string) (string, error) {
	treatMissingData := strings.TrimSpace(value)
	if treatMissingData == "" {
		return "", fmt.Errorf("treat missing data is required")
	}

	return treatMissingData, nil
}

func requireThreshold(configuration any, threshold float64) (float64, error) {
	if !hasConfigKey(configuration, "threshold") {
		return 0, fmt.Errorf("threshold is required")
	}

	return threshold, nil
}

// requireDatapointsWithinEvaluationPeriods enforces the "M out of N" rule:
// CloudWatch rejects an alarm whose datapoints to alarm exceed its evaluation periods.
func requireDatapointsWithinEvaluationPeriods(datapointsToAlarm, evaluationPeriods int) error {
	if datapointsToAlarm > 0 && datapointsToAlarm > evaluationPeriods {
		return fmt.Errorf(
			"datapoints to alarm (%d) cannot be greater than evaluation periods (%d)",
			datapointsToAlarm, evaluationPeriods,
		)
	}

	return nil
}

// effectiveEvaluationPeriods mirrors the default the client applies when the
// field is left empty, so validation compares against what CloudWatch will see.
func effectiveEvaluationPeriods(evaluationPeriods int) int {
	if evaluationPeriods <= 0 {
		return defaultAlarmEvaluationPeriods
	}

	return evaluationPeriods
}

func requireAlarmMuteRuleName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	return name, nil
}

// requireAlarmNames drops blank entries and enforces PutAlarmMuteRule's own
// bounds on how many alarms a single mute rule can target.
func requireAlarmNames(values []string) ([]string, error) {
	names := make([]string, 0, len(values))
	for _, value := range values {
		if name := strings.TrimSpace(value); name != "" {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("at least one alarm is required")
	}

	if len(names) > maxMuteRuleAlarms {
		return nil, fmt.Errorf("a mute rule can target at most %d alarms", maxMuteRuleAlarms)
	}

	return names, nil
}

func requireScheduleExpression(value string) (string, error) {
	expression := strings.TrimSpace(value)
	if expression == "" {
		return "", fmt.Errorf("schedule expression is required")
	}

	if !strings.HasPrefix(expression, "cron(") && !strings.HasPrefix(expression, "at(") {
		return "", fmt.Errorf(`schedule expression must be a recurring "cron(...)" or one-time "at(...)" expression`)
	}

	return expression, nil
}

func requireScheduleDuration(value string) (string, error) {
	duration := strings.TrimSpace(value)
	if duration == "" {
		return "", fmt.Errorf("duration is required")
	}

	return duration, nil
}

// muteRuleDatetimeLayouts covers the value shape produced by the "datetime"
// field's HTML datetime-local input, which carries no timezone of its own.
var muteRuleDatetimeLayouts = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

// parseMuteRuleTimestamp accepts either a full RFC3339 timestamp, whose own
// offset always wins, or a bare datetime-local value, interpreted in the
// given timezone since it carries no offset of its own.
func parseMuteRuleTimestamp(raw, timezone string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}

	for _, layout := range muteRuleDatetimeLayouts {
		if parsed, err := time.ParseInLocation(layout, raw, location); err == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid timestamp %q: expected RFC3339 or YYYY-MM-DDTHH:MM", raw)
}

// parseOptionalMuteRuleTimestamp is parseMuteRuleTimestamp for a togglable
// field: a blank value is left unset rather than treated as an error.
func parseOptionalMuteRuleTimestamp(raw, timezone string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parsed, err := parseMuteRuleTimestamp(raw, timezone)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

// effectiveTimezone mirrors the "timezone" field's own UI default, so
// validation resolves the same value the client will send.
func effectiveTimezone(value string) string {
	if timezone := strings.TrimSpace(value); timezone != "" {
		return timezone
	}

	return "UTC"
}

func hasConfigKey(configuration any, key string) bool {
	configurationMap, ok := configuration.(map[string]any)
	if !ok {
		return false
	}

	value, exists := configurationMap[key]
	return exists && value != nil
}

var updateAlarmFieldKeys = []string{
	"thresholdCondition",
	"statistic",
	"period",
	"evaluationPeriods",
	"datapointsToAlarm",
	"alarmDescription",
	"treatMissingData",
	"unit",
	"actionsEnabled",
	"alarmActions",
	"ec2Action",
	"okActions",
	"insufficientDataActions",
}

var updateAlarmFieldLabels = map[string]string{
	"thresholdCondition":      "Threshold",
	"statistic":               "Statistic",
	"period":                  "Period",
	"evaluationPeriods":       "Evaluation Periods",
	"datapointsToAlarm":       "Datapoints To Alarm",
	"alarmDescription":        "Description",
	"treatMissingData":        "Treat Missing Data",
	"unit":                    "Unit",
	"actionsEnabled":          "Actions Enabled",
	"alarmActions":            "Alarm Actions",
	"ec2Action":               "EC2 Action",
	"okActions":               "OK Actions",
	"insufficientDataActions": "Insufficient Data Actions",
}

func updatedAlarmFieldLabels(configuration any) []string {
	labels := make([]string, 0, len(updateAlarmFieldKeys))
	for _, key := range updateAlarmFieldKeys {
		if hasConfigKey(configuration, key) {
			if label, ok := updateAlarmFieldLabels[key]; ok {
				labels = append(labels, label)
			}
		}
	}

	return labels
}

func requireAtLeastOneAlarmUpdate(configuration any) error {
	for _, key := range updateAlarmFieldKeys {
		if hasConfigKey(configuration, key) {
			return nil
		}
	}

	return fmt.Errorf("at least one alarm property to update is required")
}
