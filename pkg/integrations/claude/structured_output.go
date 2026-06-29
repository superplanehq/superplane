package claude

import "fmt"

// validateClaudeOutputSchema enforces Anthropic's structured-output requirement
// that every object set "additionalProperties": false. It walks nested objects,
// array items, combinators, and $defs. Unlike OpenAI strict mode, Anthropic
// permits truly optional fields, so this intentionally does NOT require every
// property to be listed in "required".
func validateClaudeOutputSchema(node any, path string) error {
	m, ok := node.(map[string]any)
	if !ok {
		return nil
	}

	if isObjectSchema(m) {
		if b, ok := m["additionalProperties"].(bool); !ok || b {
			return fmt.Errorf("schema object at %s must set \"additionalProperties\": false", schemaPath(path))
		}
		if props, ok := m["properties"].(map[string]any); ok {
			for key, child := range props {
				if err := validateClaudeOutputSchema(child, path+"."+key); err != nil {
					return err
				}
			}
		}
	}

	if items, ok := m["items"]; ok {
		if err := validateClaudeOutputSchema(items, path+".items"); err != nil {
			return err
		}
	}
	for _, key := range []string{"anyOf", "allOf", "oneOf"} {
		if branches, ok := m[key].([]any); ok {
			for i, branch := range branches {
				if err := validateClaudeOutputSchema(branch, fmt.Sprintf("%s.%s[%d]", path, key, i)); err != nil {
					return err
				}
			}
		}
	}
	for _, key := range []string{"$defs", "definitions"} {
		if defs, ok := m[key].(map[string]any); ok {
			for name, def := range defs {
				if err := validateClaudeOutputSchema(def, path+"."+key+"."+name); err != nil {
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

func schemaPath(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}
