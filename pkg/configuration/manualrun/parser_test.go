package manualrun

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasParams(t *testing.T) {
	assert.True(t, HasParams(issueExamplePayload()))
	assert.False(t, HasParams(map[string]any{"message": "hello"}))
}

func TestParseParams_issueExample(t *testing.T) {
	defs, err := ParseParams(issueExamplePayload())
	require.NoError(t, err)
	require.Len(t, defs, 2)

	assert.Equal(t, []string{"body.name", "body.size"}, pathsFromDefs(defs))
	assert.Equal(t, 1, defs[0].Order)
	assert.Equal(t, 2, defs[1].Order)
}

func pathsFromDefs(defs []Definition) []string {
	paths := make([]string, len(defs))
	for i, def := range defs {
		paths[i] = def.Path
	}
	return paths
}

func TestIsParamString(t *testing.T) {
	assert.True(t, IsParamString("  param(type:string)  "))
	assert.False(t, IsParamString("param type string"))
	assert.False(t, IsParamString("static value"))
}

func TestParseParamString_issueExamples(t *testing.T) {
	t.Run("string param", func(t *testing.T) {
		def, err := ParseParamString("body.name", "param(type:string, title:'Enter a machine name', default:'machine-1', required:false, order:1)")
		require.NoError(t, err)
		assert.Equal(t, "body.name", def.Path)
		assert.Equal(t, ParamTypeString, def.Type)
		assert.Equal(t, "Enter a machine name", def.Title)
		assert.Equal(t, "machine-1", def.Default)
		assert.False(t, def.Required)
		assert.Equal(t, 1, def.Order)
	})

	t.Run("select param", func(t *testing.T) {
		def, err := ParseParamString("body.size", "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true, order:2)")
		require.NoError(t, err)
		assert.Equal(t, "body.size", def.Path)
		assert.Equal(t, ParamTypeSelect, def.Type)
		assert.Equal(t, "Select size", def.Title)
		assert.True(t, def.Required)
		assert.Equal(t, []string{"2 vCPU", "4 vCPU", "8 vCPU"}, def.Values)
		assert.Equal(t, 2, def.Order)
	})

	t.Run("omitted order defaults to zero", func(t *testing.T) {
		def, err := ParseParamString("body.name", "param(type:string, required:false)")
		require.NoError(t, err)
		assert.Equal(t, 0, def.Order)
	})
}

func TestParseParamString_rejectsInvalidQuotedCharset(t *testing.T) {
	_, err := ParseParamString("body.name", "param(type:string, title:'bad,comma')")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "comma")

	_, err = ParseParamString("body.name", "param(type:string, title:'bad\"quote')")
	require.Error(t, err)

	_, err = ParseParamString("body.name", "param(type:string, title:'bad\\'quote')")
	require.Error(t, err)
}

func TestParseParamString_rejectsMalformed(t *testing.T) {
	_, err := ParseParamString("body.name", "not a param")
	require.Error(t, err)

	_, err = ParseParamString("body.name", "param()")
	require.Error(t, err)

	_, err = ParseParamString("body.name", "param(type:string, title:'unterminated")
	require.Error(t, err)
}

func TestParseParamString_booleanAndNumber(t *testing.T) {
	def, err := ParseParamString("enabled", "param(type:boolean, required:true, default:false)")
	require.NoError(t, err)
	assert.Equal(t, ParamTypeBoolean, def.Type)
	assert.Equal(t, false, def.Default)
	assert.True(t, def.Required)

	def, err = ParseParamString("count", "param(type:number, default:42)")
	require.NoError(t, err)
	assert.Equal(t, ParamTypeNumber, def.Type)
	assert.Equal(t, float64(42), def.Default)
}

func TestParseParamString_defaultBeforeType(t *testing.T) {
	def, err := ParseParamString("body.name", "param(default:'machine-1', type:string, required:false)")
	require.NoError(t, err)
	assert.Equal(t, "machine-1", def.Default)
	assert.Equal(t, ParamTypeString, def.Type)
}

func TestParseParams_skipsStaticLeaves(t *testing.T) {
	payload := map[string]any{
		"message": "hello",
		"name":    "param(type:string, default:'a', required:false)",
	}
	defs, err := ParseParams(payload)
	require.NoError(t, err)
	require.Len(t, defs, 1)
	assert.Equal(t, "name", defs[0].Path)
}

func TestParseParams_wrapsPathOnError(t *testing.T) {
	payload := map[string]any{
		"bad": "param(type:string, title:'bad,comma')",
	}
	_, err := ParseParams(payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad:")
}

func TestParseParams_stopsAtFirstInvalidParam(t *testing.T) {
	payload := map[string]any{
		"first":  "param(type:string, title:'bad,comma')",
		"second": "param(type:string, default:'ok')",
	}
	_, err := ParseParams(payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "first:")
}

func TestParseParamString_rejectsUnknownKey(t *testing.T) {
	_, err := ParseParamString("x", "param(type:string, foo:'bar')")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown param() key")
}

func TestParseParamString_rejectsInvalidRequired(t *testing.T) {
	_, err := ParseParamString("x", "param(type:string, required:maybe)")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required:")
}

func TestParseParamString_rejectsInvalidDefault(t *testing.T) {
	_, err := ParseParamString("x", "param(type:number, default:not-a-number)")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default:")
}

func TestSplitArgs(t *testing.T) {
	args, err := splitArgs("type:string, title:'Name'")
	require.NoError(t, err)
	assert.Equal(t, "string", args["type"])
	assert.Equal(t, "'Name'", args["title"])

	_, err = splitArgs("")
	require.Error(t, err)

	_, err = splitArgs("type:string, , title:'x'")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty argument")

	_, err = splitArgs("type")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing ':'")

	_, err = splitArgs(":value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty key")

	_, err = splitArgs("type:string, type:number")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate param() key")
}

func TestParseTypeName(t *testing.T) {
	typ, err := parseTypeName("string")
	require.NoError(t, err)
	assert.Equal(t, ParamTypeString, typ)

	_, err = parseTypeName("json")
	require.Error(t, err)
}

func TestParseBoolToken(t *testing.T) {
	value, err := parseBoolToken("true")
	require.NoError(t, err)
	assert.True(t, value)

	_, err = parseBoolToken("maybe")
	require.Error(t, err)
}

func TestParseQuotedString(t *testing.T) {
	value, err := parseQuotedString("'hello'")
	require.NoError(t, err)
	assert.Equal(t, "hello", value)

	_, err = parseQuotedString("hello")
	require.Error(t, err)

	_, err = parseQuotedString("'unterminated")
	require.Error(t, err)
}

func TestParseSelectValues(t *testing.T) {
	values, err := parseSelectValues("'a|b|c'")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, values)

	_, err = parseSelectValues("'a||b'")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")

	_, err = parseSelectValues("'bad,comma'")
	require.Error(t, err)
}

func TestParseParams_sortsByOrderThenPath(t *testing.T) {
	payload := map[string]any{
		"z": "param(type:string, order:2)",
		"a": "param(type:string, order:1)",
		"m": "param(type:string, order:1)",
	}
	defs, err := ParseParams(payload)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "m", "z"}, pathsFromDefs(defs))
}

func TestParseParamString_rejectsInvalidOrder(t *testing.T) {
	_, err := ParseParamString("x", "param(type:string, order:-1)")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order:")

	_, err = ParseParamString("x", "param(type:string, order:1.5)")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order:")

	_, err = ParseParamString("x", "param(type:string, order:abc)")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order:")
}

func TestParseOrder(t *testing.T) {
	value, err := parseOrder("10")
	require.NoError(t, err)
	assert.Equal(t, 10, value)

	_, err = parseOrder("-1")
	require.Error(t, err)

	_, err = parseOrder("1.5")
	require.Error(t, err)
}

func TestParseDefaultValue(t *testing.T) {
	value, err := parseDefaultValue(ParamTypeBoolean, "false")
	require.NoError(t, err)
	assert.Equal(t, false, value)

	_, err = parseDefaultValue(ParamTypeNumber, "nope")
	require.Error(t, err)

	_, err = parseDefaultValue(ParamType("json"), "'x'")
	require.Error(t, err)
}
