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
			data:     []byte(`{"name":"superplane","enabled":true}`),
			expected: map[string]any{"name": "superplane", "enabled": true},
		},
		{
			name:     "nested object",
			data:     []byte(`{"outer":{"inner":42}}`),
			expected: map[string]any{"outer": map[string]any{"inner": float64(42)}},
		},
		{
			name:     "empty object",
			data:     []byte(`{}`),
			expected: map[string]any{},
		},
		{
			name:     "invalid JSON returns empty map",
			data:     []byte(`not valid json`),
			expected: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var once sync.Once
			var target map[string]any

			result := UnmarshalEmbeddedJSON(&once, tc.data, &target)
			assert.Equal(t, tc.expected, result)
			assert.Equal(t, tc.expected, target)
		})
	}
}

func TestUnmarshalEmbeddedJSONOnlyRunsOnce(t *testing.T) {
	var once sync.Once
	var target map[string]any

	first := []byte(`{"value":"first"}`)
	second := []byte(`{"value":"second"}`)

	// First call parses the provided data.
	result := UnmarshalEmbeddedJSON(&once, first, &target)
	assert.Equal(t, map[string]any{"value": "first"}, result)

	// Second call with different data must return the already-parsed
	// result because sync.Once guards the unmarshalling.
	result = UnmarshalEmbeddedJSON(&once, second, &target)
	assert.Equal(t, map[string]any{"value": "first"}, result)
	assert.Equal(t, map[string]any{"value": "first"}, target)
}
