package organizations

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func Test__CreateIntegration(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		OnSync: func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		},
	})

	t.Run("duplicate integration name -> error", func(t *testing.T) {
		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"organization": "test-org"})
		require.NoError(t, err)

		//
		// Create first integration
		//
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Integration)
		assert.Equal(t, name, response.Integration.Metadata.Name)

		//
		// Try to create second integration with the same name
		//
		_, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, code)
		assert.Contains(t, msg, fmt.Sprintf("an integration with the name %s already exists", name))
	})

	t.Run("reuse integration name after deletion -> success", func(t *testing.T) {
		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"organization": "test-org"})
		require.NoError(t, err)

		//
		// Create first integration
		//
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response)
		integrationID := response.Integration.Metadata.Id

		//
		// Verify integration exists with the correct name
		//
		integration, err := models.FindIntegrationByName(database.Conn(), r.Organization.ID, name)
		require.NoError(t, err)
		assert.Equal(t, integrationID, integration.ID.String())
		assert.Equal(t, name, integration.InstallationName)

		//
		// Delete the integration
		//
		_, err = DeleteIntegration(ctx, r.Organization.ID.String(), integrationID)
		require.NoError(t, err)

		//
		// Verify the installation is soft-deleted and the name has been modified
		//
		deletedIntegration, err := models.FindMaybeDeletedIntegrationInTransaction(database.Conn(), integration.ID)
		require.NoError(t, err)
		assert.True(t, deletedIntegration.DeletedAt.Valid)
		assert.NotEqual(t, name, deletedIntegration.InstallationName)
		assert.Contains(t, deletedIntegration.InstallationName, "deleted-")

		//
		// Verify we cannot find it by the original name anymore
		//
		_, err = models.FindIntegrationByName(database.Conn(), r.Organization.ID, name)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		//
		// Create a new installation with the same name
		//
		response2, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response2)
		assert.Equal(t, name, response2.Integration.Metadata.Name)
		assert.NotEqual(t, integrationID, response2.Integration.Metadata.Id, "New integration should have different ID")

		//
		// Verify we can find the new installation by name
		//
		newIntegration, err := models.FindIntegrationByName(database.Conn(), r.Organization.ID, name)
		require.NoError(t, err)
		assert.Equal(t, name, newIntegration.InstallationName)
		assert.Equal(t, response2.Integration.Metadata.Id, newIntegration.ID.String())
	})

	t.Run("different organizations can have integrations with the same name -> success", func(t *testing.T) {
		//
		// Create a second organization
		//
		org2, err := models.CreateOrganization("org-2", "")
		require.NoError(t, err)

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"organization": "test-org"})
		require.NoError(t, err)

		//
		// Create integration in first organization
		//
		response1, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response1)
		assert.Equal(t, name, response1.Integration.Metadata.Name)

		//
		// Create integration with same name in second organization
		//
		response2, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org2.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response2)
		assert.Equal(t, name, response2.Integration.Metadata.Name)
		assert.NotEqual(t, response1.Integration.Metadata.Id, response2.Integration.Metadata.Id)

		//
		// Verify both integrations exist and can be found by name in their respective organizations
		//
		integration1, err := models.FindIntegrationByName(database.Conn(), r.Organization.ID, name)
		require.NoError(t, err)
		assert.Equal(t, response1.Integration.Metadata.Id, integration1.ID.String())

		integration2, err := models.FindIntegrationByName(database.Conn(), org2.ID, name)
		require.NoError(t, err)
		assert.Equal(t, response2.Integration.Metadata.Id, integration2.ID.String())
	})

	t.Run("integration does not exist -> error", func(t *testing.T) {
		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{})
		require.NoError(t, err)

		//
		// Try to create an integration that doesn't exist
		//
		_, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "nonexistent-app", name, appConfig)
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Contains(t, msg, "integration nonexistent-app not found")
	})

	t.Run("sync fails -> integration created in error state", func(t *testing.T) {
		//
		// Register a test integration that always fails on Sync
		//
		r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				return errors.New("oops")
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]interface{}{})
		require.NoError(t, err)

		//
		// Create the integration
		//
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Integration)

		//
		// Verify integration was created
		//
		integration, err := models.FindIntegrationByName(database.Conn(), r.Organization.ID, name)
		require.NoError(t, err)
		assert.Equal(t, name, integration.InstallationName)

		//
		// Verify integration is in error state with the error message
		//
		assert.Equal(t, models.IntegrationStateError, integration.State)
		assert.Equal(t, "oops", integration.StateDescription)

		//
		// Verify the response also reflects the error state
		//
		assert.Equal(t, models.IntegrationStateError, response.Integration.Status.State)
		assert.Equal(t, "oops", response.Integration.Status.StateDescription)
	})

	for _, appName := range []string{"claude", "openai"} {
		t.Run(appName+" rejects an invalid key before saving", func(t *testing.T) {
			original := r.Registry.Integrations[appName]
			t.Cleanup(func() { r.Registry.Integrations[appName] = original })
			invalidKey := true
			r.Registry.Integrations[appName] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
				OnSync: func(ctx core.SyncContext) error {
					if invalidKey {
						return errors.New("invalid API key")
					}
					ctx.Integration.Ready()
					return nil
				},
			})
			name := support.RandomName("key")
			response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), appName, name, nil)
			require.Error(t, err)
			assert.Nil(t, response)
			code, message, ok := grpcerrors.HandlerStatus(err)
			require.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, code)
			assert.Equal(t, "Unable to verify the API key. Check the key and try again.", message)
			var count int64
			require.NoError(t, database.DB(ctx).Unscoped().Model(&models.Integration{}).
				Where("organization_id = ? AND installation_name = ?", r.Organization.ID, name).Count(&count).Error)
			assert.Zero(t, count)

			invalidKey = false
			response, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), appName, name, nil)
			require.NoError(t, err)
			assert.Equal(t, models.IntegrationStateReady, response.Integration.Status.State)
			stored, err := models.FindIntegrationByName(database.DB(ctx), r.Organization.ID, name)
			require.NoError(t, err)
			assert.Equal(t, response.Integration.Metadata.Id, stored.ID.String())
		})
	}

	t.Run("successful integration -> integration created in ready state", func(t *testing.T) {
		//
		// Register a test integration that succeeds on Sync
		//
		r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{})
		require.NoError(t, err)

		//
		// Create the integration
		//
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Integration)

		//
		// Verify integration was created
		//
		integration, err := models.FindIntegrationByName(database.Conn(), r.Organization.ID, name)
		require.NoError(t, err)
		assert.Equal(t, name, integration.InstallationName)
		assert.Equal(t, "dummy", integration.AppName)

		//
		// Verify integration is not in error state
		//
		assert.Equal(t, models.IntegrationStateReady, integration.State)
		assert.Empty(t, integration.StateDescription)

		//
		// Verify the response reflects successful integration
		//
		assert.Equal(t, models.IntegrationStateReady, response.Integration.Status.State)
		assert.Empty(t, response.Integration.Status.StateDescription)
	})

	t.Run("github uses legacy create when new setup flow feature is off", func(t *testing.T) {
		t.Setenv("SUPERPLANE_GITHUB_APP_ID", "")
		t.Setenv("SUPERPLANE_GITHUB_APP_SLUG", "")
		t.Setenv("SUPERPLANE_GITHUB_APP_PRIVATE_KEY", "")
		t.Setenv("SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET", "")

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"organization": "test-org"})
		require.NoError(t, err)

		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response.Integration)
		assert.Nil(t, response.Integration.Status.SetupState)
	})

	t.Run("github uses setup provider when new setup flow feature is on", func(t *testing.T) {
		t.Setenv("SUPERPLANE_GITHUB_APP_ID", "")
		t.Setenv("SUPERPLANE_GITHUB_APP_SLUG", "")
		t.Setenv("SUPERPLANE_GITHUB_APP_PRIVATE_KEY", "")
		t.Setenv("SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET", "")

		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureNewIntegrationSetupFlow))

		name := support.RandomName("integration")
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org.ID.String(), "github", name, nil)
		require.NoError(t, err)
		require.NotNil(t, response.Integration)
		require.NotNil(t, response.Integration.Status.SetupState)
		require.NotNil(t, response.Integration.Status.SetupState.CurrentStep)
	})

	t.Run("github requires repository selection when the public app is configured", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureNewIntegrationSetupFlow))
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureFactories))

		t.Setenv("SUPERPLANE_GITHUB_APP_ID", "99")
		t.Setenv("SUPERPLANE_GITHUB_APP_SLUG", "superplane")
		t.Setenv("SUPERPLANE_GITHUB_APP_PRIVATE_KEY", "test-pem")
		t.Setenv("SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET", "whsec")

		name := support.RandomName("integration")
		_, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org.ID.String(), "github", name, nil)
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "select a repository")
		_, findErr := models.FindIntegrationByName(database.Conn(), org.ID, name)
		assert.ErrorIs(t, findErr, gorm.ErrRecordNotFound)
	})

	t.Run("sentry uses hosted install when the public app env is set", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureNewIntegrationSetupFlow))

		t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
		t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
		t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")

		name := support.RandomName("integration")
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org.ID.String(), "sentry", name, nil)
		require.NoError(t, err)
		require.NotNil(t, response.Integration)
		assert.Nil(t, response.Integration.Status.SetupState)
		require.NotNil(t, response.Integration.Status.BrowserAction)
		assert.Equal(t, "GET", response.Integration.Status.BrowserAction.Method)
		assert.Contains(t, response.Integration.Status.BrowserAction.Url, "/api/v1/sentry/app/install?state=")
	})

	t.Run("github keeps legacy create when privateApp is set and setup flow feature is off", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureFactories))

		t.Setenv("SUPERPLANE_GITHUB_APP_ID", "99")
		t.Setenv("SUPERPLANE_GITHUB_APP_SLUG", "superplane")
		t.Setenv("SUPERPLANE_GITHUB_APP_PRIVATE_KEY", "test-pem")
		t.Setenv("SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET", "whsec")

		appConfig, err := structpb.NewStruct(map[string]any{"privateApp": true})
		require.NoError(t, err)

		name := support.RandomName("integration")
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response.Integration)
		assert.Nil(t, response.Integration.Status.SetupState)
		require.NotNil(t, response.Integration.Status.BrowserAction)
		assert.Equal(t, "POST", response.Integration.Status.BrowserAction.Method)
	})

	t.Run("github uses setup provider when privateApp is set even if hosted install is on", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureNewIntegrationSetupFlow))
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureFactories))

		t.Setenv("SUPERPLANE_GITHUB_APP_ID", "99")
		t.Setenv("SUPERPLANE_GITHUB_APP_SLUG", "superplane")
		t.Setenv("SUPERPLANE_GITHUB_APP_PRIVATE_KEY", "test-pem")
		t.Setenv("SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET", "whsec")

		appConfig, err := structpb.NewStruct(map[string]any{"privateApp": true})
		require.NoError(t, err)

		name := support.RandomName("integration")
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response.Integration)
		require.NotNil(t, response.Integration.Status.SetupState)
		require.NotNil(t, response.Integration.Status.SetupState.CurrentStep)
		assert.Nil(t, response.Integration.Status.BrowserAction)

		stored, err := models.FindIntegrationByName(database.Conn(), org.ID, name)
		require.NoError(t, err)
		assert.Equal(t, "", stored.Configuration.Data()["host"])
		assert.Equal(t, true, stored.Configuration.Data()["privateApp"])
	})

	t.Run("github wizard rejects a GitHub Enterprise host", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureNewIntegrationSetupFlow))
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureFactories))

		appConfig, err := structpb.NewStruct(map[string]any{"privateApp": true, "host": "acme.ghe.com"})
		require.NoError(t, err)

		name := support.RandomName("integration")
		_, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org.ID.String(), "github", name, appConfig)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub Enterprise is not available")
	})

	t.Run("github wizard accepts GitHub Enterprise Cloud when the flag is on", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureGitHubEnterprise))

		t.Setenv("SUPERPLANE_GITHUB_APP_ID", "99")
		t.Setenv("SUPERPLANE_GITHUB_APP_SLUG", "superplane")
		t.Setenv("SUPERPLANE_GITHUB_APP_PRIVATE_KEY", "test-pem")
		t.Setenv("SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET", "whsec")
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureFactories))

		restore := github.WithProbeGitHubHostForTest(func(core.HTTPContext, githubcommon.Host, uuid.UUID) error {
			return nil
		})
		t.Cleanup(restore)

		appConfig, err := structpb.NewStruct(map[string]any{"host": "acme.ghe.com"})
		require.NoError(t, err)

		name := support.RandomName("integration")
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response.Integration)
		require.NotNil(t, response.Integration.Status.SetupState)
		assert.Nil(t, response.Integration.Status.BrowserAction)

		stored, err := models.FindIntegrationByName(database.Conn(), org.ID, name)
		require.NoError(t, err)
		assert.Equal(t, "acme.ghe.com", stored.Configuration.Data()["host"])
		assert.Equal(t, true, stored.Configuration.Data()["privateApp"])
		_, bindErr := models.FindVCSProviderIntegrationBinding(database.Conn(), stored.ID)
		assert.ErrorIs(t, bindErr, gorm.ErrRecordNotFound)
	})

	t.Run("smtp host does not force the GitHub wizard", func(t *testing.T) {
		assert.False(t, usesSetupWizard(r.Registry, r.Organization.ID, "smtp", map[string]any{"host": "smtp.gmail.com"}))
		assert.True(t, usesSetupWizard(r.Registry, r.Organization.ID, "github", map[string]any{"host": "acme.ghe.com"}))
	})
}
