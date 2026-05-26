package manualrun

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoerceParamValue(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		out, err := coerceParamValue(Definition{Type: ParamTypeString}, "hello")
		require.NoError(t, err)
		assert.Equal(t, "hello", out)
	})

	t.Run("string rejects non-string", func(t *testing.T) {
		_, err := coerceParamValue(Definition{Type: ParamTypeString}, 42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected string")
	})

	t.Run("number from float64", func(t *testing.T) {
		out, err := coerceParamValue(Definition{Type: ParamTypeNumber}, float64(3.5))
		require.NoError(t, err)
		assert.Equal(t, float64(3.5), out)
	})

	t.Run("number from int types", func(t *testing.T) {
		cases := []struct {
			in   any
			want float64
		}{
			{float32(2), 2},
			{int(3), 3},
			{int64(4), 4},
			{int32(5), 5},
		}
		for _, tc := range cases {
			out, err := coerceParamValue(Definition{Type: ParamTypeNumber}, tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, out)
		}
	})

	t.Run("number rejects non-number", func(t *testing.T) {
		_, err := coerceParamValue(Definition{Type: ParamTypeNumber}, "42")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected number")
	})

	t.Run("boolean", func(t *testing.T) {
		out, err := coerceParamValue(Definition{Type: ParamTypeBoolean}, true)
		require.NoError(t, err)
		assert.Equal(t, true, out)
	})

	t.Run("boolean rejects non-boolean", func(t *testing.T) {
		_, err := coerceParamValue(Definition{Type: ParamTypeBoolean}, "true")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected boolean")
	})

	t.Run("select", func(t *testing.T) {
		def := Definition{Type: ParamTypeSelect, Values: []string{"2 vCPU", "4 vCPU"}}
		out, err := coerceParamValue(def, "4 vCPU")
		require.NoError(t, err)
		assert.Equal(t, "4 vCPU", out)
	})

	t.Run("select rejects unknown option", func(t *testing.T) {
		def := Definition{Type: ParamTypeSelect, Values: []string{"2 vCPU", "4 vCPU"}}
		_, err := coerceParamValue(def, "16 vCPU")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not one of")
	})

	t.Run("select rejects non-string", func(t *testing.T) {
		def := Definition{Type: ParamTypeSelect, Values: []string{"a"}}
		_, err := coerceParamValue(def, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected string")
	})

	t.Run("unsupported type", func(t *testing.T) {
		_, err := coerceParamValue(Definition{Type: ParamType("json")}, "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type")
	})
}

func TestCoerceStaticValue(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		out, err := coerceStaticValue("hello", "world")
		require.NoError(t, err)
		assert.Equal(t, "world", out)
	})

	t.Run("string rejects non-string", func(t *testing.T) {
		_, err := coerceStaticValue("hello", 42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected string")
	})

	t.Run("number from float64", func(t *testing.T) {
		out, err := coerceStaticValue(float64(1), float64(5))
		require.NoError(t, err)
		assert.Equal(t, float64(5), out)
	})

	t.Run("number from int types", func(t *testing.T) {
		out, err := coerceStaticValue(float64(1), int(9))
		require.NoError(t, err)
		assert.Equal(t, float64(9), out)

		out, err = coerceStaticValue(float64(1), int64(10))
		require.NoError(t, err)
		assert.Equal(t, float64(10), out)

		out, err = coerceStaticValue(float64(1), float32(11))
		require.NoError(t, err)
		assert.Equal(t, float64(11), out)
	})

	t.Run("number rejects non-number", func(t *testing.T) {
		_, err := coerceStaticValue(float64(1), "5")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected number")
	})

	t.Run("boolean", func(t *testing.T) {
		out, err := coerceStaticValue(false, true)
		require.NoError(t, err)
		assert.Equal(t, true, out)
	})

	t.Run("boolean rejects non-boolean", func(t *testing.T) {
		_, err := coerceStaticValue(false, "true")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected boolean")
	})

	t.Run("passthrough for other types", func(t *testing.T) {
		value := map[string]any{"nested": true}
		out, err := coerceStaticValue(value, map[string]any{"nested": false})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"nested": false}, out)
	})
}

func TestStringsJoin(t *testing.T) {
	assert.Equal(t, "", stringsJoin(nil, ", "))
	assert.Equal(t, "a", stringsJoin([]string{"a"}, ", "))
	assert.Equal(t, "a, b, c", stringsJoin([]string{"a", "b", "c"}, ", "))
}
