package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalEmbeddedJSON(t *testing.T) {
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
			input:    []byte(`{"str":"text","num":42,"bool":true}`),
			expected: map[string]any{"str": "text", "num": float64(42), "bool": true},
			name:     "mixed value types",
		},
		{
			input:    []byte(`{}`),
			expected: map[string]any{},
			name:     "empty object",
		},
		{
			input:    []byte(`not valid json`),
			expected: map[string]any{},
			name:     "invalid json returns empty map",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var once sync.Once
			var target map[string]any

			result := UnmarshalEmbeddedJSON(&once, tc.input, &target)

			assert.Equal(t, tc.expected, result)
			assert.Equal(t, tc.expected, target)
		})
	}
}

func TestUnmarshalEmbeddedJSONOnce(t *testing.T) {
	var once sync.Once
	var target map[string]any

	first := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"first"}`), &target)
	assert.Equal(t, map[string]any{"key": "first"}, first)

	// A second call with different data must return the already-parsed
	// result because sync.Once prevents re-parsing.
	second := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"second"}`), &target)
	assert.Equal(t, map[string]any{"key": "first"}, second)
	assert.Equal(t, map[string]any{"key": "first"}, target)
}
