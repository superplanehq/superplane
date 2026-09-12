package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeprecatedFactoryCommandsRemainRegistered(t *testing.T) {
	factoryCmd, _, err := RootCmd.Find([]string{"factory"})
	require.NoError(t, err)
	assert.Contains(t, factoryCmd.Aliases, "factories")

	listCmd, _, err := RootCmd.Find([]string{"factory", "orders", "list"})
	require.NoError(t, err)
	require.NotNil(t, listCmd.Flags().Lookup("workspace"))
	require.NotNil(t, listCmd.Flags().Lookup("factory"))

	describeCmd, _, err := RootCmd.Find([]string{"factory", "order", "describe"})
	require.NoError(t, err)
	require.NotNil(t, describeCmd.Flags().Lookup("task"))
	require.NotNil(t, describeCmd.Flags().Lookup("order"))

	activeCmd, _, err := RootCmd.Find([]string{"factories", "active"})
	require.NoError(t, err)
	require.NotNil(t, activeCmd)

	describeWorkspaceCmd, _, err := RootCmd.Find([]string{"factory", "describe"})
	require.NoError(t, err)
	require.NotNil(t, describeWorkspaceCmd.Flags().Lookup("workspace"))
	require.NotNil(t, describeWorkspaceCmd.Flags().Lookup("factory"))
}
