// Package structuredoutput lets LLM components accept a JSON Schema
// describing the desired response, validating it and normalizing it to the
// constraints strict-mode providers enforce (every object gets
// "additionalProperties": false; strict mode also requires listing every
// property in "required").
//
// Some agent-style components have no server-side schema enforcement at all.
// For those, PromptSuffix and ExtractJSON emulate it by asking the model to
// comply and best-effort parsing its final message.
package structuredoutput

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

// DefaultSchema is the starter JSON Schema shown in the field for the user to edit.
const DefaultSchema = `{
  "type": "object",
  "properties": {
    "summary": {
      "type": "string",
      "description": "One-sentence summary"
    },
    "severity": {
      "type": "string",
      "enum": ["low", "medium", "high"]
    }
  },
  "required": ["summary", "severity"]
}`

// ConfigField returns the togglable JSON-schema text field for a component.
func ConfigField(name, label, description string) configuration.Field {
	return configuration.Field{
		Name:        name,
		Label:       label,
		Type:        configuration.FieldTypeText,
		Required:    false,
		Togglable:   true,
		Default:     DefaultSchema,
		Placeholder: DefaultSchema,
		Description: description,
	}
}

// Parse reads the raw config value (a JSON Schema string) into a schema object.
// It returns (nil, nil) when the field is unset, so callers can treat structured
// output as optional. Otherwise it validates that the value is a JSON object that
// looks like an object schema, so malformed input surfaces at config/Setup time
// rather than as a provider error.
func Parse(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	s, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("structured output schema must be a JSON string")
	}
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}

	var doc any
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		return nil, fmt.Errorf("structured output schema is not valid JSON: %w", err)
	}
	schema, ok := doc.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("structured output schema must be a JSON object")
	}
	if err := validateRoot(schema); err != nil {
		return nil, err
	}
	return schema, nil
}

// validateRoot checks the schema is usable as a structured-output root: an object
// type with at least one property. Both providers require an object at the root.
func validateRoot(schema map[string]any) error {
	if t, ok := schema["type"]; ok {
		if ts, isStr := t.(string); !isStr || ts != "object" {
			return fmt.Errorf(`structured output schema root "type" must be "object"`)
		}
	}
	props, ok := schema["properties"]
	if !ok {
		return fmt.Errorf(`structured output schema must define "properties"`)
	}
	pm, ok := props.(map[string]any)
	if !ok {
		return fmt.Errorf(`structured output schema "properties" must be an object`)
	}
	if len(pm) == 0 {
		return fmt.Errorf(`structured output schema "properties" must define at least one property`)
	}
	return nil
}

// Prepare returns a copy of the schema normalized to provider constraints, ready
// to send. It does not mutate the input.
func Prepare(schema map[string]any, strict bool) map[string]any {
	return normalizeMap(schema, strict)
}

// ValidateAtSetup validates a raw structured-output config value at Setup
// time. An empty value is fine (structured output is optional). A value that
// still contains an unresolved "{{" expression is skipped, mirroring how the
// prompt field itself is only resolved at Execute.
func ValidateAtSetup(raw string) error {
	if strings.TrimSpace(raw) == "" || strings.Contains(raw, "{{") {
		return nil
	}
	_, err := Parse(raw)
	return err
}

// PromptSuffix renders schema-following instructions to append to an agent's
// task prompt, for components with no server-side schema enforcement to rely
// on instead. Returns "" if schema is nil.
func PromptSuffix(schema map[string]any) string {
	if schema == nil {
		return ""
	}
	encoded, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("\n\nBefore you finish, include a fenced json code block in your message with data matching this JSON Schema exactly:\n\n```json\n%s\n```\n", encoded)
}

// jsonFencePattern matches a fenced code block, with or without a "json"
// language tag.
var jsonFencePattern = regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)\\n?```")

// ExtractJSON pulls a JSON object out of text produced by PromptSuffix's
// instructions: it tries each fenced code block (last to first), then the
// whole trimmed text, and reports false if none parse as a JSON object. This
// is best-effort with no schema validation — callers decide when to trust it.
func ExtractJSON(text string) (any, bool) {
	matches := jsonFencePattern.FindAllStringSubmatch(text, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		if parsed, ok := unmarshalJSONObject(matches[i][1]); ok {
			return parsed, true
		}
	}
	return unmarshalJSONObject(text)
}

func unmarshalJSONObject(s string) (any, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	var parsed any
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		return nil, false
	}
	if _, ok := parsed.(map[string]any); !ok {
		return nil, false
	}
	return parsed, true
}

func normalizeMap(node map[string]any, strict bool) map[string]any {
	out := make(map[string]any, len(node))
	for k, v := range node {
		out[k] = normalizeValue(v, strict)
	}

	// Enforce the constraints providers require on every object schema.
	if isObjectSchema(out) {
		out["additionalProperties"] = false
		if strict {
			out["required"] = propertyNames(out)
		}
	}
	return out
}

func normalizeValue(v any, strict bool) any {
	switch val := v.(type) {
	case map[string]any:
		return normalizeMap(val, strict)
	case []any:
		arr := make([]any, len(val))
		for i, e := range val {
			arr[i] = normalizeValue(e, strict)
		}
		return arr
	default:
		return v
	}
}

// isObjectSchema reports whether a schema node describes an object: an explicit
// "type":"object", or (when type is omitted) the presence of "properties".
func isObjectSchema(node map[string]any) bool {
	if t, ok := node["type"].(string); ok {
		return t == "object"
	}
	_, ok := node["properties"]
	return ok
}

// propertyNames returns the sorted keys of an object schema's "properties".
func propertyNames(node map[string]any) []string {
	props, ok := node["properties"].(map[string]any)
	if !ok {
		return []string{}
	}
	names := make([]string, 0, len(props))
	for k := range props {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
