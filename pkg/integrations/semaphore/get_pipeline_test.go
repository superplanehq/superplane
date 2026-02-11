package semaphore

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__GetPipeline__Setup(t *testing.T) {
	component := &GetPipeline{}

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "00000000-0000-0000-0000-000000000000",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing pipeline ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "pipeline ID is required")
	})

	t.Run("empty pipeline ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "",
			},
		})

		require.ErrorContains(t, err, "pipeline ID is required")
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})
}
