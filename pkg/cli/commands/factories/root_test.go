package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func TestNewCommand_OrdersGroup(t *testing.T) {
	root := NewCommand(core.BindOptions{})

	ordersCmd, _, err := root.Find([]string{"orders"})
	require.NoError(t, err)
	require.NotNil(t, ordersCmd)
	assert.Contains(t, ordersCmd.Aliases, "order")

	listCmd, _, err := root.Find([]string{"orders", "list"})
	require.NoError(t, err)
	require.NotNil(t, listCmd.Flags().Lookup("factory"))
	require.NotNil(t, listCmd.Flags().Lookup("assignees"))
	require.NotNil(t, listCmd.Flags().Lookup("state"))
	require.NotNil(t, listCmd.Flags().Lookup("result"))
	require.NotNil(t, listCmd.Flags().Lookup("unassigned"))

	describeCmd, _, err := root.Find([]string{"orders", "describe"})
	require.NoError(t, err)
	require.NotNil(t, describeCmd.Flags().Lookup("factory"))

	orderFlag := describeCmd.Flags().Lookup("order")
	require.NotNil(t, orderFlag)
	assert.Empty(t, orderFlag.Deprecated)

	orderIDFlag := describeCmd.Flags().Lookup("order-id")
	require.NotNil(t, orderIDFlag)
	assert.NotEmpty(t, orderIDFlag.Deprecated)
}
