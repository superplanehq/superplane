package components

import (
	"fmt"
	"regexp"
	"strings"
)

var expressionRegex = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

type ConfigurationBuilder struct{}

func (b *ConfigurationBuilder) Build(c map[string]any, fields map[string]any) (map[string]any, error) {
	resolved, err := b.resolve(c, fields)
	if err != nil {
		return nil, err
	}

	return resolved, nil
}

func (b *ConfigurationBuilder) resolve(c map[string]any, fields map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(c))

	for k, v := range c {
		resolved, err := b.resolveValue(v, fields)
		if err != nil {
			return nil, fmt.Errorf("error resolving field %s: %w", k, err)
		}
		result[k] = resolved
	}

	return result, nil
}

func (b *ConfigurationBuilder) resolveValue(value any, fields map[string]any) (any, error) {
	switch v := value.(type) {
	case string:
		return b.ResolveExpression(v, fields)

	case map[string]any:
		return b.resolve(v, fields)

	case map[string]string:
		anyMap := make(map[string]any, len(v))
		for key, value := range v {
			anyMap[key] = value
		}

		return b.resolve(anyMap, fields)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			resolved, err := b.resolveValue(item, fields)
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

func (b *ConfigurationBuilder) ResolveExpression(expression string, fields map[string]any) (any, error) {
	if !expressionRegex.MatchString(expression) {
		return expression, nil
	}

	var err error

	result := expressionRegex.ReplaceAllStringFunc(expression, func(match string) string {
		matches := expressionRegex.FindStringSubmatch(match)
		if len(matches) != 2 {
			return match
		}

		value, e := b.resolveExpression(matches[1], fields)
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

func (b *ConfigurationBuilder) resolveExpression(expression string, fields map[string]any) (any, error) {
	expression = strings.TrimSpace(expression)

	// Handle direct input access: config.CONFIG_FIELD_NAME
	if strings.HasPrefix(expression, "config.") {
		key := strings.TrimSpace(strings.TrimPrefix(expression, "config."))
		if key == "" {
			return nil, fmt.Errorf("empty config key")
		}
		if value, exists := fields[key]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("input %s not found", key)
	}

	return nil, fmt.Errorf("invalid expression format")
}
