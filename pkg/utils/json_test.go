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
			name:     "valid json",
			data:     []byte(`{"name":"superplane","stars":4400}`),
			expected: map[string]any{"name": "superplane", "stars": float64(4400)},
		},
		{
			name:     "nested object",
			data:     []byte(`{"config":{"enabled":true}}`),
			expected: map[string]any{"config": map[string]any{"enabled": true}},
		},
		{
			name:     "invalid json",
			data:     []byte("not-json"),
			expected: map[string]any{},
		},
		{
			name:     "empty data",
			data:     nil,
			expected: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var once sync.Once
			target := map[string]any{}
			result := UnmarshalEmbeddedJSON(&once, tc.data, &target)
			assert.Equal(t, tc.expected, result)
		})
	}

	t.Run("only unmarshals on the first call", func(t *testing.T) {
		var once sync.Once
		target := map[string]any{}
		first := UnmarshalEmbeddedJSON(&once, []byte(`{"a":1}`), &target)
		second := UnmarshalEmbeddedJSON(&once, []byte(`{"b":2}`), &target)
		assert.Equal(t, map[string]any{"a": float64(1)}, first)
		assert.Equal(t, first, second)
	})
}
