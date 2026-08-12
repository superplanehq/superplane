package cloudwatch

import (
	"fmt"
	"strings"
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
