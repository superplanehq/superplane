package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalEmbeddedJSON(t *testing.T) {
	testCases := []struct {
		expected map[string]any
		name     string
		data     []byte
	}{
		{
			name:     "valid JSON object",
			data:     []byte(`{"key":"value","count":2}`),
			expected: map[string]any{"key": "value", "count": float64(2)},
		},
		{
			name:     "empty JSON object",
			data:     []byte(`{}`),
			expected: map[string]any{},
		},
		{
			name:     "nil data yields empty map",
			data:     nil,
			expected: map[string]any{},
		},
		{
			name:     "empty data yields empty map",
			data:     []byte(``),
			expected: map[string]any{},
		},
		{
			name:     "invalid JSON yields empty map",
			data:     []byte(`{not valid json`),
			expected: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var once sync.Once
			var target map[string]any

			result := UnmarshalEmbeddedJSON(&once, tc.data, &target)

			assert.NotNil(t, result)
			assert.Equal(t, tc.expected, result)
			// The returned map is the same one stored in target.
			assert.Equal(t, tc.expected, target)
		})
	}
}

func TestUnmarshalEmbeddedJSONCachesFirstResult(t *testing.T) {
	var once sync.Once
	var target map[string]any

	first := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"first"}`), &target)
	assert.Equal(t, map[string]any{"key": "first"}, first)

	// Second call with different data must return the originally cached value,
	// because sync.Once only runs the unmarshal a single time.
	second := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"second"}`), &target)
	assert.Equal(t, map[string]any{"key": "first"}, second)
}
