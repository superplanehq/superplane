package params

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func issueExamplePayload() map[string]any {
	return map[string]any{
		"body": map[string]any{
			"name": "param(type:string, title:'Enter a machine name', default:'machine-1', required:false)",
			"size": "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true)",
		},
	}
}

func TestValidateRunParams(t *testing.T) {
	defs, err := ParseParams(issueExamplePayload())
	require.NoError(t, err)

	require.NoError(t, ValidateRunParams(defs, map[string]any{"body.size": "2 vCPU"}))
	require.Error(t, ValidateRunParams(defs, map[string]any{}))
}

func TestApplyParams_issueExample(t *testing.T) {
	out, err := ApplyParams(issueExamplePayload(), map[string]any{
		"body.size": "4 vCPU",
	})
	require.NoError(t, err)

	body, ok := out["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "machine-1", body["name"])
	assert.Equal(t, "4 vCPU", body["size"])
}

func TestApplyParams_usesDefaults(t *testing.T) {
	out, err := ApplyParams(issueExamplePayload(), map[string]any{
		"body.size": "2 vCPU",
	})
	require.NoError(t, err)
	body := out["body"].(map[string]any)
	assert.Equal(t, "machine-1", body["name"])
}

func TestApplyParams_missingRequired(t *testing.T) {
	_, err := ApplyParams(issueExamplePayload(), map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "body.size")
}

func TestApplyParams_unknownKey(t *testing.T) {
	_, err := ApplyParams(issueExamplePayload(), map[string]any{
		"body.size":  "2 vCPU",
		"body.extra": "nope",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown parameter")
}

func TestApplyParams_invalidSelectValue(t *testing.T) {
	_, err := ApplyParams(issueExamplePayload(), map[string]any{
		"body.size": "16 vCPU",
	})
	require.Error(t, err)
}

func TestApplyParams_staticLeafOverride(t *testing.T) {
	template := map[string]any{
		"message": "hello",
		"count":   float64(1),
	}
	out, err := ApplyParams(template, map[string]any{
		"message": "world",
		"count":   float64(5),
	})
	require.NoError(t, err)
	assert.Equal(t, "world", out["message"])
	assert.Equal(t, float64(5), out["count"])
}

func TestApplyParams_doesNotMutateTemplate(t *testing.T) {
	template := issueExamplePayload()
	originalSize := template["body"].(map[string]any)["size"]

	_, err := ApplyParams(template, map[string]any{"body.size": "2 vCPU"})
	require.NoError(t, err)
	assert.Equal(t, originalSize, template["body"].(map[string]any)["size"])
}

func TestApplyParams_arrayPath(t *testing.T) {
	template := map[string]any{
		"items": []any{
			map[string]any{
				"id": "param(type:string, default:'a', required:false)",
			},
		},
	}
	out, err := ApplyParams(template, map[string]any{"items[0].id": "b"})
	require.NoError(t, err)
	items := out["items"].([]any)
	item := items[0].(map[string]any)
	assert.Equal(t, "b", item["id"])
}

func TestApplyParams_nilRunParams(t *testing.T) {
	template := map[string]any{
		"name": "param(type:string, default:'hello', required:false)",
	}
	out, err := ApplyParams(template, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", out["name"])
}

func TestApplyParams_rejectsInvalidTemplate(t *testing.T) {
	template := map[string]any{
		"name": "param(type:string, title:'bad,comma')",
	}
	_, err := ApplyParams(template, map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name:")
}

func TestApplyParams_rejectsInvalidParamValue(t *testing.T) {
	_, err := ApplyParams(issueExamplePayload(), map[string]any{
		"body.name": 42,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "body.name")
}

func TestApplyParams_rejectsStaticTypeMismatch(t *testing.T) {
	template := map[string]any{"count": float64(1)}
	_, err := ApplyParams(template, map[string]any{"count": "not-a-number"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count")
}

func TestApplyParams_optionalParamWithoutDefaultOrValue(t *testing.T) {
	template := map[string]any{
		"label": "param(type:string, required:false)",
	}
	_, err := ApplyParams(template, map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing parameter")
}

func TestApplyParams_deepCopiesNestedStructures(t *testing.T) {
	template := map[string]any{
		"tags":   []any{"a", "b"},
		"labels": map[string]string{"env": "dev"},
		"meta": map[string]any{
			"nested": map[string]any{"x": float64(1)},
		},
	}
	out, err := ApplyParams(template, map[string]any{})
	require.NoError(t, err)

	tags := out["tags"].([]any)
	tags[0] = "changed"
	assert.Equal(t, "a", template["tags"].([]any)[0])

	labels := out["labels"].(map[string]any)
	labels["env"] = "prod"
	assert.Equal(t, "dev", template["labels"].(map[string]string)["env"])
}

func TestValidateRunParams_ignoresStaticOverrides(t *testing.T) {
	defs, err := ParseParams(issueExamplePayload())
	require.NoError(t, err)

	err = ValidateRunParams(defs, map[string]any{
		"body.size": "2 vCPU",
		"message":   "static override",
	})
	require.NoError(t, err)
}

func TestValidateRunParams_rejectsInvalidParamType(t *testing.T) {
	defs, err := ParseParams(issueExamplePayload())
	require.NoError(t, err)

	err = ValidateRunParams(defs, map[string]any{"body.name": 42})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "body.name")
}

func TestResolveParamValue(t *testing.T) {
	def := Definition{Path: "body.name", Type: ParamTypeString, Default: "default", Required: false}

	value, err := resolveParamValue(def, map[string]any{"body.name": "override"})
	require.NoError(t, err)
	assert.Equal(t, "override", value)

	value, err = resolveParamValue(def, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "default", value)

	required := Definition{Path: "body.size", Type: ParamTypeString, Required: true}
	_, err = resolveParamValue(required, map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter")
}

func TestDeepCopy(t *testing.T) {
	assert.Nil(t, deepCopyMap(nil))

	original := map[string]any{
		"list": []any{map[string]any{"k": "v"}},
		"flat": map[string]string{"a": "b"},
		"num":  float64(1),
	}
	copied := deepCopyMap(original)

	list := copied["list"].([]any)
	nested := list[0].(map[string]any)
	nested["k"] = "changed"
	assert.Equal(t, "v", original["list"].([]any)[0].(map[string]any)["k"])

	flat := copied["flat"].(map[string]any)
	flat["a"] = "c"
	assert.Equal(t, "b", original["flat"].(map[string]string)["a"])
}
