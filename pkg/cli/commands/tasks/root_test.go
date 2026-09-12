package tasks

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
	require.NotNil(t, listCmd.Flags().Lookup("workspace"))
	require.NotNil(t, listCmd.Flags().Lookup("factory"))
	require.NotNil(t, listCmd.Flags().Lookup("assignees"))
	require.NotNil(t, listCmd.Flags().Lookup("state"))
	require.NotNil(t, listCmd.Flags().Lookup("result"))
	require.NotNil(t, listCmd.Flags().Lookup("unassigned"))

	describeCmd, _, err := root.Find([]string{"describe"})
	require.NoError(t, err)
	require.NotNil(t, describeCmd.Flags().Lookup("workspace"))
	require.NotNil(t, describeCmd.Flags().Lookup("factory"))
	require.NotNil(t, describeCmd.Flags().Lookup("task"))
	require.NotNil(t, describeCmd.Flags().Lookup("order"))

	createCmd, _, err := root.Find([]string{"create"})
	require.NoError(t, err)
	require.NotNil(t, createCmd.Flags().Lookup("workspace"))
	require.NotNil(t, createCmd.Flags().Lookup("title"))
	require.NotNil(t, createCmd.Flags().Lookup("description"))
	require.NotNil(t, createCmd.Flags().Lookup("file"))
	require.NotNil(t, createCmd.Flags().Lookup("assignee"))

	dispatchCmd, _, err := root.Find([]string{"dispatch"})
	require.NoError(t, err)
	require.NotNil(t, dispatchCmd.Flags().Lookup("workspace"))
	require.NotNil(t, dispatchCmd.Flags().Lookup("line"))
	require.NotNil(t, dispatchCmd.Flags().Lookup("task"))

	assignCmd, _, err := root.Find([]string{"assign"})
	require.NoError(t, err)
	require.NotNil(t, assignCmd.Flags().Lookup("workspace"))
	require.NotNil(t, assignCmd.Flags().Lookup("assignee"))
	require.NotNil(t, assignCmd.Flags().Lookup("task"))

	artifactCmd, _, err := root.Find([]string{"artifacts"})
	require.NoError(t, err)
	assert.Contains(t, artifactCmd.Aliases, "artifact")

	artifactAddCmd, _, err := root.Find([]string{"artifacts", "add"})
	require.NoError(t, err)
	require.NotNil(t, artifactAddCmd.Flags().Lookup("workspace"))
	require.NotNil(t, artifactAddCmd.Flags().Lookup("task"))
	require.NotNil(t, artifactAddCmd.Flags().Lookup("type"))

	artifactListCmd, _, err := root.Find([]string{"artifacts", "list"})
	require.NoError(t, err)
	require.NotNil(t, artifactListCmd.Flags().Lookup("workspace"))
	require.NotNil(t, artifactListCmd.Flags().Lookup("factory"))
	require.NotNil(t, artifactListCmd.Flags().Lookup("task"))
	require.NotNil(t, artifactListCmd.Flags().Lookup("order"))
}

func TestNewCommand_Aliases(t *testing.T) {
	root := NewCommand(core.BindOptions{})
	assert.Contains(t, root.Aliases, "task")
}

func TestNewCommand_DeprecatedFactoryAndOrderFlags(t *testing.T) {
	root := NewCommand(core.BindOptions{})

	listCmd, _, err := root.Find([]string{"list"})
	require.NoError(t, err)
	require.NoError(t, listCmd.ParseFlags([]string{"--factory", "shipping"}))
	assert.Equal(t, "shipping", listCmd.Flags().Lookup("factory").Value.String())
	assert.Equal(t, "shipping", listCmd.Flags().Lookup("workspace").Value.String())

	describeCmd, _, err := root.Find([]string{"describe"})
	require.NoError(t, err)
	require.NoError(t, describeCmd.ParseFlags([]string{"--order", "tid-1"}))
	assert.Equal(t, "tid-1", describeCmd.Flags().Lookup("order").Value.String())
	assert.Equal(t, "tid-1", describeCmd.Flags().Lookup("task").Value.String())
}
