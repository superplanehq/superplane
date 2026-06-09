package loop

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/superplanehq/superplane/pkg/core"
)

type iteration struct {
	Index int
	Value any
}

func (s Spec) resolveIterations(expressions core.ExpressionContext) ([]iteration, error) {
	switch s.Mode {
	case ModeCollection:
		return resolveCollectionIterations(s.CollectionExpression, expressions)
	case ModeCount:
		return resolveCountIterations(s.CountExpression, expressions)
	case ModeRange:
		return resolveRangeIterations(s.StartExpression, s.EndExpression, s.StepExpression, expressions)
	default:
		return nil, fmt.Errorf("unsupported loop mode %q", s.Mode)
	}
}

func resolveCollectionIterations(expression string, expressions core.ExpressionContext) ([]iteration, error) {
	result, err := expressions.Run(expression)
	if err != nil {
		return nil, fmt.Errorf("collection expression evaluation failed: %w", err)
	}

	items, err := coerceToList(result)
	if err != nil {
		return nil, fmt.Errorf("collection expression must evaluate to a list: %w", err)
	}

	iterations := make([]iteration, len(items))
	for i, item := range items {
		iterations[i] = iteration{Index: i, Value: item}
	}
	return iterations, nil
}

func resolveCountIterations(expression string, expressions core.ExpressionContext) ([]iteration, error) {
	result, err := expressions.Run(expression)
	if err != nil {
		return nil, fmt.Errorf("count expression evaluation failed: %w", err)
	}

	count, err := parseNonNegativeInt(result, "count expression")
	if err != nil {
		return nil, err
	}

	iterations := make([]iteration, count)
	for i := range iterations {
		iterations[i] = iteration{Index: i, Value: i}
	}
	return iterations, nil
}

func resolveRangeIterations(startExpression, endExpression, stepExpression string, expressions core.ExpressionContext) ([]iteration, error) {
	startResult, err := expressions.Run(startExpression)
	if err != nil {
		return nil, fmt.Errorf("start expression evaluation failed: %w", err)
	}
	endResult, err := expressions.Run(endExpression)
	if err != nil {
		return nil, fmt.Errorf("end expression evaluation failed: %w", err)
	}

	start, err := parseNumber(startResult, "start expression")
	if err != nil {
		return nil, err
	}
	end, err := parseNumber(endResult, "end expression")
	if err != nil {
		return nil, err
	}

	step := 1.0
	if stepExpression != "" {
		stepResult, stepErr := expressions.Run(stepExpression)
		if stepErr != nil {
			return nil, fmt.Errorf("step expression evaluation failed: %w", stepErr)
		}
		step, err = parseNumber(stepResult, "step expression")
		if err != nil {
			return nil, err
		}
	}

	if step == 0 {
		return nil, fmt.Errorf("step expression must not evaluate to zero")
	}

	values, err := buildRangeValues(start, end, step)
	if err != nil {
		return nil, err
	}

	iterations := make([]iteration, len(values))
	for i, value := range values {
		iterations[i] = iteration{Index: i, Value: value}
	}
	return iterations, nil
}

func buildRangeValues(start, end, step float64) ([]any, error) {
	if step > 0 && start > end {
		return []any{}, nil
	}
	if step < 0 && start < end {
		return []any{}, nil
	}

	maxIterations := core.MaxEmitCount + 1
	values := make([]any, 0)

	for current := start; (step > 0 && current <= end) || (step < 0 && current >= end); current += step {
		if len(values) >= maxIterations {
			return nil, fmt.Errorf("range produces more than %d values", core.MaxEmitCount)
		}
		values = append(values, normalizeRangeValue(current))
	}

	return values, nil
}

func normalizeRangeValue(value float64) any {
	if math.Trunc(value) == value {
		return int(value)
	}
	return value
}

func coerceToList(value any) ([]any, error) {
	switch v := value.(type) {
	case nil:
		return []any{}, nil
	case []any:
		return v, nil
	case []map[string]any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []string:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out, nil
	}

	return nil, fmt.Errorf("got %T", value)
}

func parseNonNegativeInt(value any, field string) (int, error) {
	parsed, err := parseNumber(value, field)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, fmt.Errorf("%s must evaluate to a non-negative integer, got %v", field, value)
	}
	if math.Trunc(parsed) != parsed {
		return 0, fmt.Errorf("%s must evaluate to a whole number, got %v", field, value)
	}
	return int(parsed), nil
}

func parseNumber(value any, field string) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must evaluate to a number, got %q", field, v)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%s must evaluate to a number, got %T", field, value)
	}
}
