package dash0

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseSpecification validates and parses a JSON object field used by upsert actions.
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

// parseCheckRuleSpecification parses and normalizes check rule specs for Dash0 API upserts.
func parseCheckRuleSpecification(specification, fieldName, scope string) (map[string]any, error) {
	parsed, err := parseSpecification(specification, fieldName, scope)
	if err != nil {
		return nil, err
	}

	normalized, err := normalizeCheckRuleSpecification(parsed, fieldName, scope)
	if err != nil {
		return nil, err
	}

	return normalized, nil
}

// normalizeCheckRuleSpecification converts Prometheus-style rule shapes to Dash0 check rule payloads.
func normalizeCheckRuleSpecification(specification map[string]any, fieldName, scope string) (map[string]any, error) {
	if specification == nil {
		return nil, fmt.Errorf("%s: %s is required", scope, fieldName)
	}

	if specValue, ok := specification["spec"]; ok {
		specMap, mapErr := asObjectMap(specValue, fmt.Sprintf("%s.spec", fieldName), scope)
		if mapErr == nil {
			if _, hasGroups := specMap["groups"]; hasGroups {
				specification = specMap
			}
		}
	}

	if groupsValue, ok := specification["groups"]; ok {
		return convertPrometheusGroupsToDash0CheckRule(groupsValue, fieldName, scope)
	}

	return validateDash0CheckRuleSpecification(specification, fieldName, scope)
}

// convertPrometheusGroupsToDash0CheckRule extracts exactly one alert rule and maps it to Dash0 payload.
func convertPrometheusGroupsToDash0CheckRule(groupsValue any, fieldName, scope string) (map[string]any, error) {
	groups, ok := groupsValue.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: %s.groups must be a JSON array", scope, fieldName)
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("%s: %s.groups cannot be empty", scope, fieldName)
	}

	var selectedRule map[string]any
	selectedGroupInterval := ""
	alertRuleCount := 0

	for groupIndex, groupValue := range groups {
		groupPath := fmt.Sprintf("%s.groups[%d]", fieldName, groupIndex)
		group, err := asObjectMap(groupValue, groupPath, scope)
		if err != nil {
			return nil, err
		}

		groupInterval, _ := optionalStringValue(group, "interval")

		rulesValue, hasRules := group["rules"]
		if !hasRules {
			continue
		}

		rules, ok := rulesValue.([]any)
		if !ok {
			return nil, fmt.Errorf("%s: %s.rules must be a JSON array", scope, groupPath)
		}

		for ruleIndex, ruleValue := range rules {
			rulePath := fmt.Sprintf("%s.rules[%d]", groupPath, ruleIndex)
			rule, err := asObjectMap(ruleValue, rulePath, scope)
			if err != nil {
				return nil, err
			}

			if recordValue, exists := optionalStringValue(rule, "record"); exists && recordValue != "" {
				continue
			}

			if _, hasExpression := firstNonEmptyMappedString(rule, "expression", "expr"); !hasExpression {
				continue
			}

			alertRuleCount++
			if alertRuleCount == 1 {
				selectedRule = rule
				selectedGroupInterval = groupInterval
			}
		}
	}

	if alertRuleCount == 0 {
		return nil, fmt.Errorf("%s: %s must contain one alert rule with expr/expression", scope, fieldName)
	}

	if alertRuleCount > 1 {
		return nil, fmt.Errorf("%s: %s must contain exactly one alert rule; found %d", scope, fieldName, alertRuleCount)
	}

	checkRule := map[string]any{}
	if ruleName, ok := firstNonEmptyMappedString(selectedRule, "name", "alert"); ok {
		checkRule["name"] = ruleName
	}

	expression, ok := firstNonEmptyMappedString(selectedRule, "expression", "expr")
	if !ok {
		return nil, fmt.Errorf("%s: %s must include a non-empty expression", scope, fieldName)
	}
	checkRule["expression"] = expression

	if intervalValue, ok := firstNonEmptyMappedString(selectedRule, "interval"); ok {
		checkRule["interval"] = intervalValue
	} else if selectedGroupInterval != "" {
		checkRule["interval"] = selectedGroupInterval
	}

	if forValue, ok := firstNonEmptyMappedString(selectedRule, "for"); ok {
		checkRule["for"] = forValue
	}

	if keepFiringValue, ok := firstNonEmptyMappedString(selectedRule, "keepFiringFor", "keep_firing_for"); ok {
		checkRule["keepFiringFor"] = keepFiringValue
	}

	annotations, err := extractStringMapField(selectedRule, "annotations", fieldName, scope)
	if err != nil {
		return nil, err
	}
	if len(annotations) > 0 {
		checkRule["annotations"] = annotations
	}

	labels, err := extractStringMapField(selectedRule, "labels", fieldName, scope)
	if err != nil {
		return nil, err
	}
	if len(labels) > 0 {
		checkRule["labels"] = labels
	}

	return validateDash0CheckRuleSpecification(checkRule, fieldName, scope)
}

// validateDash0CheckRuleSpecification enforces required Dash0 check rule fields.
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

	annotations, err := extractStringMapField(specification, "annotations", fieldName, scope)
	if err != nil {
		return nil, err
	}
	if len(annotations) > 0 {
		specification["annotations"] = annotations
	}

	labels, err := extractStringMapField(specification, "labels", fieldName, scope)
	if err != nil {
		return nil, err
	}
	if len(labels) > 0 {
		specification["labels"] = labels
	}

	return specification, nil
}

// requireNonEmptyValue trims and validates required string configuration values.
func requireNonEmptyValue(value, fieldName, scope string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s: %s is required", scope, fieldName)
	}

	return trimmed, nil
}

// asObjectMap asserts that a value is a JSON object represented as map[string]any.
func asObjectMap(value any, fieldPath, scope string) (map[string]any, error) {
	decoded, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: %s must be a JSON object", scope, fieldPath)
	}
	return decoded, nil
}

// optionalStringValue extracts a trimmed string value from a map field.
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

// firstNonEmptyMappedString returns the first available non-empty string among candidate keys.
func firstNonEmptyMappedString(values map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := optionalStringValue(values, key); ok {
			return value, true
		}
	}
	return "", false
}

// extractStringMapField normalizes map[string]string fields from decoded JSON objects.
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

// validateSyntheticCheckSpecification validates the minimum shape required by Dash0 synthetic checks.
func validateSyntheticCheckSpecification(specification map[string]any, fieldName, scope string) error {
	kindValue, ok := specification["kind"]
	if !ok {
		return fmt.Errorf("%s: %s.kind is required (expected \"Dash0SyntheticCheck\")", scope, fieldName)
	}

	kind, ok := kindValue.(string)
	if !ok {
		return fmt.Errorf("%s: %s.kind must be a string", scope, fieldName)
	}

	trimmedKind := strings.TrimSpace(kind)
	if trimmedKind == "" {
		return fmt.Errorf("%s: %s.kind is required (expected \"Dash0SyntheticCheck\")", scope, fieldName)
	}

	if !strings.EqualFold(trimmedKind, "Dash0SyntheticCheck") {
		return fmt.Errorf("%s: %s.kind must be \"Dash0SyntheticCheck\"", scope, fieldName)
	}

	specification["kind"] = "Dash0SyntheticCheck"

	specValue, ok := specification["spec"]
	if !ok {
		return fmt.Errorf("%s: %s must include object field spec", scope, fieldName)
	}

	specMap, ok := specValue.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: %s.spec must be a JSON object", scope, fieldName)
	}

	pluginValue, ok := specMap["plugin"]
	if !ok {
		return fmt.Errorf("%s: %s.spec.plugin is required", scope, fieldName)
	}

	pluginMap, ok := pluginValue.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: %s.spec.plugin must be a JSON object", scope, fieldName)
	}

	pluginKindValue, ok := pluginMap["kind"]
	if !ok {
		return fmt.Errorf("%s: %s.spec.plugin.kind is required (for example: \"http\")", scope, fieldName)
	}

	pluginKind, ok := pluginKindValue.(string)
	if !ok {
		return fmt.Errorf("%s: %s.spec.plugin.kind must be a string", scope, fieldName)
	}

	trimmedPluginKind := strings.TrimSpace(pluginKind)
	if trimmedPluginKind == "" {
		return fmt.Errorf("%s: %s.spec.plugin.kind is required (for example: \"http\")", scope, fieldName)
	}

	pluginMap["kind"] = strings.ToLower(trimmedPluginKind)

	return nil
}
