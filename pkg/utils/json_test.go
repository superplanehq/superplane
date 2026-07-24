package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalEmbeddedJSON(t *testing.T) {
	t.Run("unmarshals JSON into the target map", func(t *testing.T) {
		var once sync.Once
		var target map[string]any

		result := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"value","n":1}`), &target)

		require.NotNil(t, result)
		assert.Equal(t, "value", result["key"])
		assert.Equal(t, float64(1), result["n"])
		// The returned map is the same instance pointed to by target.
		assert.Equal(t, target, result)
	})

	t.Run("caches the result and only unmarshals once", func(t *testing.T) {
		var once sync.Once
		var target map[string]any

		first := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"first"}`), &target)
		assert.Equal(t, "first", first["key"])

		// A second call with different data must be ignored because
		// sync.Once has already run - the original result is returned.
		second := UnmarshalEmbeddedJSON(&once, []byte(`{"key":"second"}`), &target)
		assert.Equal(t, "first", second["key"])
	})

	t.Run("invalid JSON results in an empty (non-nil) map", func(t *testing.T) {
		var once sync.Once
		var target map[string]any

		result := UnmarshalEmbeddedJSON(&once, []byte(`not-json`), &target)

		require.NotNil(t, result)
		assert.Empty(t, result)
	})
}
