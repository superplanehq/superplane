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
			name:     "nested object with mixed types",
			data:     []byte(`{"name":"test","count":2,"enabled":true}`),
			expected: map[string]any{"name": "test", "count": float64(2), "enabled": true},
		},
		{
			name:     "empty object",
			data:     []byte(`{}`),
			expected: map[string]any{},
		},
		{
			name:     "invalid JSON returns empty map",
			data:     []byte(`not-json`),
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

func TestUnmarshalEmbeddedJSON_OnceBehavior(t *testing.T) {
	var once sync.Once
	var target map[string]any

	first := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"first"}`), &target)
	assert.Equal(t, map[string]any{"key": "first"}, first)

	// A second call with different data must not re-parse; it returns the
	// already-parsed result thanks to the sync.Once guard.
	second := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"second"}`), &target)
	assert.Equal(t, map[string]any{"key": "first"}, second)
	assert.Equal(t, first, second)
}
