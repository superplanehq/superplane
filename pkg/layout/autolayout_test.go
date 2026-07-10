package layout

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAutoLayout(t *testing.T) {
	autoLayout := DefaultAutoLayout()
	assert.Equal(t, AlgorithmHorizontal, autoLayout.Algorithm)
	assert.Equal(t, ScopeFullCanvas, autoLayout.Scope)
}

func TestParseAutoLayout_Disable(t *testing.T) {
	autoLayout, err := ParseAutoLayout("disable", "", nil)
	require.NoError(t, err)
	assert.Nil(t, autoLayout)

	_, err = ParseAutoLayout("off", "full-canvas", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used when --auto-layout disables layout")

	_, err = ParseAutoLayout("none", "", []string{"node-1"})
	require.Error(t, err)
}

func TestParseAutoLayout_Horizontal(t *testing.T) {
	autoLayout, err := ParseAutoLayout("", "", nil)
	require.NoError(t, err)
	require.NotNil(t, autoLayout)
	assert.Equal(t, AlgorithmHorizontal, autoLayout.Algorithm)
	assert.Empty(t, autoLayout.Scope)
	assert.Empty(t, autoLayout.NodeIDs)

	autoLayout, err = ParseAutoLayout("horizontal", "", nil)
	require.NoError(t, err)
	require.NotNil(t, autoLayout)
	assert.Equal(t, AlgorithmHorizontal, autoLayout.Algorithm)
}

func TestParseAutoLayout_UnsupportedValue(t *testing.T) {
	_, err := ParseAutoLayout("vertical", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported auto layout")
}

func TestParseAutoLayout_Scope(t *testing.T) {
	autoLayout, err := ParseAutoLayout("horizontal", "full-canvas", nil)
	require.NoError(t, err)
	assert.Equal(t, ScopeFullCanvas, autoLayout.Scope)

	autoLayout, err = ParseAutoLayout("horizontal", "connected-component", nil)
	require.NoError(t, err)
	assert.Equal(t, ScopeConnectedComponent, autoLayout.Scope)

	autoLayout, err = ParseAutoLayout("horizontal", "full", nil)
	require.NoError(t, err)
	assert.Equal(t, ScopeFullCanvas, autoLayout.Scope)

	autoLayout, err = ParseAutoLayout("horizontal", "connected", nil)
	require.NoError(t, err)
	assert.Equal(t, ScopeConnectedComponent, autoLayout.Scope)

	_, err = ParseAutoLayout("horizontal", "selection", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported auto layout scope")
}

func TestParseAutoLayout_NodeIDs(t *testing.T) {
	autoLayout, err := ParseAutoLayout("horizontal", "", []string{" node-a ", "node-b", "node-a", ""})
	require.NoError(t, err)
	assert.Equal(t, []string{"node-a", "node-b"}, autoLayout.NodeIDs)
}
