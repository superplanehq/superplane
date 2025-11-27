package http

import (
	"fmt"

	"github.com/expr-lang/expr"
)

// ProcessPayload recursively evaluates string values against ctx.Data
func ProcessPayload(payload any, data any) (any, error) {
	return processValue(payload, data)
}

func processValue(value any, data any) (any, error) {
	switch v := value.(type) {
	case string:
		return evaluateStringExpression(v, data)
	case map[string]any:
		return processMap(v, data)
	case []any:
		return processList(v, data)
	default:
		// Numbers, booleans, nil pass through unchanged
		return v, nil
	}
}

func processMap(m map[string]any, data any) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range m {
		processed, err := processValue(v, data)
		if err != nil {
			return nil, fmt.Errorf("failed to process field '%s': %w", k, err)
		}
		result[k] = processed
	}
	return result, nil
}

func processList(lst []any, data any) ([]any, error) {
	result := make([]any, len(lst))
	for i, v := range lst {
		processed, err := processValue(v, data)
		if err != nil {
			return nil, fmt.Errorf("failed to process list item [%d]: %w", i, err)
		}
		result[i] = processed
	}
	return result, nil
}

func evaluateStringExpression(exprStr string, data any) (any, error) {
	env := map[string]any{
		"$": data,
	}

	program, err := expr.Compile(exprStr, expr.Env(env))
	if err != nil {
		// Graceful degradation: return original string
		return exprStr, nil
	}

	result, err := expr.Run(program, env)
	if err != nil {
		// Graceful degradation: return original string
		return exprStr, nil
	}

	return result, nil
}

// ProcessHeaders evaluates string values in headers against ctx.Data
func ProcessHeaders(headers []Header, data any) ([]Header, error) {
	result := make([]Header, len(headers))
	for i, header := range headers {
		// Evaluate header name
		name, err := evaluateStringExpression(header.Name, data)
		if err != nil {
			return nil, fmt.Errorf("failed to process header name at index %d: %w", i, err)
		}
		nameStr, ok := name.(string)
		if !ok {
			return nil, fmt.Errorf("header name at index %d must evaluate to string, got %T", i, name)
		}

		// Evaluate header value
		value, err := evaluateStringExpression(header.Value, data)
		if err != nil {
			return nil, fmt.Errorf("failed to process header value at index %d: %w", i, err)
		}
		valueStr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("header value at index %d must evaluate to string, got %T", i, value)
		}

		result[i] = Header{
			Name:  nameStr,
			Value: valueStr,
		}
	}
	return result, nil
}
