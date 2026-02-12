package dash0

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CheckRuleKeyValue represents a single key/value entry for labels or annotations.
type CheckRuleKeyValue struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

// UpsertCheckRuleConfiguration contains user input for the check rule upsert actions.
//
// Spec is kept for backward compatibility with existing saved workflows that still
// provide raw JSON. New flows should use the structured form fields.
type UpsertCheckRuleConfiguration struct {
	OriginOrID    string              `json:"originOrId" mapstructure:"originOrId"`
	Name          string              `json:"name" mapstructure:"name"`
	Expression    string              `json:"expression" mapstructure:"expression"`
	For           string              `json:"for" mapstructure:"for"`
	Interval      string              `json:"interval" mapstructure:"interval"`
	KeepFiringFor string              `json:"keepFiringFor" mapstructure:"keepFiringFor"`
	Labels        []CheckRuleKeyValue `json:"labels" mapstructure:"labels"`
	Annotations   []CheckRuleKeyValue `json:"annotations" mapstructure:"annotations"`
	Spec          string              `json:"spec" mapstructure:"spec"`
}

// buildCheckRuleSpecification validates and normalizes the check rule payload.
func buildCheckRuleSpecification(config UpsertCheckRuleConfiguration, scope string) (map[string]any, error) {
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

		normalized[key] = strings.TrimSpace(pair.Value)
	}

	if len(normalized) == 0 {
		return nil, nil
	}

	return normalized, nil
}

// parseCheckRuleSpecification parses and validates backward-compatible JSON specs.
func parseCheckRuleSpecification(specification, fieldName, scope string) (map[string]any, error) {
	parsed, err := parseSpecification(specification, fieldName, scope)
	if err != nil {
		return nil, err
	}

	return validateDash0CheckRuleSpecification(parsed, fieldName, scope)
}

// parseSpecification parses a JSON object or a single-item JSON array containing one object.
func parseSpecification(specification, fieldName, scope string) (map[string]any, error) {
	trimmed := strings.TrimSpace(specification)
	if trimmed == "" {
		return nil, fmt.Errorf("%s: %s is required", scope, fieldName)
	}

	var payload map[string]any
	objectErr := json.Unmarshal([]byte(trimmed), &payload)
	if objectErr == nil {
		if len(payload) == 0 {
			return nil, fmt.Errorf("%s: %s cannot be an empty JSON object", scope, fieldName)
		}

		return payload, nil
	}

	var payloadArray []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payloadArray); err == nil {
		if len(payloadArray) == 0 {
			return nil, fmt.Errorf("%s: %s cannot be an empty JSON array", scope, fieldName)
		}
		if len(payloadArray) > 1 {
			return nil, fmt.Errorf("%s: %s must be a JSON object or a single-item JSON array", scope, fieldName)
		}
		if len(payloadArray[0]) == 0 {
			return nil, fmt.Errorf("%s: %s cannot contain an empty JSON object", scope, fieldName)
		}

		return payloadArray[0], nil
	}

	return nil, fmt.Errorf("%s: parse %s as JSON object: %w", scope, fieldName, objectErr)
}

// validateDash0CheckRuleSpecification enforces required fields and normalizes aliases.
func validateDash0CheckRuleSpecification(specification map[string]any, fieldName, scope string) (map[string]any, error) {
	ruleName, ok := firstNonEmptyMappedString(specification, "name", "alert")
	if !ok {
		return nil, fmt.Errorf("%s: %s.name is required", scope, fieldName)
	}
	specification["name"] = ruleName
	delete(specification, "alert")

	expression, ok := firstNonEmptyMappedString(specification, "expression", "expr")
	if !ok {
		return nil, fmt.Errorf("%s: %s.expression is required", scope, fieldName)
	}
	specification["expression"] = expression
	delete(specification, "expr")

	if keepFiringValue, ok := firstNonEmptyMappedString(specification, "keepFiringFor", "keep_firing_for"); ok {
		specification["keepFiringFor"] = keepFiringValue
	}
	delete(specification, "keep_firing_for")

	if intervalValue, ok := firstNonEmptyMappedString(specification, "interval"); ok {
		specification["interval"] = intervalValue
	}

	if forValue, ok := firstNonEmptyMappedString(specification, "for"); ok {
		specification["for"] = forValue
	}

	labels, err := extractStringMapField(specification, "labels", fieldName, scope)
	if err != nil {
		return nil, err
	}
	if len(labels) > 0 {
		specification["labels"] = labels
	}

	annotations, err := extractStringMapField(specification, "annotations", fieldName, scope)
	if err != nil {
		return nil, err
	}
	if len(annotations) > 0 {
		specification["annotations"] = annotations
	}

	return specification, nil
}

// requireNonEmptyValue trims and validates required string values.
func requireNonEmptyValue(value, fieldName, scope string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s: %s is required", scope, fieldName)
	}

	return trimmed, nil
}

// optionalStringValue returns a trimmed non-empty string value from the provided map key.
func optionalStringValue(values map[string]any, key string) (string, bool) {
	rawValue, ok := values[key]
	if !ok || rawValue == nil {
		return "", false
	}

	stringValue, ok := rawValue.(string)
	if !ok {
		return "", false
	}

	trimmed := strings.TrimSpace(stringValue)
	if trimmed == "" {
		return "", false
	}

	return trimmed, true
}

// firstNonEmptyMappedString returns the first non-empty string value for the provided keys.
func firstNonEmptyMappedString(values map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := optionalStringValue(values, key); ok {
			return value, true
		}
	}

	return "", false
}

// extractStringMapField validates and normalizes object fields with string values only.
func extractStringMapField(values map[string]any, key, fieldName, scope string) (map[string]string, error) {
	rawValue, ok := values[key]
	if !ok || rawValue == nil {
		return nil, nil
	}

	fieldPath := fmt.Sprintf("%s.%s", fieldName, key)
	switch typed := rawValue.(type) {
	case map[string]string:
		return typed, nil
	case map[string]any:
		normalized := make(map[string]string, len(typed))
		for mapKey, mapValue := range typed {
			stringValue, isString := mapValue.(string)
			if !isString {
				return nil, fmt.Errorf("%s: %s.%s must be a string", scope, fieldPath, mapKey)
			}

			normalized[mapKey] = stringValue
		}

		return normalized, nil
	default:
		return nil, fmt.Errorf("%s: %s must be a JSON object of string values", scope, fieldPath)
	}
}
