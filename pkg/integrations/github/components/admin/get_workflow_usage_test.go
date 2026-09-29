package admin

import (
	"testing"

	gh "github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetWorkflowUsage__Setup(t *testing.T) {
	helloRepo := common.Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	worldRepo := common.Repository{ID: 123457, Name: "world", URL: "https://github.com/testhq/world"}
	component := GetWorkflowUsage{}

	t.Run("setup succeeds with no configuration", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		nodeMetadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      nodeMetadataCtx,
			Configuration: map[string]any{},
		})

		require.NoError(t, err)
	})

	t.Run("setup succeeds with empty repositories", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				Repositories: []common.Repository{helloRepo, worldRepo},
			},
		}
		nodeMetadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      nodeMetadataCtx,
			Configuration: map[string]any{"repositories": []string{}},
		})

		require.NoError(t, err)
	})

	t.Run("setup succeeds with valid repositories and stores metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				Repositories: []common.Repository{helloRepo, worldRepo},
			},
		}
		nodeMetadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      nodeMetadataCtx,
			Configuration: map[string]any{"repositories": []string{"hello", "world"}},
		})

		require.NoError(t, err)
		// Verify metadata was stored with proper structure
		metadata := nodeMetadataCtx.Get()
		require.NotNil(t, metadata)

		// Should be GetWorkflowUsageMetadata struct
		var usageMetadata GetWorkflowUsageMetadata
		err = mapstructure.Decode(metadata, &usageMetadata)
		require.NoError(t, err)

		require.Len(t, usageMetadata.Repositories, 2)
		require.Equal(t, "hello", usageMetadata.Repositories[0].Name)
		require.Equal(t, "world", usageMetadata.Repositories[1].Name)
		require.Equal(t, int64(123456), usageMetadata.Repositories[0].ID)
		require.Equal(t, int64(123457), usageMetadata.Repositories[1].ID)
		require.Equal(t, "https://github.com/testhq/hello", usageMetadata.Repositories[0].URL)
		require.Equal(t, "https://github.com/testhq/world", usageMetadata.Repositories[1].URL)
	})

	t.Run("setup uses catalog repositories for a hosted binding", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		organization, err := models.CreateOrganization("org-"+uuid.NewString(), "")
		require.NoError(t, err)
		require.NoError(t, models.UpsertVCSProviderInstallation(database.Conn(), &models.VCSProviderInstallation{
			Provider:       models.ProviderGitHub,
			InstallationID: 501,
			AccountLogin:   "testhq",
		}))
		integration, err := models.FindOrCreateVCSProviderBinding(
			database.Conn(),
			organization.ID,
			models.ProviderGitHub,
			501,
			"testhq",
		)
		require.NoError(t, err)
		require.NoError(t, models.ReplaceVCSProviderRepositories(
			database.Conn(),
			models.ProviderGitHub,
			501,
			[]models.VCSProviderRepository{{RepositoryID: 123456, FullName: "testhq/hello"}},
		))
		require.NoError(t, models.GrantVCSProviderBindingRepository(
			database.Conn(),
			integration.ID,
			models.ProviderGitHub,
			123456,
		))

		nodeMetadataCtx := &contexts.MetadataContext{}
		err = component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				IntegrationID: integration.ID.String(),
				Metadata:      common.Metadata{HostedApp: true},
			},
			Metadata:      nodeMetadataCtx,
			Configuration: map[string]any{"repositories": []string{"hello"}},
		})

		require.NoError(t, err)
		var usageMetadata GetWorkflowUsageMetadata
		require.NoError(t, mapstructure.Decode(nodeMetadataCtx.Get(), &usageMetadata))
		require.Equal(t, []RepositoryMetadata{{
			ID:   123456,
			Name: "hello",
			URL:  "https://github.com/testhq/hello",
		}}, usageMetadata.Repositories)
	})

	t.Run("setup stores max 5 repositories in metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				Repositories: []common.Repository{helloRepo, worldRepo},
			},
		}
		nodeMetadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      nodeMetadataCtx,
			Configuration: map[string]any{"repositories": []string{"hello", "world", "repo3", "repo4", "repo5", "repo6"}},
		})

		require.ErrorContains(t, err, "not accessible") // Will fail validation first
	})

	t.Run("setup fails when repository is not accessible", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				Repositories: []common.Repository{helloRepo},
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repositories": []string{"hello", "notfound"}},
		})

		require.ErrorContains(t, err, "repository notfound is not accessible")
	})
}

func Test__GetWorkflowUsage__Execute(t *testing.T) {
	component := GetWorkflowUsage{}

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
			Configuration:  map[string]any{},
		})

		require.ErrorContains(t, err, "failed to decode metadata")
	})
}

func TestAggregateUsageDataMatchesFullRepositoryName(t *testing.T) {
	repositoryName := "hello"
	report := &gh.UsageReport{UsageItems: []*gh.UsageItem{{
		Product:        "actions",
		Quantity:       12,
		RepositoryName: &repositoryName,
	}}}

	result := aggregateUsageData(report, []string{"testhq/hello"})

	require.Equal(t, float64(12), result.MinutesUsed)
}
