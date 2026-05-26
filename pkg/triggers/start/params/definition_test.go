package params

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefinition_success(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		def, err := NewDefinition("body.name", ParamTypeString, "Name", "machine-1", false, nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "body.name", def.Path)
		assert.Equal(t, ParamTypeString, def.Type)
		assert.Equal(t, "Name", def.Title)
		assert.Equal(t, "machine-1", def.Default)
		assert.False(t, def.Required)
		assert.Empty(t, def.Values)
	})

	t.Run("number", func(t *testing.T) {
		def, err := NewDefinition("count", ParamTypeNumber, "", float64(42), false, nil, 0)
		require.NoError(t, err)
		assert.Equal(t, ParamTypeNumber, def.Type)
		assert.Equal(t, float64(42), def.Default)
	})

	t.Run("boolean", func(t *testing.T) {
		def, err := NewDefinition("enabled", ParamTypeBoolean, "", true, true, nil, 0)
		require.NoError(t, err)
		assert.Equal(t, ParamTypeBoolean, def.Type)
		assert.Equal(t, true, def.Default)
		assert.True(t, def.Required)
	})

	t.Run("select", func(t *testing.T) {
		def, err := NewDefinition("body.size", ParamTypeSelect, "Size", "2 vCPU", true, []string{"2 vCPU", "4 vCPU"}, 0)
		require.NoError(t, err)
		assert.Equal(t, ParamTypeSelect, def.Type)
		assert.Equal(t, "2 vCPU", def.Default)
		assert.Equal(t, []string{"2 vCPU", "4 vCPU"}, def.Values)
	})

	t.Run("select without default", func(t *testing.T) {
		def, err := NewDefinition("body.size", ParamTypeSelect, "Size", nil, true, []string{"a", "b"}, 0)
		require.NoError(t, err)
		assert.Nil(t, def.Default)
	})
}

func TestNewDefinition_rejectsInvalid(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		_, err := NewDefinition("", ParamTypeString, "", nil, false, nil, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path is required")
	})

	t.Run("missing type", func(t *testing.T) {
		_, err := NewDefinition("body.name", "", "", nil, false, nil, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing type")
	})

	t.Run("unsupported type", func(t *testing.T) {
		_, err := NewDefinition("body.name", ParamType("json"), "", nil, false, nil, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type")
	})

	t.Run("select without values", func(t *testing.T) {
		_, err := NewDefinition("body.size", ParamTypeSelect, "", nil, false, nil, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "select param requires values")
	})

	t.Run("select default not in values", func(t *testing.T) {
		_, err := NewDefinition("body.size", ParamTypeSelect, "", "8 vCPU", false, []string{"2 vCPU", "4 vCPU"}, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not one of the select values")
	})

	t.Run("select default not a string", func(t *testing.T) {
		_, err := NewDefinition("body.size", ParamTypeSelect, "", 42, false, []string{"a", "b"}, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "select default must be a string")
	})

	t.Run("values on non-select", func(t *testing.T) {
		_, err := NewDefinition("body.name", ParamTypeString, "", nil, false, []string{"a"}, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "values is only valid for select params")
	})
}
