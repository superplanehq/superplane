package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalEmbeddedJSON(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected map[string]any
	}{
		{
			name:  "valid json object",
			input: []byte(`{"key":"value","number":42}`),
			expected: map[string]any{
				"key":    "value",
				"number": float64(42),
			},
		},
		{
			name:     "empty json object",
			input:    []byte(`{}`),
			expected: map[string]any{},
		},
		{
			name:     "invalid json fallback",
			input:    []byte(`invalid json`),
			expected: map[string]any{},
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

	t.Run("sync.Once prevents re-parsing on subsequent calls", func(t *testing.T) {
		var once sync.Once
		var target map[string]any

		firstInput := []byte(`{"initial":"data"}`)
		firstResult := UnmarshalEmbeddedJSON(&once, firstInput, &target)
		expectedFirst := map[string]any{"initial": "data"}
		assert.Equal(t, expectedFirst, firstResult)

		secondInput := []byte(`{"updated":"data"}`)
		secondResult := UnmarshalEmbeddedJSON(&once, secondInput, &target)
		assert.Equal(t, expectedFirst, secondResult)
		assert.Equal(t, expectedFirst, target)
	})
}
