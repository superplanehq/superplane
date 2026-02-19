package dash0

import (
	"fmt"
	"strings"
)

// CheckRuleKeyValue represents a single key/value entry for labels or annotations.
type CheckRuleKeyValue struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

// UpsertCheckRuleConfiguration contains user input for the check rule upsert actions.
type UpsertCheckRuleConfiguration struct {
	OriginOrID    string              `json:"originOrId" mapstructure:"originOrId"`
	Name          string              `json:"name" mapstructure:"name"`
	Expression    string              `json:"expression" mapstructure:"expression"`
	For           string              `json:"for" mapstructure:"for"`
	Interval      string              `json:"interval" mapstructure:"interval"`
	KeepFiringFor string              `json:"keepFiringFor" mapstructure:"keepFiringFor"`
	Labels        []CheckRuleKeyValue `json:"labels" mapstructure:"labels"`
	Annotations   []CheckRuleKeyValue `json:"annotations" mapstructure:"annotations"`
}

// buildCheckRuleSpecification validates and normalizes the check rule payload.
func buildCheckRuleSpecification(config UpsertCheckRuleConfiguration, scope string) (map[string]any, error) {
	ruleName, err := requireNonEmptyValue(config.Name, "name", scope)
	if err != nil {
		return nil, err
	}

	expression, err := requireNonEmptyValue(config.Expression, "expression", scope)
	if err != nil {
		return nil, err
	}

	specification := map[string]any{
		"name":       ruleName,
		"expression": expression,
	}

	addOptionalStringField(specification, "for", config.For)
	addOptionalStringField(specification, "interval", config.Interval)
	addOptionalStringField(specification, "keepFiringFor", config.KeepFiringFor)

	labels, err := normalizeKeyValuePairs(config.Labels, "labels", scope)
	if err != nil {
		return nil, err
	}
	if len(labels) > 0 {
		specification["labels"] = labels
	}

	annotations, err := normalizeKeyValuePairs(config.Annotations, "annotations", scope)
	if err != nil {
		return nil, err
	}
	if len(annotations) > 0 {
		specification["annotations"] = annotations
	}

	return specification, nil
}

// addOptionalStringField adds a field when the provided value is non-empty.
func addOptionalStringField(target map[string]any, fieldName, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}

	target[fieldName] = trimmed
}

// normalizeKeyValuePairs validates and normalizes list-based key/value entries.
func normalizeKeyValuePairs(pairs []CheckRuleKeyValue, fieldName, scope string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	normalized := make(map[string]string, len(pairs))
	for index, pair := range pairs {
		key := strings.TrimSpace(pair.Key)
		if key == "" {
			return nil, fmt.Errorf("%s: %s[%d].key is required", scope, fieldName, index)
		}

		if _, exists := normalized[key]; exists {
			return nil, fmt.Errorf("%s: %s[%d].key %q is duplicated", scope, fieldName, index, key)
		}

		normalized[key] = strings.TrimSpace(pair.Value)
	}

	return normalized, nil
}

// requireNonEmptyValue trims and validates required string values.
func requireNonEmptyValue(value, fieldName, scope string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s: %s is required", scope, fieldName)
	}

	return trimmed, nil
}
