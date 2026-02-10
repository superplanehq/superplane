package github

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

func Test__CreateReview__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := CreateReview{}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set successfully", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
	})
}
