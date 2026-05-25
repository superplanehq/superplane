package params

import (
	"fmt"
)

// ApplyParams merges run-time parameter values into a template payload.
// param() leaves are substituted; static leaves are overridden
// when a matching path exists in runParams.
func ApplyParams(template map[string]any, runParams map[string]any) (map[string]any, error) {
	if runParams == nil {
		runParams = map[string]any{}
	}

	// Discover param() leaves and validate runParams before mutating the template.
	defs, err := ParseParams(template)
	if err != nil {
		return nil, err
	}
	if err := ValidateRunParams(defs, runParams); err != nil {
		return nil, err
	}
	if err := checkForUnknownParams(template, runParams); err != nil {
		return nil, err
	}

	out := deepCopyMap(template)

	// Union of paths to apply:
	// - every param() path (for defaults),
	// - any runParams static override.
	defByPath := make(map[string]Definition, len(defs))
	paths := make(map[string]struct{})
	for _, def := range defs {
		defByPath[def.Path] = def
		paths[def.Path] = struct{}{}
	}
	for path := range runParams {
		paths[path] = struct{}{}
	}

	for path := range paths {
		if def, ok := defByPath[path]; ok {
			// Replace param(...) with a run-time value or the definition default.
			value, err := resolveParamValue(def, runParams)
			if err != nil {
				return nil, err
			}
			if err := setValueAtPath(out, path, value); err != nil {
				return nil, err
			}
			continue
		}

		// Static leaf override: path is in runParams but not a param() slot.
		value, ok := runParams[path]
		if !ok {
			continue
		}
		existing, ok, err := getValueAtPath(out, path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if !ok {
			return nil, fmt.Errorf("unknown parameter %q", path)
		}
		coerced, err := coerceStaticValue(existing, value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if err := setValueAtPath(out, path, coerced); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// ValidateRunParams checks run-time values for param() definitions.
// Keys in runParams that are not param paths are ignored
// (static overrides are validated in ApplyParams).
func ValidateRunParams(defs []Definition, runParams map[string]any) error {
	defByPath := make(map[string]Definition, len(defs))
	for _, def := range defs {
		defByPath[def.Path] = def
	}

	// Check types of run-time values against param() definitions.
	for path, value := range runParams {
		def, ok := defByPath[path]
		if !ok {
			// Static leaf override: path is in runParams but not a param() slot.
			continue
		}
		if _, err := coerceParamValue(def, value); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}

	// Check for missing required parameters.
	for _, def := range defs {
		if _, provided := runParams[def.Path]; provided {
			continue
		}
		if def.Required && def.Default == nil {
			return fmt.Errorf("missing required parameter %q", def.Path)
		}
	}

	return nil
}

// Every runParams key must match a leaf path in the template (param or static).
func checkForUnknownParams(template map[string]any, runParams map[string]any) error {
	allowed := make(map[string]struct{})
	WalkPayload(template, "", func(path string, value any) WalkControl {
		allowed[path] = struct{}{}
		return WalkContinue
	})

	for path := range runParams {
		if _, ok := allowed[path]; !ok {
			return fmt.Errorf("unknown parameter %q", path)
		}
	}

	return nil
}

func resolveParamValue(def Definition, runParams map[string]any) (any, error) {
	if value, ok := runParams[def.Path]; ok {
		return coerceParamValue(def, value)
	}
	if def.Default != nil {
		return def.Default, nil
	}
	if def.Required {
		return nil, fmt.Errorf("missing required parameter %q", def.Path)
	}
	return nil, fmt.Errorf("missing parameter %q", def.Path)
}

func deepCopy(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case map[string]string:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = vv
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = deepCopy(item)
		}
		return out
	default:
		return val
	}
}

func deepCopyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = deepCopy(value)
	}
	return dst
}
