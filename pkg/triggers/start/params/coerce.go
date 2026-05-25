package params

import (
	"fmt"
	"slices"
)

// coerceParamValue coerces a runParams value for a param() leaf.
// Coercion rules come from the param() definition: declared type, and for select
// params, membership in the allowed options. Used by ValidateRunParams and ApplyParams
// for paths backed by param(...) placeholders rather than static template values.
func coerceParamValue(def Definition, value any) (any, error) {
	switch def.Type {
	case ParamTypeString:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return s, nil
	case ParamTypeNumber:
		switch n := value.(type) {
		case float64:
			return n, nil
		case float32:
			return float64(n), nil
		case int:
			return float64(n), nil
		case int64:
			return float64(n), nil
		case int32:
			return float64(n), nil
		default:
			return nil, fmt.Errorf("expected number, got %T", value)
		}
	case ParamTypeBoolean:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected boolean, got %T", value)
		}
		return b, nil
	case ParamTypeSelect:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		if !slices.Contains(def.Values, s) {
			return nil, fmt.Errorf("value %q is not one of: %s", s, stringsJoin(def.Values, ", "))
		}
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", def.Type)
	}
}

// coerceStaticValue coerces a runParams override for a static (non-param) template leaf.
// The expected type is inferred from the existing leaf value in the template—for example,
// overriding "message": "hello" with a new string, or "count": 1 with a new number.
// Used by ApplyParams when a runParams path does not match a param() definition.
func coerceStaticValue(existing any, value any) (any, error) {
	switch existing.(type) {
	case string:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return s, nil
	case float64:
		switch n := value.(type) {
		case float64:
			return n, nil
		case float32:
			return float64(n), nil
		case int:
			return float64(n), nil
		case int64:
			return float64(n), nil
		default:
			return nil, fmt.Errorf("expected number, got %T", value)
		}
	case bool:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected boolean, got %T", value)
		}
		return b, nil
	default:
		return value, nil
	}
}

func stringsJoin(values []string, sep string) string {
	if len(values) == 0 {
		return ""
	}
	out := values[0]
	for _, v := range values[1:] {
		out += sep + v
	}
	return out
}
