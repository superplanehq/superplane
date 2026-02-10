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

// requireNonEmptyValue trims and validates required string configuration values.
func requireNonEmptyValue(value, fieldName, scope string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s: %s is required", scope, fieldName)
	}

	return trimmed, nil
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
