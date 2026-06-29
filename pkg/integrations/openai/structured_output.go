package openai

import "fmt"

// decodeOutputSchema normalizes the optional structured-output JSON schema from
// the component configuration. It returns a nil map when no schema is configured
// (field absent, or toggled on but empty). The schema must be a JSON object; a
// string value (e.g. an expression) is rejected because the schema cannot be
// templated.
func decodeOutputSchema(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	schema, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("outputSchema must be a JSON object (expressions are not supported in the schema)")
	}
	if len(schema) == 0 {
		return nil, nil
	}
	if t, _ := schema["type"].(string); t != "object" {
		return nil, fmt.Errorf("outputSchema root must have \"type\": \"object\"")
	}
	return schema, nil
}

// validateStrictJSONSchema enforces the rules OpenAI requires under strict mode:
// every object must set "additionalProperties": false and list all of its
// properties in "required". It walks nested objects, array items, combinators,
// and $defs. Shapes it does not recognize as object schemas are skipped, so a
// correctly authored OpenAI schema always passes; the checks only fire on the
// documented violations (e.g. a schema authored for a more permissive provider).
func validateStrictJSONSchema(node any, path string) error {
	m, ok := node.(map[string]any)
	if !ok {
		return nil
	}

	if isObjectSchema(m) {
		if b, ok := m["additionalProperties"].(bool); !ok || b {
			return fmt.Errorf("schema object at %s must set \"additionalProperties\": false for OpenAI strict mode", schemaPath(path))
		}
		props, _ := m["properties"].(map[string]any)
		required := stringSet(m["required"])
		for key := range props {
			if !required[key] {
				return fmt.Errorf("schema object at %s must list property %q in \"required\" (OpenAI strict mode requires every property; use a nullable type such as [\"string\",\"null\"] for optional fields)", schemaPath(path), key)
			}
		}
		for key, child := range props {
			if err := validateStrictJSONSchema(child, path+"."+key); err != nil {
				return err
			}
		}
	}

	if items, ok := m["items"]; ok {
		if err := validateStrictJSONSchema(items, path+".items"); err != nil {
			return err
		}
	}
	for _, key := range []string{"anyOf", "allOf", "oneOf"} {
		if branches, ok := m[key].([]any); ok {
			for i, branch := range branches {
				if err := validateStrictJSONSchema(branch, fmt.Sprintf("%s.%s[%d]", path, key, i)); err != nil {
					return err
				}
			}
		}
	}
	for _, key := range []string{"$defs", "definitions"} {
		if defs, ok := m[key].(map[string]any); ok {
			for name, def := range defs {
				if err := validateStrictJSONSchema(def, path+"."+key+"."+name); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func isObjectSchema(m map[string]any) bool {
	if t, ok := m["type"].(string); ok {
		return t == "object"
	}
	_, hasProps := m["properties"]
	return hasProps
}

func stringSet(v any) map[string]bool {
	set := map[string]bool{}
	if arr, ok := v.([]any); ok {
		for _, e := range arr {
			if s, ok := e.(string); ok {
				set[s] = true
			}
		}
	}
	return set
}

func schemaPath(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}
