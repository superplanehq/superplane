package dash0

import (
	"fmt"
	"strings"
)

// requireNonEmptyValue trims and validates required string configuration values.
func requireNonEmptyValue(value, fieldName, scope string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s: %s is required", scope, fieldName)
	}

	return trimmed, nil
}

// buildSyntheticCheckSpecificationFromConfiguration validates and builds a synthetic check specification map.
func buildSyntheticCheckSpecificationFromConfiguration(config UpsertSyntheticCheckConfiguration, scope string) (map[string]any, error) {
	name, err := requireNonEmptyValue(config.Name, "name", scope)
	if err != nil {
		return nil, err
	}

	method, err := requireNonEmptyValue(config.Method, "method", scope)
	if err != nil {
		return nil, err
	}

	requestURL, err := requireNonEmptyValue(config.URL, "url", scope)
	if err != nil {
		return nil, err
	}

	pluginKind := strings.TrimSpace(config.PluginKind)
	if pluginKind == "" {
		pluginKind = "http"
	}

	request := map[string]any{
		"method": strings.ToLower(method),
		"url":    requestURL,
	}

	headers, err := normalizeSyntheticCheckFields(config.Headers, "headers", scope)
	if err != nil {
		return nil, err
	}
	if len(headers) > 0 {
		request["headers"] = headers
	}

	requestBody := strings.TrimSpace(config.RequestBody)
	if requestBody != "" {
		request["body"] = requestBody
	}

	specification := map[string]any{
		"kind": "Dash0SyntheticCheck",
		"metadata": map[string]any{
			"name": name,
		},
		"spec": map[string]any{
			"enabled": config.Enabled,
			"plugin": map[string]any{
				"kind": strings.ToLower(pluginKind),
				"spec": map[string]any{
					"request": request,
				},
			},
		},
	}

	if err := validateSyntheticCheckSpecification(specification, "spec", scope); err != nil {
		return nil, err
	}

	return specification, nil
}

// normalizeSyntheticCheckFields converts list-based key/value entries into a request map.
func normalizeSyntheticCheckFields(fields []SyntheticCheckField, fieldName, scope string) (map[string]string, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	normalized := make(map[string]string, len(fields))
	for index, field := range fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			return nil, fmt.Errorf("%s: %s[%d].key is required", scope, fieldName, index)
		}

		normalized[key] = strings.TrimSpace(field.Value)
	}

	return normalized, nil
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
