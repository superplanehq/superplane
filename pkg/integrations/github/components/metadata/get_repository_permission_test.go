package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetRepositoryPermission__Execute(t *testing.T) {
	component := GetRepositoryPermission{}

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("fails when metadata decode fails", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: "not a valid metadata",
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"username":   "octocat",
			},
		})

		require.ErrorContains(t, err, "failed to decode metadata")
	})
}
