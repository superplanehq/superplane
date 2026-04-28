package expressionvalidation

import (
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/configuration"
)

func knownSet(names ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, n := range names {
		out[n] = struct{}{}
	}
	return out
}

func assertOneError(t *testing.T, errs []ExpressionError, wantPath, wantMsg string) {
	t.Helper()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(errs), errs)
	}
	if errs[0].FieldPath != wantPath {
		t.Fatalf("expected FieldPath %q, got %q", wantPath, errs[0].FieldPath)
	}
	if !strings.Contains(errs[0].Err.Error(), wantMsg) {
		t.Fatalf("expected error containing %q, got: %v", wantMsg, errs[0].Err)
	}
}

func TestValidateNodeExpressions_FieldTypes(t *testing.T) {
	t.Run("string with embedded expr", func(t *testing.T) {
		fields := []configuration.Field{{Name: "command", Type: configuration.FieldTypeString}}
		errs := validateNodeExpressions("n1", "Deploy", map[string]any{
			"command": "echo {{ $['Missing'].x }}",
		}, fields, knownSet("Build"))
		assertOneError(t, errs, "command", "unknown node reference 'Missing'")
	})

	t.Run("text with mixed text and expression", func(t *testing.T) {
		fields := []configuration.Field{{Name: "message", Type: configuration.FieldTypeText}}
		errs := validateNodeExpressions("n1", "Notify", map[string]any{
			"message": `My value: {{ "dsadasd" + invalid() }}`,
		}, fields, knownSet())
		assertOneError(t, errs, "message", "compile error")
	})

	t.Run("text with multiple placeholders one broken", func(t *testing.T) {
		fields := []configuration.Field{{Name: "message", Type: configuration.FieldTypeText}}
		errs := validateNodeExpressions("n1", "Notify", map[string]any{
			"message": "{{ $['Build'].x }} and {{ $['Missing'].y }}",
		}, fields, knownSet("Build"))
		assertOneError(t, errs, "message", "unknown node reference 'Missing'")
	})

	t.Run("expression bare literals pass", func(t *testing.T) {
		fields := []configuration.Field{{Name: "value", Type: configuration.FieldTypeExpression}}
		for _, raw := range []string{"true", "default", "ok", "down", "1 + 2", `"plain"`, `$["Build"].x + 1`} {
			errs := validateNodeExpressions("n1", "Mem", map[string]any{
				"value": raw,
			}, fields, knownSet("Build"))
			if len(errs) != 0 {
				t.Fatalf("expected no errors for %q, got %+v", raw, errs)
			}
		}
	})

	t.Run("expression bare syntax error caught", func(t *testing.T) {
		fields := []configuration.Field{{Name: "value", Type: configuration.FieldTypeExpression}}
		errs := validateNodeExpressions("n1", "Mem", map[string]any{
			"value": `$["Build"].data.calendar.day asdadasd 2313312`,
		}, fields, knownSet("Build"))
		assertOneError(t, errs, "value", "syntax error")
	})

	t.Run("expression bare unknown node ref caught", func(t *testing.T) {
		fields := []configuration.Field{{Name: "value", Type: configuration.FieldTypeExpression}}
		errs := validateNodeExpressions("n1", "Mem", map[string]any{
			"value": `$["Missing"].x`,
		}, fields, knownSet("Build"))
		assertOneError(t, errs, "value", "unknown node reference 'Missing'")
	})

	t.Run("expression placeholder content validated", func(t *testing.T) {
		fields := []configuration.Field{{Name: "value", Type: configuration.FieldTypeExpression}}
		errs := validateNodeExpressions("n1", "Mem", map[string]any{
			"value": "{{ $['Missing'].x }}",
		}, fields, knownSet("Build"))
		assertOneError(t, errs, "value", "unknown node reference 'Missing'")
	})

	t.Run("expression valid placeholder passes", func(t *testing.T) {
		fields := []configuration.Field{{Name: "value", Type: configuration.FieldTypeExpression}}
		errs := validateNodeExpressions("n1", "Mem", map[string]any{
			"value": "{{ int(now().Unix()) }}",
		}, fields, knownSet())
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %+v", errs)
		}
	})
}

func TestValidateNodeExpressions_NestedPaths(t *testing.T) {
	t.Run("nested object schema", func(t *testing.T) {
		fields := []configuration.Field{
			{
				Name: "body",
				Type: configuration.FieldTypeObject,
				TypeOptions: &configuration.TypeOptions{
					Object: &configuration.ObjectTypeOptions{
						Schema: []configuration.Field{
							{
								Name: "headers",
								Type: configuration.FieldTypeObject,
								TypeOptions: &configuration.TypeOptions{
									Object: &configuration.ObjectTypeOptions{
										Schema: []configuration.Field{
											{Name: "auth", Type: configuration.FieldTypeString},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		errs := validateNodeExpressions("n1", "Call", map[string]any{
			"body": map[string]any{
				"headers": map[string]any{
					"auth": "Bearer {{ $['Missing'].token }}",
				},
			},
		}, fields, knownSet("Build"))
		assertOneError(t, errs, "body.headers.auth", "unknown node reference 'Missing'")
	})

	t.Run("list of objects", func(t *testing.T) {
		fields := []configuration.Field{
			{
				Name: "items",
				Type: configuration.FieldTypeList,
				TypeOptions: &configuration.TypeOptions{
					List: &configuration.ListTypeOptions{
						ItemDefinition: &configuration.ListItemDefinition{
							Type: configuration.FieldTypeObject,
							Schema: []configuration.Field{
								{Name: "url", Type: configuration.FieldTypeString},
							},
						},
					},
				},
			},
		}
		errs := validateNodeExpressions("n1", "Fetch", map[string]any{
			"items": []any{
				map[string]any{"url": "ok"},
				map[string]any{"url": "ok"},
				map[string]any{"url": "{{ root( }}"},
			},
		}, fields, knownSet())
		assertOneError(t, errs, "items[2].url", "syntax error")
	})

	t.Run("list of strings", func(t *testing.T) {
		fields := []configuration.Field{
			{
				Name: "items",
				Type: configuration.FieldTypeList,
				TypeOptions: &configuration.TypeOptions{
					List: &configuration.ListTypeOptions{
						ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
					},
				},
			},
		}
		errs := validateNodeExpressions("n1", "Fetch", map[string]any{
			"items": []any{"{{ $['Missing'].x }}"},
		}, fields, knownSet())
		assertOneError(t, errs, "items[0]", "unknown node reference 'Missing'")
	})

	t.Run("key not in schema still walked", func(t *testing.T) {
		fields := []configuration.Field{}
		errs := validateNodeExpressions("n1", "Free", map[string]any{
			"extra": map[string]any{
				"unknown": "{{ $['Missing'].x }}",
			},
		}, fields, knownSet())
		assertOneError(t, errs, "extra.unknown", "unknown node reference 'Missing'")
	})
}

func TestValidateNodeExpressions_Aggregation(t *testing.T) {
	t.Run("multiple errors single node deterministic order", func(t *testing.T) {
		fields := []configuration.Field{
			{Name: "a", Type: configuration.FieldTypeString},
			{Name: "b", Type: configuration.FieldTypeString},
			{Name: "c", Type: configuration.FieldTypeString},
		}
		errs := validateNodeExpressions("n1", "Multi", map[string]any{
			"a": "{{ $['M1'].x }}",
			"b": "{{ $['M2'].x }}",
			"c": "{{ $['M3'].x }}",
		}, fields, knownSet())
		if len(errs) != 3 {
			t.Fatalf("expected 3 errors, got %+v", errs)
		}
		paths := make(map[string]bool, 3)
		for _, e := range errs {
			paths[e.FieldPath] = true
		}
		for _, p := range []string{"a", "b", "c"} {
			if !paths[p] {
				t.Fatalf("missing FieldPath %q in errors: %+v", p, errs)
			}
		}
	})
}

func TestValidateNodeExpressions_Edge(t *testing.T) {
	t.Run("nil configuration emits no errors", func(t *testing.T) {
		errs := validateNodeExpressions("n1", "Empty", nil, nil, knownSet())
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %+v", errs)
		}
	})

	t.Run("empty placeholder rejected", func(t *testing.T) {
		fields := []configuration.Field{{Name: "command", Type: configuration.FieldTypeString}}
		errs := validateNodeExpressions("n1", "Empty", map[string]any{
			"command": "{{   }}",
		}, fields, knownSet())
		assertOneError(t, errs, "command", "expression body is empty")
	})

	t.Run("non-expression value not validated", func(t *testing.T) {
		fields := []configuration.Field{{Name: "command", Type: configuration.FieldTypeString}}
		errs := validateNodeExpressions("n1", "Plain", map[string]any{
			"command": "hello world",
		}, fields, knownSet())
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %+v", errs)
		}
	})
}
