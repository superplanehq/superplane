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
			name: "valid JSON object",
			data: []byte(`{"name":"superplane","count":3,"enabled":true}`),
			expected: map[string]any{
				"name":    "superplane",
				"count":   float64(3),
				"enabled": true,
			},
		},
		{
			name: "valid JSON with nested object",
			data: []byte(`{"outer":{"inner":"value"}}`),
			expected: map[string]any{
				"outer": map[string]any{"inner": "value"},
			},
		},
		{
			name:     "empty object",
			data:     []byte(`{}`),
			expected: map[string]any{},
		},
		{
			name:     "invalid JSON yields empty non-nil map",
			data:     []byte(`{invalid`),
			expected: map[string]any{},
		},
		{
			name:     "empty input yields empty non-nil map",
			data:     []byte(``),
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
		})
	}

	t.Run("sync.Once caches the first result", func(t *testing.T) {
		var once sync.Once
		var target map[string]any

		first := UnmarshalEmbeddedJSON(&once, []byte(`{"value":"first"}`), &target)
		assert.Equal(t, map[string]any{"value": "first"}, first)

		// A second call with different data should return the originally
		// cached value because once.Do only runs the parsing once.
		second := UnmarshalEmbeddedJSON(&once, []byte(`{"value":"second"}`), &target)
		assert.Equal(t, map[string]any{"value": "first"}, second)
	})
}
