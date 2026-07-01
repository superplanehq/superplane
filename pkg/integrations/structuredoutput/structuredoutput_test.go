package structuredoutput

import (
	"reflect"
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
