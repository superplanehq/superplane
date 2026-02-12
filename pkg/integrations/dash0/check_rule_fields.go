package dash0

import (
	"fmt"
	"strings"
)

// buildCheckRuleSpecificationFromConfiguration validates and normalizes check rule configuration fields.
func buildCheckRuleSpecificationFromConfiguration(config UpsertCheckRuleConfiguration, scope string) (map[string]any, error) {
	if strings.TrimSpace(config.Spec) != "" {
		return parseCheckRuleSpecification(config.Spec, "spec", scope)
	}

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

	addOptionalCheckRuleStringField(specification, "for", config.For)
	addOptionalCheckRuleStringField(specification, "interval", config.Interval)
	addOptionalCheckRuleStringField(specification, "keepFiringFor", config.KeepFiringFor)

	labels, err := normalizeCheckRuleKeyValuePairs(config.Labels, "labels", scope)
	if err != nil {
		return nil, err
	}
	if len(labels) > 0 {
		specification["labels"] = labels
	}

	annotations, err := normalizeCheckRuleKeyValuePairs(config.Annotations, "annotations", scope)
	if err != nil {
		return nil, err
	}
	if len(annotations) > 0 {
		specification["annotations"] = annotations
	}

	return specification, nil
}

// addOptionalCheckRuleStringField adds a trimmed string field when the value is non-empty.
func addOptionalCheckRuleStringField(target map[string]any, fieldName, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}

	target[fieldName] = trimmed
}

// normalizeCheckRuleKeyValuePairs converts list-based key/value entries into a map.
func normalizeCheckRuleKeyValuePairs(pairs []CheckRuleKeyValue, fieldName, scope string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	normalized := make(map[string]string, len(pairs))
	for index, pair := range pairs {
		key := strings.TrimSpace(pair.Key)
		if key == "" {
			return nil, fmt.Errorf("%s: %s[%d].key is required", scope, fieldName, index)
		}

		normalized[key] = strings.TrimSpace(pair.Value)
	}

	if len(normalized) == 0 {
		return nil, nil
	}

	return normalized, nil
}
