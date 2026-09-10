package workspaces

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func TestNewCommand_Hierarchy(t *testing.T) {
	root := NewCommand(core.BindOptions{})

	activeCmd, _, err := root.Find([]string{"active"})
	require.NoError(t, err)
	require.NotNil(t, activeCmd)
	assert.Contains(t, activeCmd.Use, "active")
}

func TestNewCommand_Aliases(t *testing.T) {
	root := NewCommand(core.BindOptions{})
	assert.Contains(t, root.Aliases, "workspaces")
}
