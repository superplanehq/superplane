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

func TestNewCommand_OrdersCreate(t *testing.T) {
	root := NewCommand(core.BindOptions{})

	createCmd, _, err := root.Find([]string{"orders", "create"})
	require.NoError(t, err)
	require.NotNil(t, createCmd.Flags().Lookup("factory"))
	require.NotNil(t, createCmd.Flags().Lookup("title"))
	require.NotNil(t, createCmd.Flags().Lookup("description"))
	require.NotNil(t, createCmd.Flags().Lookup("file"))
	require.NotNil(t, createCmd.Flags().Lookup("assignee"))
}

func TestNewCommand_OrdersDispatch(t *testing.T) {
	root := NewCommand(core.BindOptions{})

	dispatchCmd, _, err := root.Find([]string{"orders", "dispatch"})
	require.NoError(t, err)
	require.NotNil(t, dispatchCmd.Flags().Lookup("factory"))
	require.NotNil(t, dispatchCmd.Flags().Lookup("line"))

	orderFlag := dispatchCmd.Flags().Lookup("order")
	require.NotNil(t, orderFlag)
	assert.Empty(t, orderFlag.Deprecated)

	orderIDFlag := dispatchCmd.Flags().Lookup("order-id")
	require.NotNil(t, orderIDFlag)
	assert.NotEmpty(t, orderIDFlag.Deprecated)
}

func TestNewCommand_OrdersAssign(t *testing.T) {
	root := NewCommand(core.BindOptions{})

	assignCmd, _, err := root.Find([]string{"orders", "assign"})
	require.NoError(t, err)
	require.NotNil(t, assignCmd.Flags().Lookup("factory"))
	require.NotNil(t, assignCmd.Flags().Lookup("assignee"))

	orderFlag := assignCmd.Flags().Lookup("order")
	require.NotNil(t, orderFlag)
	assert.Empty(t, orderFlag.Deprecated)

	orderIDFlag := assignCmd.Flags().Lookup("order-id")
	require.NotNil(t, orderIDFlag)
	assert.NotEmpty(t, orderIDFlag.Deprecated)
}
