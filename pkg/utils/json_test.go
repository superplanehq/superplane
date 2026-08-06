package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalEmbeddedJSON(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		expected map[string]any
	}{
		{
			name:     "simple object",
			data:     []byte(`{"key":"value"}`),
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "nested object",
			data:     []byte(`{"a":1,"b":{"c":true}}`),
			expected: map[string]any{"a": float64(1), "b": map[string]any{"c": true}},
		},
		{
			name:     "empty object",
			data:     []byte(`{}`),
			expected: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var once sync.Once
			var target map[string]any

			result := UnmarshalEmbeddedJSON(&once, tc.data, &target)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUnmarshalEmbeddedJSON_OnlyParsesOnce(t *testing.T) {
	var once sync.Once
	var target map[string]any

	first := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"first"}`), &target)
	assert.Equal(t, map[string]any{"key": "first"}, first)

	second := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"second"}`), &target)

	assert.Equal(t, map[string]any{"key": "first"}, second)
	assert.Equal(t, first, second)
}
