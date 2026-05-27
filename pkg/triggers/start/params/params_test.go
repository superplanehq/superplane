package params

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseParamString_stringParam(t *testing.T) {
	def, ok, err := ParseParamString("param(type:string, title:'Enter a machine name', default:'machine-1', required:false)")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, TypeString, def.Type)
	assert.Equal(t, "Enter a machine name", def.Title)
	assert.Equal(t, "machine-1", def.Default)
	assert.False(t, def.Required)
}

func TestParseParamString_selectParam(t *testing.T) {
	def, ok, err := ParseParamString("param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true)")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, TypeSelect, def.Type)
	assert.Equal(t, []string{"2 vCPU", "4 vCPU", "8 vCPU"}, def.Values)
	assert.True(t, def.Required)
}

func TestParseParamString_notParam(t *testing.T) {
	_, ok, err := ParseParamString("machine-1")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestHasParams_andExtractFields(t *testing.T) {
	static := map[string]any{
		"body": map[string]any{
			"name": "machine-1",
			"size": "2 vCPU",
		},
	}
	assert.False(t, HasParams(static))
	assert.Empty(t, ExtractFields(static))

	mixed := map[string]any{
		"body": map[string]any{
			"name": "param(type:string, title:'Enter a machine name', default:'machine-1', required:false)",
			"size": "2 vCPU",
		},
	}
	fields := ExtractFields(mixed)
	require.Len(t, fields, 1)
	assert.Equal(t, "body.name", fields[0].Path)
	assert.True(t, HasParams(mixed))
}

func TestMerge_mixedAndMulti(t *testing.T) {
	mixed := map[string]any{
		"body": map[string]any{
			"name": "param(type:string, title:'Enter a machine name', default:'machine-1', required:false)",
			"size": "2 vCPU",
		},
	}
	out, err := Merge(mixed, map[string]any{"body.name": "machine-9"})
	require.NoError(t, err)
	body := out["body"].(map[string]any)
	assert.Equal(t, "machine-9", body["name"])
	assert.Equal(t, "2 vCPU", body["size"])

	multi := map[string]any{
		"body": map[string]any{
			"name": "param(type:string, title:'Name', default:'machine-1', required:false)",
			"size": "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true)",
		},
	}
	out, err = Merge(multi, map[string]any{
		"body.name": "machine-2",
		"body.size": "4 vCPU",
	})
	require.NoError(t, err)
	body = out["body"].(map[string]any)
	assert.Equal(t, "machine-2", body["name"])
	assert.Equal(t, "4 vCPU", body["size"])
}

func TestMerge_usesDefaults(t *testing.T) {
	tmpl := map[string]any{
		"name": "param(type:string, title:'Name', default:'machine-1', required:false)",
	}
	out, err := Merge(tmpl, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "machine-1", out["name"])
}

func TestMerge_requiredMissing(t *testing.T) {
	tmpl := map[string]any{
		"size": "param(type:select, values:'2 vCPU|4 vCPU', title:'Size', required:true)",
	}
	_, err := Merge(tmpl, map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "size")
}

func TestValidatePayload_invalidSyntax(t *testing.T) {
	tmpl := map[string]any{
		"name": "param(type:unknown, title:'Name')",
	}
	_, err := ValidatePayload(tmpl)
	require.Error(t, err)
}

func TestValidatePayload_missingTitle(t *testing.T) {
	tmpl := map[string]any{
		"name": "param(type:string, default:'x')",
	}
	_, err := ValidatePayload(tmpl)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title")
}

func TestContainsUnresolvedParams(t *testing.T) {
	tmpl := map[string]any{
		"name": "param(type:string, title:'Name', default:'a')",
	}
	assert.True(t, ContainsUnresolvedParams(tmpl))
	merged, err := Merge(tmpl, map[string]any{})
	require.NoError(t, err)
	assert.False(t, ContainsUnresolvedParams(merged))
}
