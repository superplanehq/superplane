package canvases

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__toStructpbCompatible__ConvertsJSONNumbersOnly(t *testing.T) {
	converted, ok := toStructpbCompatible(map[string]any{
		"number":    json.Number("42"),
		"unsafeInt": json.Number("9007199254740993"),
		"snowflake": json.Number("12345678901234567890"),
		"invalid":   json.Number("not-a-number"),
		"string":    "deploy",
		"active":    true,
		"missing":   nil,
		"list": []any{
			json.Number("3"),
			"prod",
		},
		"nested": map[string]any{
			"small": json.Number("0.0000001"),
		},
	}).(map[string]any)

	require.True(t, ok)
	assert.Equal(t, float64(42), converted["number"])
	assert.Equal(t, float64(1), jsonNumberForStructpb(json.Number("1.0")))
	assert.Equal(t, float64(100000), jsonNumberForStructpb(json.Number("1e5")))
	assert.Equal(t, float64(10.5), jsonNumberForStructpb(json.Number("10.5")))
	assert.Equal(t, "9007199254740993", converted["unsafeInt"])
	assert.Equal(t, "12345678901234567890", converted["snowflake"])
	assert.Equal(t, "not-a-number", converted["invalid"])
	assert.Equal(t, "deploy", converted["string"])
	assert.Equal(t, true, converted["active"])
	assert.Nil(t, converted["missing"])

	list, ok := converted["list"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{float64(3), "prod"}, list)

	nested, ok := converted["nested"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0.0000001), nested["small"])
}

func Test__toStructpbCompatible(t *testing.T) {
	in := map[string]any{
		"arr": []any{float64(1), "two"},
		"nested": map[string]any{
			"k": true,
		},
	}
	out := toStructpbCompatible(in)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	arr, ok := m["arr"].([]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), arr[0])
}
