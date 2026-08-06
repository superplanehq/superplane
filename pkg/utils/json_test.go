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
			input:    []byte(`{"key":"value","count":1}`),
			expected: map[string]any{"key": "value", "count": float64(1)},
			name:     "simple flat object",
		},
		{
			input:    []byte(`{"a":{"b":"c"}}`),
			expected: map[string]any{"a": map[string]any{"b": "c"}},
			name:     "nested object",
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
			assert.Equal(t, tc.expected, target)
		})
	}
}

func TestUnmarshalEmbeddedJSON_OnlyParsesOnce(t *testing.T) {
	var once sync.Once
	var target map[string]any

	first := UnmarshalEmbeddedJSON(&once, []byte(`{"first":"value"}`), &target)
	assert.Equal(t, map[string]any{"first": "value"}, first)

	second := UnmarshalEmbeddedJSON(&once, []byte(`{"second":"other"}`), &target)
	assert.Equal(t, map[string]any{"first": "value"}, second)
}
