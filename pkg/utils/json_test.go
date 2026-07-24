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
			name:  "valid JSON object",
			input: []byte(`{"name":"test","enabled":true}`),
			expected: map[string]any{
				"name":    "test",
				"enabled": true,
			},
		},
		{
			name:     "empty JSON object",
			input:    []byte(`{}`),
			expected: map[string]any{},
		},
		{
			name:     "invalid JSON returns empty map",
			input:    []byte(`{invalid`),
			expected: map[string]any{},
		},
		{
			name:     "empty input returns empty map",
			input:    []byte(``),
			expected: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var once sync.Once
			var target map[string]any

			result := UnmarshalEmbeddedJSON(&once, tc.input, &target)

			assert.NotNil(t, result)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUnmarshalEmbeddedJSON_OnceSemantics(t *testing.T) {
	var once sync.Once
	var target map[string]any

	first := UnmarshalEmbeddedJSON(&once, []byte(`{"value":"first"}`), &target)
	assert.Equal(t, map[string]any{"value": "first"}, first)

	// A second call with different data must not re-parse; it should
	// return the originally cached result guarded by sync.Once.
	second := UnmarshalEmbeddedJSON(&once, []byte(`{"value":"second"}`), &target)
	assert.Equal(t, map[string]any{"value": "first"}, second)
	assert.Equal(t, first, second)
}
