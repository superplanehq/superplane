package railway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__TriggerDeploy__Setup(t *testing.T) {
	component := TriggerDeploy{}

	t.Run("project is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "", "service": "srv-123", "environment": "env-123"},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("service is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "proj-123", "service": "", "environment": "env-123"},
		})

		require.ErrorContains(t, err, "service is required")
	})

	t.Run("environment is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "proj-123", "service": "srv-123", "environment": ""},
		})

		require.ErrorContains(t, err, "environment is required")
	})
}

func Test__TriggerDeploy__Configuration(t *testing.T) {
	component := TriggerDeploy{}
	config := component.Configuration()

	t.Run("has three required fields", func(t *testing.T) {
		require.Len(t, config, 3)

		// Verify project field
		assert.Equal(t, "project", config[0].Name)
		assert.Equal(t, "Project", config[0].Label)
		assert.True(t, config[0].Required)
		assert.Equal(t, "integration-resource", config[0].Type)
		assert.Equal(t, "project", config[0].TypeOptions.Resource.Type)

		// Verify service field
		assert.Equal(t, "service", config[1].Name)
		assert.Equal(t, "Service", config[1].Label)
		assert.True(t, config[1].Required)
		assert.Equal(t, "integration-resource", config[1].Type)
		assert.Equal(t, "service", config[1].TypeOptions.Resource.Type)
		require.Len(t, config[1].TypeOptions.Resource.Parameters, 1)
		assert.Equal(t, "projectId", config[1].TypeOptions.Resource.Parameters[0].Name)
		assert.Equal(t, "project", config[1].TypeOptions.Resource.Parameters[0].ValueFrom.Field)

		// Verify environment field
		assert.Equal(t, "environment", config[2].Name)
		assert.Equal(t, "Environment", config[2].Label)
		assert.True(t, config[2].Required)
		assert.Equal(t, "integration-resource", config[2].Type)
		assert.Equal(t, "environment", config[2].TypeOptions.Resource.Type)
		require.Len(t, config[2].TypeOptions.Resource.Parameters, 1)
		assert.Equal(t, "projectId", config[2].TypeOptions.Resource.Parameters[0].Name)
		assert.Equal(t, "project", config[2].TypeOptions.Resource.Parameters[0].ValueFrom.Field)
	})
}
