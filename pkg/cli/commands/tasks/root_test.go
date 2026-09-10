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
	require.NotNil(t, listCmd.Flags().Lookup("assignees"))
	require.NotNil(t, listCmd.Flags().Lookup("state"))
	require.NotNil(t, listCmd.Flags().Lookup("result"))
	require.NotNil(t, listCmd.Flags().Lookup("unassigned"))

	describeCmd, _, err := root.Find([]string{"describe"})
	require.NoError(t, err)
	require.NotNil(t, describeCmd.Flags().Lookup("workspace"))
	require.NotNil(t, describeCmd.Flags().Lookup("task"))

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
	require.NotNil(t, artifactListCmd.Flags().Lookup("task"))
}

func TestNewCommand_Aliases(t *testing.T) {
	root := NewCommand(core.BindOptions{})
	assert.Contains(t, root.Aliases, "task")
}
