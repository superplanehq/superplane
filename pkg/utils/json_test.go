package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalEmbeddedJSON_ParsesValidJSON(t *testing.T) {
	testCases := []struct {
		input    []byte
		expected map[string]any
		name     string
	}{
		{
			input:    []byte(`{"key":"value"}`),
			expected: map[string]any{"key": "value"},
			name:     "simple object",
		},
		{
			input: []byte(`{"a":1,"b":{"c":true}}`),
			expected: map[string]any{
				"a": float64(1),
				"b": map[string]any{"c": true},
			},
			name: "nested object with number and boolean",
		},
		{
			input:    []byte(`{}`),
			expected: map[string]any{},
			name:     "empty object",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var once sync.Once
			var target map[string]any

			result := UnmarshalEmbeddedJSON(&once, tc.input, &target)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUnmarshalEmbeddedJSON_OnceBehavior(t *testing.T) {
	var once sync.Once
	var target map[string]any

	first := UnmarshalEmbeddedJSON(&once, []byte(`{"first":"value"}`), &target)
	assert.Equal(t, map[string]any{"first": "value"}, first)

	second := UnmarshalEmbeddedJSON(&once, []byte(`{"second":"other"}`), &target)

	// The second call must not re-parse: it should return the exact same
	// result as the first call, and must not contain data from the
	// second (unused) payload.
	assert.Equal(t, first, second)
	assert.NotContains(t, second, "second")

	// The target itself should still reflect the original parsed value.
	assert.Equal(t, map[string]any{"first": "value"}, target)
}
