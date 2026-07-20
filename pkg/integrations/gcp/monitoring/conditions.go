package monitoring

import (
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	// Cloud Monitoring threshold conditions only support GT and LT.
	comparisonGT = "COMPARISON_GT"
	comparisonLT = "COMPARISON_LT"

	defaultAlignmentPeriod = "60s"
	reduceNone             = "REDUCE_NONE"

	triggerCount   = "count"
	triggerPercent = "percent"

	// conditionKind selects how a policy's condition is expressed: a curated
	// instance metric threshold, or a PromQL query against Google Managed
	// Prometheus (the Prometheus-style alerting rule).
	conditionKindThreshold = "threshold"
	conditionKindPromQL    = "promql"
)

var conditionKindOptions = []configuration.FieldOption{
	{Label: "Metric threshold", Value: conditionKindThreshold},
	{Label: "PromQL query (Managed Prometheus)", Value: conditionKindPromQL},
}

// evaluationIntervalOptions are how often Cloud Monitoring evaluates a PromQL
// condition's query.
var evaluationIntervalOptions = []configuration.FieldOption{
	{Label: "30 seconds", Value: "30s"},
	{Label: "1 minute", Value: "60s"},
	{Label: "5 minutes", Value: "300s"},
}

func isValidEvaluationInterval(v string) bool {
	return optionHasValue(evaluationIntervalOptions, v)
}

// ConditionSpec is one threshold condition within an alerting policy.
type ConditionSpec struct {
	MetricType         string   `mapstructure:"metricType"`
	Comparison         string   `mapstructure:"comparison"`
	Threshold          *float64 `mapstructure:"threshold"`
	Duration           string   `mapstructure:"duration"`
	Aligner            string   `mapstructure:"aligner"`
	AlignmentPeriod    string   `mapstructure:"alignmentPeriod"`
	CrossSeriesReducer string   `mapstructure:"crossSeriesReducer"`
	GroupByFields      []string `mapstructure:"groupByFields"`
	TriggerType        string   `mapstructure:"triggerType"`
	TriggerValue       *float64 `mapstructure:"triggerValue"`
}

var comparisonOptions = []configuration.FieldOption{
	{Label: "Above ( > )", Value: comparisonGT},
	{Label: "Below ( < )", Value: comparisonLT},
}

func isValidComparison(comparison string) bool {
	switch comparison {
	case comparisonGT, comparisonLT:
		return true
	}
	return false
}

var alignerOptions = []configuration.FieldOption{
	{Label: "Mean", Value: "ALIGN_MEAN"},
	{Label: "Max", Value: "ALIGN_MAX"},
	{Label: "Min", Value: "ALIGN_MIN"},
	{Label: "Sum", Value: "ALIGN_SUM"},
	{Label: "Rate (per second)", Value: "ALIGN_RATE"},
	{Label: "Count", Value: "ALIGN_COUNT"},
	{Label: "99th percentile", Value: "ALIGN_PERCENTILE_99"},
	{Label: "95th percentile", Value: "ALIGN_PERCENTILE_95"},
	{Label: "50th percentile (median)", Value: "ALIGN_PERCENTILE_50"},
	{Label: "Standard deviation", Value: "ALIGN_STDDEV"},
}

func isValidAligner(aligner string) bool {
	return optionHasValue(alignerOptions, aligner)
}

var alignmentPeriodOptions = []configuration.FieldOption{
	{Label: "1 minute", Value: "60s"},
	{Label: "5 minutes", Value: "300s"},
	{Label: "10 minutes", Value: "600s"},
	{Label: "30 minutes", Value: "1800s"},
	{Label: "1 hour", Value: "3600s"},
}

var reducerOptions = []configuration.FieldOption{
	{Label: "None (per time series)", Value: reduceNone},
	{Label: "Mean", Value: "REDUCE_MEAN"},
	{Label: "Max", Value: "REDUCE_MAX"},
	{Label: "Min", Value: "REDUCE_MIN"},
	{Label: "Sum", Value: "REDUCE_SUM"},
	{Label: "Count", Value: "REDUCE_COUNT"},
}

func isValidReducer(reducer string) bool {
	return optionHasValue(reducerOptions, reducer)
}

var triggerTypeOptions = []configuration.FieldOption{
	{Label: "Number of time series", Value: triggerCount},
	{Label: "Percent of time series", Value: triggerPercent},
}

func optionHasValue(options []configuration.FieldOption, value string) bool {
	for _, o := range options {
		if o.Value == value {
			return true
		}
	}
	return false
}

// conditionFields is the schema for one item in the conditions list.
func conditionFields() []configuration.Field {
	return []configuration.Field{
		{
			Name: "metricType", Label: "Metric", Type: configuration.FieldTypeSelect, Required: true,
			Description: "The Compute Engine instance metric to monitor.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: metricFieldOptions()}},
		},
		{
			Name: "comparison", Label: "Comparison", Type: configuration.FieldTypeSelect, Required: true,
			Description: "How to compare the metric value to the threshold.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: comparisonOptions}},
		},
		{
			Name: "threshold", Label: "Threshold", Type: configuration.FieldTypeNumber, Required: true,
			Description: "The numeric threshold. CPU utilization is a fraction (0.8 = 80%).",
			Placeholder: "e.g. 0.8",
		},
		{
			Name: "duration", Label: "Duration", Type: configuration.FieldTypeSelect, Required: true,
			Description: "How long the condition must hold before firing.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: durationOptions}},
		},
		{
			Name: "aligner", Label: "Aligner", Type: configuration.FieldTypeSelect, Required: false,
			Description: "Per-series alignment. Defaults to mean (or rate for counter metrics).",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: alignerOptions}},
		},
		{
			Name: "alignmentPeriod", Label: "Rolling window", Type: configuration.FieldTypeSelect, Required: false,
			Description: "The window each data point is aligned over.",
			Default:     defaultAlignmentPeriod,
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: alignmentPeriodOptions}},
		},
		{
			Name: "crossSeriesReducer", Label: "Group reducer", Type: configuration.FieldTypeSelect, Required: false,
			Description: "Combine time series across the group-by fields.",
			Default:     reduceNone,
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: reducerOptions}},
		},
		{
			Name: "groupByFields", Label: "Group by fields", Type: configuration.FieldTypeList, Required: false,
			Description: "Resource/metric labels to group by when a reducer is set (e.g. resource.zone).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Field",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name: "triggerType", Label: "Trigger when", Type: configuration.FieldTypeSelect, Required: false,
			Description: "Fire when a number or percent of time series breach.",
			Default:     triggerCount,
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: triggerTypeOptions}},
		},
		{
			Name: "triggerValue", Label: "Trigger value", Type: configuration.FieldTypeNumber, Required: false,
			Description: "Count (default 1) or percent (0–100) of breaching time series.",
			Placeholder: "e.g. 1",
		},
	}
}

// PromQLSpec holds the PromQL-condition inputs (used when conditionKind=promql).
type PromQLSpec struct {
	ConditionKind      string `mapstructure:"conditionKind"`
	Query              string `mapstructure:"promqlQuery"`
	Duration           string `mapstructure:"promqlDuration"`
	EvaluationInterval string `mapstructure:"promqlEvaluationInterval"`
}

// conditionKindField is the toggle between threshold and PromQL conditions.
func conditionKindField() configuration.Field {
	return configuration.Field{
		Name:        "conditionKind",
		Label:       "Condition type",
		Type:        configuration.FieldTypeSelect,
		Required:    false,
		Default:     conditionKindThreshold,
		Description: "Build the condition from a curated instance-metric threshold, or from a PromQL query (Google Managed Prometheus).",
		TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: conditionKindOptions}},
	}
}

// promqlConditionFields are the PromQL inputs, shown when conditionKind=promql.
func promqlConditionFields() []configuration.Field {
	promqlOnly := []configuration.VisibilityCondition{{Field: "conditionKind", Values: []string{conditionKindPromQL}}}
	return []configuration.Field{
		{
			Name:                 "promqlQuery",
			Label:                "PromQL query",
			Type:                 configuration.FieldTypeText,
			Required:             false,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "conditionKind", Values: []string{conditionKindPromQL}}},
			VisibilityConditions: promqlOnly,
			Description:          "The PromQL expression. The policy fires while the query returns one or more time series (e.g. an expression with a comparison).",
			Placeholder:          `rate(container_cpu_usage_seconds_total[5m]) > 0.8`,
		},
		{
			Name:                 "promqlDuration",
			Label:                "For (duration)",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Default:              "0s",
			VisibilityConditions: promqlOnly,
			Description:          "How long the query must keep returning results before firing (the PromQL FOR clause).",
			TypeOptions:          &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: durationOptions}},
		},
		{
			Name:                 "promqlEvaluationInterval",
			Label:                "Evaluation interval",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Default:              "60s",
			VisibilityConditions: promqlOnly,
			Description:          "How often Cloud Monitoring evaluates the query.",
			TypeOptions:          &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: evaluationIntervalOptions}},
		},
	}
}

// buildPolicyConditions assembles the conditions array for either a threshold or
// a PromQL policy, based on conditionKind (empty defaults to threshold).
func buildPolicyConditions(p PromQLSpec, thresholds []ConditionSpec) ([]any, error) {
	if p.ConditionKind == conditionKindPromQL {
		return buildPromQLConditions(p)
	}
	return buildConditions(thresholds)
}

func buildPromQLConditions(p PromQLSpec) ([]any, error) {
	query := strings.TrimSpace(p.Query)
	if query == "" {
		return nil, errors.New("promqlQuery is required for a PromQL condition")
	}
	if p.Duration != "" && !isValidDuration(p.Duration) {
		return nil, fmt.Errorf("invalid PromQL duration %q", p.Duration)
	}
	if p.EvaluationInterval != "" && !isValidEvaluationInterval(p.EvaluationInterval) {
		return nil, fmt.Errorf("invalid evaluation interval %q", p.EvaluationInterval)
	}
	return []any{buildPromQLCondition(query, p.Duration, p.EvaluationInterval)}, nil
}

func buildPromQLCondition(query, duration, evalInterval string) map[string]any {
	pql := map[string]any{"query": query}
	if duration != "" && duration != "0s" {
		pql["duration"] = duration
	}
	if evalInterval != "" {
		pql["evaluationInterval"] = evalInterval
	}
	return map[string]any{
		"displayName":                      promqlConditionDisplayName(query),
		"conditionPrometheusQueryLanguage": pql,
	}
}

func promqlConditionDisplayName(query string) string {
	const max = 60
	q := strings.Join(strings.Fields(query), " ")
	if len(q) > max {
		return q[:max-1] + "…"
	}
	return q
}

func validateCondition(c ConditionSpec) error {
	if _, ok := metricByType(c.MetricType); !ok {
		return fmt.Errorf("invalid or missing metricType %q", c.MetricType)
	}
	if !isValidComparison(c.Comparison) {
		return errors.New("invalid comparison")
	}
	if c.Threshold == nil {
		return errors.New("threshold is required")
	}
	if !isValidDuration(c.Duration) {
		return errors.New("invalid duration: must be one of 0s, 60s, 300s, 600s, 1800s")
	}
	if c.Aligner != "" && !isValidAligner(c.Aligner) {
		return fmt.Errorf("invalid aligner %q", c.Aligner)
	}
	if c.AlignmentPeriod != "" && !optionHasValue(alignmentPeriodOptions, c.AlignmentPeriod) {
		return fmt.Errorf("invalid alignmentPeriod %q", c.AlignmentPeriod)
	}
	if c.CrossSeriesReducer != "" && !isValidReducer(c.CrossSeriesReducer) {
		return fmt.Errorf("invalid crossSeriesReducer %q", c.CrossSeriesReducer)
	}
	if c.TriggerType != "" && c.TriggerType != triggerCount && c.TriggerType != triggerPercent {
		return fmt.Errorf("invalid triggerType %q", c.TriggerType)
	}
	return validateTriggerValue(c)
}

// validateTriggerValue guards the optional trigger value. A count must be a
// positive whole number (Cloud Monitoring rejects/ignores 0 or fractional
// counts); a percent must be within (0, 100].
func validateTriggerValue(c ConditionSpec) error {
	if c.TriggerValue == nil {
		return nil
	}
	v := *c.TriggerValue
	if c.TriggerType == triggerPercent {
		if v <= 0 || v > 100 {
			return errors.New("trigger percent must be greater than 0 and at most 100")
		}
		return nil
	}
	if v < 1 || v != float64(int64(v)) {
		return errors.New("trigger count must be a positive whole number")
	}
	return nil
}

// buildConditions validates and assembles the conditions array for the policy.
func buildConditions(specs []ConditionSpec) ([]any, error) {
	if len(specs) == 0 {
		return nil, errors.New("at least one condition is required")
	}
	if len(specs) > maxPolicyConditions {
		return nil, fmt.Errorf("at most %d conditions are allowed", maxPolicyConditions)
	}
	out := make([]any, 0, len(specs))
	for i, c := range specs {
		if err := validateCondition(c); err != nil {
			return nil, fmt.Errorf("condition %d: %w", i+1, err)
		}
		out = append(out, buildCondition(c))
	}
	return out, nil
}

func buildCondition(c ConditionSpec) map[string]any {
	aligner := c.Aligner
	if aligner == "" {
		if m, ok := metricByType(c.MetricType); ok {
			aligner = m.Aligner
		} else {
			aligner = "ALIGN_MEAN"
		}
	}
	period := c.AlignmentPeriod
	if period == "" {
		period = defaultAlignmentPeriod
	}

	aggregation := map[string]any{
		"alignmentPeriod":  period,
		"perSeriesAligner": aligner,
	}
	if c.CrossSeriesReducer != "" && c.CrossSeriesReducer != reduceNone {
		aggregation["crossSeriesReducer"] = c.CrossSeriesReducer
		if len(c.GroupByFields) > 0 {
			aggregation["groupByFields"] = c.GroupByFields
		}
	}

	trigger := map[string]any{}
	if c.TriggerType == triggerPercent {
		value := 100.0
		if c.TriggerValue != nil {
			value = *c.TriggerValue
		}
		trigger["percent"] = value
	} else {
		value := 1
		if c.TriggerValue != nil {
			value = int(*c.TriggerValue)
		}
		trigger["count"] = value
	}

	return map[string]any{
		"displayName": conditionDisplayName(c.MetricType, c.Comparison, *c.Threshold),
		"conditionThreshold": map[string]any{
			"filter":         instanceMetricFilter(c.MetricType),
			"comparison":     c.Comparison,
			"thresholdValue": *c.Threshold,
			"duration":       c.Duration,
			"trigger":        trigger,
			"aggregations":   []any{aggregation},
		},
	}
}

func conditionDisplayName(metricType, comparison string, threshold float64) string {
	return fmt.Sprintf("%s %s %g", lastSegment(metricType), comparisonSymbol(comparison), threshold)
}

func comparisonSymbol(comparison string) string {
	switch comparison {
	case comparisonGT:
		return ">"
	case comparisonLT:
		return "<"
	default:
		return comparison
	}
}
