package structuredoutput

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/configuration"
)

func boolPtr(b bool) *bool { return &b }

func TestBuildSchema_Scalars_NonStrict(t *testing.T) {
	fields := []OutputField{
		{Name: "title", Type: TypeString, Required: boolPtr(true)},
		{Name: "count", Type: TypeInteger},                         // required nil -> default true
		{Name: "note", Type: TypeString, Required: boolPtr(false)}, // optional
	}
	schema := BuildSchema(fields, false) // Anthropic rules

	if schema["type"] != "object" || schema["additionalProperties"] != false {
		t.Fatalf("unexpected root: %v", schema)
	}
	props := schema["properties"].(map[string]any)
	if props["title"].(map[string]any)["type"] != "string" {
		t.Errorf("title type wrong: %v", props["title"])
	}
	req := toStringSlice(schema["required"])
	if !contains(req, "title") || !contains(req, "count") {
		t.Errorf("expected title and count required, got %v", req)
	}
	if contains(req, "note") {
		t.Errorf("non-strict: optional note should be omitted from required, got %v", req)
	}
}

func TestBuildSchema_StrictNullable(t *testing.T) {
	fields := []OutputField{
		{Name: "title", Type: TypeString, Required: boolPtr(true)},
		{Name: "note", Type: TypeString, Required: boolPtr(false)},
	}
	schema := BuildSchema(fields, true) // OpenAI strict

	req := toStringSlice(schema["required"])
	if !contains(req, "title") || !contains(req, "note") {
		t.Fatalf("strict mode must list every field in required, got %v", req)
	}
	props := schema["properties"].(map[string]any)
	noteType := props["note"].(map[string]any)["type"]
	ts, ok := noteType.([]any)
	if !ok || len(ts) != 2 || ts[0] != "string" || ts[1] != "null" {
		t.Errorf("optional field should be nullable [string,null], got %v", noteType)
	}
	if props["title"].(map[string]any)["type"] != "string" {
		t.Errorf("required field should stay a plain string, got %v", props["title"])
	}
}

func TestBuildSchema_Nested(t *testing.T) {
	fields := []OutputField{
		{Name: "topic", Type: TypeString, Required: boolPtr(true)},
		{Name: "tags", Type: TypeStringArray, Required: boolPtr(true)},
		{Name: "contacts", Type: TypeObjectArray, Required: boolPtr(true), Fields: []OutputField{
			{Name: "name", Type: TypeString, Required: boolPtr(true)},
			{Name: "email", Type: TypeString, Required: boolPtr(true)},
		}},
		{Name: "address", Type: TypeObject, Required: boolPtr(true), Fields: []OutputField{
			{Name: "city", Type: TypeString, Required: boolPtr(true)},
		}},
	}
	props := BuildSchema(fields, false)["properties"].(map[string]any)

	tags := props["tags"].(map[string]any)
	if tags["type"] != "array" || tags["items"].(map[string]any)["type"] != "string" {
		t.Errorf("tags should be array of strings, got %v", tags)
	}

	contacts := props["contacts"].(map[string]any)
	if contacts["type"] != "array" {
		t.Fatalf("contacts should be array, got %v", contacts["type"])
	}
	items := contacts["items"].(map[string]any)
	if items["type"] != "object" || items["additionalProperties"] != false {
		t.Errorf("array items should be object with additionalProperties:false, got %v", items)
	}
	if _, ok := items["properties"].(map[string]any)["email"]; !ok {
		t.Errorf("nested item missing email property")
	}

	address := props["address"].(map[string]any)
	if address["type"] != "object" || address["additionalProperties"] != false {
		t.Errorf("address should be a nested object, got %v", address)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		fields  []OutputField
		wantErr bool
	}{
		{"valid", []OutputField{{Name: "a", Type: TypeString}}, false},
		{"missing name", []OutputField{{Type: TypeString}}, true},
		{"missing type", []OutputField{{Name: "a"}}, true},
		{"unknown type", []OutputField{{Name: "a", Type: "weird"}}, true},
		{"object without sub-fields", []OutputField{{Name: "a", Type: TypeObject}}, true},
		{"duplicate name", []OutputField{{Name: "a", Type: TypeString}, {Name: "a", Type: TypeNumber}}, true},
		{"nested object valid", []OutputField{{Name: "a", Type: TypeObject, Fields: []OutputField{{Name: "b", Type: TypeString}}}}, false},
		{"nested duplicate", []OutputField{{Name: "a", Type: TypeObject, Fields: []OutputField{{Name: "b", Type: TypeString}, {Name: "b", Type: TypeNumber}}}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Validate(c.fields)
			if c.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDecode_RequiredDefault(t *testing.T) {
	raw := []any{
		map[string]any{"name": "a", "type": "string"},                    // required absent -> default true
		map[string]any{"name": "b", "type": "string", "required": false}, // explicit optional
		map[string]any{"name": "c", "type": "object", "fields": []any{
			map[string]any{"name": "d", "type": "string"},
		}},
	}
	fields, err := Decode(raw)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}
	if !isRequired(fields[0]) {
		t.Error("absent required toggle should default to true")
	}
	if isRequired(fields[1]) {
		t.Error("explicit required:false should be optional")
	}
	if len(fields[2].Fields) != 1 || fields[2].Fields[0].Name != "d" {
		t.Errorf("nested fields not decoded: %+v", fields[2].Fields)
	}
}

func TestConfigField(t *testing.T) {
	f := ConfigField("outputFields", "Structured Output", "desc")
	if f.Type != configuration.FieldTypeList || !f.Togglable {
		t.Fatalf("expected togglable list field, got type=%s togglable=%v", f.Type, f.Togglable)
	}
	item := f.TypeOptions.List.ItemDefinition
	if item.Type != configuration.FieldTypeObject {
		t.Fatalf("item should be object, got %s", item.Type)
	}

	names := map[string]bool{}
	var sub *configuration.Field
	for i := range item.Schema {
		names[item.Schema[i].Name] = true
		if item.Schema[i].Name == "fields" {
			sub = &item.Schema[i]
		}
	}
	for _, n := range []string{"name", "type", "description", "required", "fields"} {
		if !names[n] {
			t.Errorf("missing item field %q", n)
		}
	}
	if sub == nil || sub.Type != configuration.FieldTypeList {
		t.Fatal("expected nested 'fields' list for recursion")
	}
	if len(sub.VisibilityConditions) == 0 {
		t.Error("nested 'fields' should be gated by VisibilityConditions on type")
	}
}

func toStringSlice(v any) []string {
	if s, ok := v.([]string); ok {
		return s
	}
	out := []string{}
	if arr, ok := v.([]any); ok {
		for _, e := range arr {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
