package executors

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/models"
)

type SpecBuilder struct{}

func (b *SpecBuilder) Build(spec models.ExecutorSpec, inputs map[string]any, secrets map[string]string) (*models.ExecutorSpec, error) {
	m, err := b.specToMap(spec)
	if err != nil {
		return nil, err
	}

	resolved, err := b.resolveMap(m, inputs, secrets)
	if err != nil {
		return nil, err
	}

	return b.mapToSpec(resolved)
}

func (b *SpecBuilder) specToMap(spec models.ExecutorSpec) (map[string]any, error) {
	var result map[string]any

	config := &mapstructure.DecoderConfig{TagName: "json", Result: &result}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(spec); err != nil {
		return nil, fmt.Errorf("failed to decode spec: %w", err)
	}

	return result, nil
}

func (b *SpecBuilder) mapToSpec(data map[string]any) (*models.ExecutorSpec, error) {
	var spec models.ExecutorSpec

	config := &mapstructure.DecoderConfig{TagName: "json", Result: &spec}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(data); err != nil {
		return nil, fmt.Errorf("failed to decode spec: %w", err)
	}

	return &spec, nil
}

func (b *SpecBuilder) resolveMap(m map[string]any, inputs map[string]any, secrets map[string]string) (map[string]any, error) {
	result := make(map[string]any, len(m))

	for k, v := range m {
		resolved, err := b.resolveValue(v, inputs, secrets)
		if err != nil {
			return nil, fmt.Errorf("error resolving field %s: %w", k, err)
		}
		result[k] = resolved
	}

	return result, nil
}

func (b *SpecBuilder) resolveValue(value any, inputs map[string]any, secrets map[string]string) (any, error) {
	switch v := value.(type) {
	case string:
		return b.ResolveExpression(v, inputs, secrets)

	case map[string]any:
		return b.resolveMap(v, inputs, secrets)

	case map[string]string:
		anyMap := make(map[string]any, len(v))
		for key, value := range v {
			anyMap[key] = value
		}

		return b.resolveMap(anyMap, inputs, secrets)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			resolved, err := b.resolveValue(item, inputs, secrets)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil

	default:
		return v, nil
	}
}

func (b *SpecBuilder) ResolveExpression(expression string, inputs map[string]any, secrets map[string]string) (any, error) {
	if !expressionRegex.MatchString(expression) {
		return expression, nil
	}

	var err error

	result := expressionRegex.ReplaceAllStringFunc(expression, func(match string) string {
		matches := expressionRegex.FindStringSubmatch(match)
		if len(matches) != 2 {
			return match
		}

		value, e := b.resolveExpression(matches[1], inputs, secrets)
		if e != nil {
			err = e
			return ""
		}

		return fmt.Sprintf("%v", value)
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (b *SpecBuilder) resolveExpression(expression string, inputs map[string]any, secrets map[string]string) (any, error) {
	expression = strings.TrimSpace(expression)

	// Handle direct secret access: secrets.SECRET_NAME
	if strings.HasPrefix(expression, "secrets.") {
		key := strings.TrimSpace(strings.TrimPrefix(expression, "secrets."))
		if key == "" {
			return nil, fmt.Errorf("empty secret key")
		}
		if value, exists := secrets[key]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("secret %s not found", key)
	}

	// Handle direct input access: inputs.INPUT_NAME
	if strings.HasPrefix(expression, "inputs.") {
		key := strings.TrimSpace(strings.TrimPrefix(expression, "inputs."))
		if key == "" {
			return nil, fmt.Errorf("empty input key")
		}
		if value, exists := inputs[key]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("input %s not found", key)
	}

	return nil, fmt.Errorf("invalid expression format")
}
