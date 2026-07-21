package structuredoutput

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse_Unset(t *testing.T) {
	for _, raw := range []any{nil, "", "   ", "\n\t "} {
		schema, err := Parse(raw)
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", raw, err)
		}
		if schema != nil {
			t.Errorf("Parse(%q) = %v, want nil (treated as unset)", raw, schema)
		}
	}
}

func TestParse_Valid(t *testing.T) {
	schema, err := Parse(`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("expected type object, got %v", schema["type"])
	}
}

func TestParse_DefaultSchemaIsValid(t *testing.T) {
	if _, err := Parse(DefaultSchema); err != nil {
		t.Errorf("DefaultSchema should parse cleanly: %v", err)
	}
}

func TestParse_Errors(t *testing.T) {
	cases := map[string]string{
		"invalid JSON":          `{"type":"object",}`,
		"not an object":         `["a","b"]`,
		"scalar":                `"hello"`,
		"non-object root type":  `{"type":"array","properties":{"a":{"type":"string"}}}`,
		"missing properties":    `{"type":"object"}`,
		"empty properties":      `{"type":"object","properties":{}}`,
		"properties not object": `{"type":"object","properties":[]}`,
	}
	for name, raw := range cases {
		if _, err := Parse(raw); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestPrepare_AddsAdditionalProperties(t *testing.T) {
	schema, _ := Parse(`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}`)
	out := Prepare(schema, false)
	if out["additionalProperties"] != false {
		t.Errorf("expected additionalProperties false, got %v", out["additionalProperties"])
	}
	// non-strict keeps required as-is
	if req, _ := out["required"].([]any); len(req) != 1 || req[0] != "a" {
		t.Errorf("non-strict should keep required as written, got %v", out["required"])
	}
}

func TestPrepare_NestedObjectsAndArrays(t *testing.T) {
	schema, _ := Parse(`{
		"type":"object",
		"properties":{
			"item":{"type":"object","properties":{"x":{"type":"number"}},"required":["x"]},
			"list":{"type":"array","items":{"type":"object","properties":{"y":{"type":"string"}}}}
		},
		"required":["item","list"]
	}`)
	out := Prepare(schema, false)

	props := out["properties"].(map[string]any)
	item := props["item"].(map[string]any)
	if item["additionalProperties"] != false {
		t.Errorf("nested object should get additionalProperties false")
	}
	items := props["list"].(map[string]any)["items"].(map[string]any)
	if items["additionalProperties"] != false {
		t.Errorf("array item object should get additionalProperties false")
	}
}

func TestPrepare_StrictWithNoProperties(t *testing.T) {
	// An object schema with no "properties" (e.g. an empty object schema
	// built directly rather than through Parse, which rejects that case)
	// must not panic; strict mode just requires nothing.
	out := Prepare(map[string]any{"type": "object"}, true)
	req, ok := out["required"].([]string)
	if !ok {
		t.Fatalf("required should be []string in strict mode, got %T", out["required"])
	}
	if len(req) != 0 {
		t.Errorf("expected no required properties, got %v", req)
	}
}

func TestPrepare_StrictForcesRequired(t *testing.T) {
	schema, _ := Parse(`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"number"}},"required":["a"]}`)
	out := Prepare(schema, true)
	req, ok := out["required"].([]string)
	if !ok {
		t.Fatalf("required should be []string in strict mode, got %T", out["required"])
	}
	if !reflect.DeepEqual(req, []string{"a", "b"}) {
		t.Errorf("strict mode should require all properties (sorted), got %v", req)
	}
}

func TestPrepare_DoesNotMutateInput(t *testing.T) {
	schema, _ := Parse(`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}`)
	_ = Prepare(schema, true)
	if _, exists := schema["additionalProperties"]; exists {
		t.Errorf("Prepare must not mutate the input schema")
	}
}

func TestValidateAtSetup(t *testing.T) {
	for _, raw := range []string{"", "   ", "{{ event.schema }}"} {
		if err := ValidateAtSetup(raw); err != nil {
			t.Errorf("ValidateAtSetup(%q) unexpected error: %v", raw, err)
		}
	}

	if err := ValidateAtSetup(`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}`); err != nil {
		t.Errorf("unexpected error for valid schema: %v", err)
	}

	if err := ValidateAtSetup(`{"type":"object"}`); err == nil {
		t.Error("expected error for schema missing properties")
	}
}

func TestPromptSuffix(t *testing.T) {
	if got := PromptSuffix(nil); got != "" {
		t.Errorf("PromptSuffix(nil) = %q, want empty", got)
	}

	schema, _ := Parse(`{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`)
	got := PromptSuffix(schema)
	if !strings.Contains(got, "```json") {
		t.Errorf("expected a fenced json block, got %q", got)
	}
	if !strings.Contains(got, `"summary"`) {
		t.Errorf("expected the schema to be embedded, got %q", got)
	}
}

func TestPromptSuffix_UnencodableSchema(t *testing.T) {
	// A channel value can't be marshaled to JSON, so PromptSuffix must fall
	// back to its empty-string result rather than panicking or erroring.
	schema := map[string]any{"bad": make(chan int)}
	if got := PromptSuffix(schema); got != "" {
		t.Errorf("PromptSuffix(unencodable) = %q, want empty", got)
	}
}

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name string
		text string
		want map[string]any
		ok   bool
	}{
		{
			name: "fenced json block",
			text: "Here you go:\n\n```json\n{\"summary\": \"ok\"}\n```\n",
			want: map[string]any{"summary": "ok"},
			ok:   true,
		},
		{
			name: "bare fence without json tag",
			text: "```\n{\"summary\": \"ok\"}\n```",
			want: map[string]any{"summary": "ok"},
			ok:   true,
		},
		{
			name: "whole message is JSON",
			text: `{"summary": "ok"}`,
			want: map[string]any{"summary": "ok"},
			ok:   true,
		},
		{
			name: "last fenced block wins",
			text: "Example:\n```json\n{\"summary\": \"example\"}\n```\n\nActual:\n```json\n{\"summary\": \"real\"}\n```",
			want: map[string]any{"summary": "real"},
			ok:   true,
		},
		{
			name: "non-json fence falls back to next candidate",
			text: "```bash\necho hi\n```\n```json\n{\"summary\": \"ok\"}\n```",
			want: map[string]any{"summary": "ok"},
			ok:   true,
		},
		{
			name: "no JSON anywhere",
			text: "Sorry, I can't help with that.",
			ok:   false,
		},
		{
			name: "JSON array is not an object",
			text: `["a", "b"]`,
			ok:   false,
		},
		{
			name: "blank text",
			text: "   ",
			ok:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ExtractJSON(tc.text)
			if ok != tc.ok {
				t.Fatalf("ExtractJSON(%q) ok = %v, want %v", tc.text, ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ExtractJSON(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}
