package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalEmbeddedJSON(t *testing.T) {
	t.Run("unmarshals valid JSON into the target", func(t *testing.T) {
		var once sync.Once
		var target map[string]any

		result := UnmarshalEmbeddedJSON(&once, []byte(`{"name":"superplane","count":2}`), &target)

		assert.Equal(t, "superplane", result["name"])
		assert.EqualValues(t, 2, result["count"])
		assert.Equal(t, target, result)
	})

	t.Run("only unmarshals once even when called repeatedly", func(t *testing.T) {
		var once sync.Once
		var target map[string]any

		first := UnmarshalEmbeddedJSON(&once, []byte(`{"value":"first"}`), &target)
		assert.Equal(t, "first", first["value"])

		second := UnmarshalEmbeddedJSON(&once, []byte(`{"value":"second"}`), &target)
		assert.Equal(t, "first", second["value"], "second call must reuse the first result")
	})

	t.Run("invalid JSON yields an initialized empty map", func(t *testing.T) {
		var once sync.Once
		var target map[string]any

		result := UnmarshalEmbeddedJSON(&once, []byte(`not json`), &target)

		require.NotNil(t, result)
		assert.Empty(t, result)
	})
}
