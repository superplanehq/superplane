package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetWorkflowUsage__Setup(t *testing.T) {
	component := GetWorkflowUsage{}
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}

	t.Run("passes when repositories are empty", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})
		require.NoError(t, err)
	})

	t.Run("fails when repository is not accessible", func(t *testing.T) {
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
		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})
}

func Test__GetWorkflowUsage__Execute__Validation(t *testing.T) {
	component := GetWorkflowUsage{}
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}

	makeCtx := func(configuration map[string]any) core.ExecutionContext {
		return core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					InstallationID: "12345",
					Owner:          "testhq",
					Repositories:   []Repository{helloRepo},
					GitHubApp:      GitHubAppMetadata{ID: 12345},
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  configuration,
		}
	}

	t.Run("returns error when month is out of range", func(t *testing.T) {
		err := component.Execute(makeCtx(map[string]any{
			"month": "13",
		}))
		require.ErrorContains(t, err, "month must be between 1 and 12")
	})

	t.Run("returns error when day is out of range", func(t *testing.T) {
		err := component.Execute(makeCtx(map[string]any{
			"day": "0",
		}))
		require.ErrorContains(t, err, "day must be between 1 and 31")
	})

	t.Run("returns error when year is invalid", func(t *testing.T) {
		err := component.Execute(makeCtx(map[string]any{
			"year": "abcd",
		}))
		require.ErrorContains(t, err, "year must be a valid number")
	})
}
