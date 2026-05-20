package display

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Display__Execute(t *testing.T) {
	component := &Display{}

	t.Run("stores message and color in execution metadata", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"message": "Hello",
				"color":   "green",
			},
			ExecutionState: stateCtx,
			Metadata:       metadataCtx,
		})

		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)

		result, ok := metadataCtx.Metadata.(DisplayExecutionResult)
		assert.True(t, ok)
		assert.Equal(t, DisplayExecutionResult{Message: "Hello", Color: "green"}, result)
	})

	t.Run("emits display.executed on the default channel", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"message": "Build completed",
				"color":   "gray",
			},
			ExecutionState: stateCtx,
			Metadata:       metadataCtx,
		})

		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, "display.executed", stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)

		payload := stateCtx.Payloads[0].(map[string]any)
		assert.Equal(t, map[string]any{}, payload["data"])

		result, ok := metadataCtx.Metadata.(DisplayExecutionResult)
		assert.True(t, ok)
		assert.Equal(t, DisplayExecutionResult{Message: "Build completed", Color: "gray"}, result)
	})

	t.Run("replaces prior execution metadata", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{"existing": "value"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"message": "Done",
				"color":   "blue",
			},
			ExecutionState: stateCtx,
			Metadata:       metadataCtx,
		})

		assert.NoError(t, err)

		result, ok := metadataCtx.Metadata.(DisplayExecutionResult)
		assert.True(t, ok)
		assert.Equal(t, DisplayExecutionResult{Message: "Done", Color: "blue"}, result)
	})
}
