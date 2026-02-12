package dash0

import (
	"fmt"
	"strings"
)

// buildSyntheticCheckSpecificationFromConfiguration validates and builds a synthetic check specification map.
func buildSyntheticCheckSpecificationFromConfiguration(config UpsertSyntheticCheckConfiguration, scope string) (map[string]any, error) {
	if strings.TrimSpace(config.Spec) != "" {
		specification, err := parseSpecification(config.Spec, "spec", scope)
		if err != nil {
			return nil, err
		}

		if err := validateSyntheticCheckSpecification(specification, "spec", scope); err != nil {
			return nil, err
		}

		return specification, nil
	}

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
