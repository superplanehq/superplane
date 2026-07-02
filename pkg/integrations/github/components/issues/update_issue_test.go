package issues

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIssue__Setup(t *testing.T) {
	component := UpdateIssue{}

	t.Run("issue number is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueNumber": "", "repository": "hello"},
		})

		require.ErrorContains(t, err, "issue number is required")
	})
}

func Test__UpdateIssue__Execute(t *testing.T) {
	component := UpdateIssue{}

	t.Run("fails when issue number is not a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"issueNumber": "abc",
				"repository":  "hello",
			},
		})

		require.ErrorContains(t, err, "issue number is not a number")
	})

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}
