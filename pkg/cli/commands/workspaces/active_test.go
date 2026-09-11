package workspaces

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

func TestActiveCommand_PrintActive(t *testing.T) {
	workspaceID := "11111111-1111-1111-1111-111111111111"

	t.Run("text output", func(t *testing.T) {
		stdout := bytes.NewBuffer(nil)
		renderer, err := core.NewRenderer("text", stdout)
		require.NoError(t, err)

		ctx := core.CommandContext{
			Cmd:      &cobra.Command{},
			Renderer: renderer,
			Config:   &cli.FakeConfig{ActiveWorkspace: workspaceID},
		}
		require.NoError(t, (&activeCommand{}).printActive(ctx))
		assert.Equal(t, workspaceID+"\n", stdout.String())
	})

	t.Run("json output", func(t *testing.T) {
		stdout := bytes.NewBuffer(nil)
		renderer, err := core.NewRenderer("json", stdout)
		require.NoError(t, err)

		ctx := core.CommandContext{
			Cmd:      &cobra.Command{},
			Renderer: renderer,
			Config:   &cli.FakeConfig{ActiveWorkspace: workspaceID},
		}
		require.NoError(t, (&activeCommand{}).printActive(ctx))
		assert.Contains(t, stdout.String(), workspaceID)
		assert.Contains(t, stdout.String(), `"id"`)
	})

	t.Run("errors when no active workspace", func(t *testing.T) {
		stdout := bytes.NewBuffer(nil)
		renderer, err := core.NewRenderer("text", stdout)
		require.NoError(t, err)

		ctx := core.CommandContext{
			Cmd:      &cobra.Command{},
			Renderer: renderer,
			Config:   &cli.FakeConfig{},
		}
		err = (&activeCommand{}).printActive(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no active workspace")
	})
}
