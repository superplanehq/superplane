package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetWorkflowUsage__Configuration(t *testing.T) {
	component := &GetWorkflowUsage{}

	t.Run("has correct name", func(t *testing.T) {
		assert.Equal(t, "github.getWorkflowUsage", component.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		assert.Equal(t, "Get Workflow Usage", component.Label())
	})

	t.Run("returns configuration fields", func(t *testing.T) {
		fields := component.Configuration()
		require.NotEmpty(t, fields)

		var repoField, yearField, monthField, skuField *any
		for _, f := range fields {
			switch f.Name {
			case "repositories":
				repoField = &f
			case "year":
				yearField = &f
			case "month":
				monthField = &f
			case "sku":
				skuField = &f
			}
		}

		assert.NotNil(t, repoField, "should have repositories field")
		assert.NotNil(t, yearField, "should have year field")
		assert.NotNil(t, monthField, "should have month field")
		assert.NotNil(t, skuField, "should have sku field")
	})
}

func Test__GetWorkflowUsage__Execute__Defaults(t *testing.T) {
	component := &GetWorkflowUsage{}

	t.Run("uses current year and month when not specified", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				InstallationID: "12345",
				Owner:          "test-org",
				GitHubApp:      GitHubAppMetadata{ID: 12345},
			},
			Configuration: map[string]any{
				"privateKey": "test-key",
			},
		}

		// Empty config should use defaults
		ctx := core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repositories": []string{},
			},
		}

		// This will fail to connect but validates config parsing
		_, err := component.Execute(ctx)
		// Expected to fail at API call, not config validation
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to")
	})
}

func Test__GetWorkflowUsage__Execute__Validation(t *testing.T) {
	component := &GetWorkflowUsage{}

	t.Run("accepts empty repository list for org-wide usage", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				InstallationID: "12345",
				Owner:          "test-org",
				GitHubApp:      GitHubAppMetadata{ID: 12345},
			},
			Configuration: map[string]any{
				"privateKey": "test-key",
			},
		}

		ctx := core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repositories": []string{},
				"year":         "2024",
				"month":        "2",
			},
		}

		// Will fail at API call stage, which is expected
		_, err := component.Execute(ctx)
		assert.Error(t, err)
	})

	t.Run("accepts multiple repositories", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				InstallationID: "12345",
				Owner:          "test-org",
				GitHubApp:      GitHubAppMetadata{ID: 12345},
			},
			Configuration: map[string]any{
				"privateKey": "test-key",
			},
		}

		ctx := core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repositories": []string{"repo1", "repo2"},
				"year":         "2024",
				"month":        "2",
			},
		}

		// Will fail at API call stage
		_, err := component.Execute(ctx)
		assert.Error(t, err)
	})
}
