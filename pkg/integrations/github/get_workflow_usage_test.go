package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetWorkflowUsage__Execute__Validation(t *testing.T) {
	component := GetWorkflowUsage{}

	t.Run("year must be a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"year": "abc"},
		})
		require.ErrorContains(t, err, "year is not a number")
	})

	t.Run("year must be > 0", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"year": "0"},
		})
		require.ErrorContains(t, err, "year must be greater than 0")
	})

	t.Run("month must be a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"year": "2026", "month": "abc"},
		})
		require.ErrorContains(t, err, "month is not a number")
	})

	t.Run("month must be between 1 and 12", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"year": "2026", "month": "13"},
		})
		require.ErrorContains(t, err, "month must be between 1 and 12")
	})

	t.Run("day requires month", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"year": "2026", "day": "8"},
		})
		require.ErrorContains(t, err, "month is required when day is set")
	})

	t.Run("month/day require year", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"month": "2"},
		})
		require.ErrorContains(t, err, "year is required when month/day is set")
	})
}

func Test__GetWorkflowUsage__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := GetWorkflowUsage{}

	t.Run("no repositories selected is ok", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repositories": []string{}},
		}))
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
			Configuration: map[string]any{"repositories": []string{"world"}},
		})
		require.ErrorContains(t, err, "repository world is not accessible")
	})

	t.Run("metadata is set successfully (first repo)", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repositories": []string{"hello"}},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
	})
}
