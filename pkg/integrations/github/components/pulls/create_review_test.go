package pulls

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateReview__Execute__Validation(t *testing.T) {
	component := CreateReview{}

	t.Run("body is conditionally required for request changes and comment", func(t *testing.T) {
		fields := component.Configuration()
		var bodyFieldFound bool
		for _, f := range fields {
			if f.Name != "body" {
				continue
			}

			bodyFieldFound = true
			require.Len(t, f.RequiredConditions, 1)
			require.Equal(t, "event", f.RequiredConditions[0].Field)
			require.ElementsMatch(t, []string{"REQUEST_CHANGES", "COMMENT"}, f.RequiredConditions[0].Values)
		}

		require.True(t, bodyFieldFound, "expected to find body field in configuration")
	})

	t.Run("pull number is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"repository": "hello", "pullNumber": "", "event": "APPROVE"},
		})
		require.ErrorContains(t, err, "pull number is required")
	})

	t.Run("pull number must be a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"repository": "hello", "pullNumber": "abc", "event": "APPROVE"},
		})
		require.ErrorContains(t, err, "pull number is not a number")
	})

	t.Run("event must be valid", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"repository": "hello", "pullNumber": "1", "event": "NOPE"},
		})
		require.ErrorContains(t, err, "invalid event")
	})

	t.Run("body is required for request changes", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"repository": "hello", "pullNumber": "1", "event": "REQUEST_CHANGES", "body": ""},
		})
		require.ErrorContains(t, err, "body is required for REQUEST_CHANGES")
	})

	t.Run("body is required for comment", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"repository": "hello", "pullNumber": "1", "event": "COMMENT"},
		})
		require.ErrorContains(t, err, "body is required for COMMENT")
	})
}
