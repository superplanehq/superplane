package models_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__JSONValue__ScanPreservesJSONNumbers(t *testing.T) {
	var value models.JSONValue

	err := value.Scan([]byte(`{"data":{"plain":14000000,"large":12345678901234567890,"small":0.0000001,"name":"deploy","active":true,"missing":null,"labels":["prod"]},"items":[42]}`))
	require.NoError(t, err)

	data, ok := value.Data().(map[string]any)
	require.True(t, ok)

	payload, ok := data["data"].(map[string]any)
	require.True(t, ok)

	plain, ok := payload["plain"].(json.Number)
	require.True(t, ok)
	assert.Equal(t, "14000000", plain.String())

	large, ok := payload["large"].(json.Number)
	require.True(t, ok)
	assert.Equal(t, "12345678901234567890", large.String())

	small, ok := payload["small"].(json.Number)
	require.True(t, ok)
	assert.Equal(t, "0.0000001", small.String())

	assert.Equal(t, "deploy", payload["name"])
	assert.Equal(t, true, payload["active"])
	assert.Nil(t, payload["missing"])

	labels, ok := payload["labels"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"prod"}, labels)

	encoded, err := value.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"large":12345678901234567890`)
	assert.Contains(t, string(encoded), `"small":0.0000001`)
}

func Test__UnmarshalJSONValue__RejectsMultipleTopLevelValues(t *testing.T) {
	var value any
	err := models.UnmarshalJSONValue([]byte(`{"a":1}{"b":2}`), &value)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple top-level values")
}

func Test__IsJSONIntegerToken(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "integer", raw: "42", want: true},
		{name: "negative integer", raw: "-42", want: true},
		{name: "large integer", raw: "12345678901234567890", want: true},
		{name: "decimal", raw: "1.0", want: false},
		{name: "exponent", raw: "1e5", want: false},
		{name: "empty", raw: "", want: false},
		{name: "minus only", raw: "-", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, models.IsJSONIntegerToken(tt.raw))
		})
	}
}
