package workspaces

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func TestNewCommand_Hierarchy(t *testing.T) {
	root := NewCommand(core.BindOptions{})

	listCmd, _, err := root.Find([]string{"list"})
	require.NoError(t, err)
	require.NotNil(t, listCmd)
	assert.Contains(t, listCmd.Use, "list")

	describeCmd, _, err := root.Find([]string{"describe"})
	require.NoError(t, err)
	require.NotNil(t, describeCmd)
	assert.Contains(t, describeCmd.Use, "describe")
	require.NotNil(t, describeCmd.Flags().Lookup("workspace"))
	require.NotNil(t, describeCmd.Flags().Lookup("factory"))

	activeCmd, _, err := root.Find([]string{"active"})
	require.NoError(t, err)
	require.NotNil(t, activeCmd)
	assert.Contains(t, activeCmd.Use, "active")
}

func TestNewCommand_Aliases(t *testing.T) {
	root := NewCommand(core.BindOptions{})
	assert.Contains(t, root.Aliases, "workspaces")
}
