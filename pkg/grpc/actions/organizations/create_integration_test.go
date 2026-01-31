package organizations

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func Test__CreateIntegration(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("duplicate integration name -> error", func(t *testing.T) {
		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"organization": "test-org"})
		require.NoError(t, err)

		//
		// Create first integration
		//
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Integration)
		assert.Equal(t, name, response.Integration.Metadata.Name)

		//
		// Try to create second integration with the same name
		//
		_, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "github", name, appConfig)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, s.Code())
		assert.Contains(t, s.Message(), fmt.Sprintf("an integration with the name %s already exists", name))
	})

	t.Run("reuse integration name after deletion -> success", func(t *testing.T) {
		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"organization": "test-org"})
		require.NoError(t, err)

		//
		// Create first integration
		//
		response, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response)
		integrationID := response.Integration.Metadata.Id

		//
		// Verify integration exists with the correct name
		//
		integration, err := models.FindIntegrationByName(r.Organization.ID, name)
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
		_, err = models.FindIntegrationByName(r.Organization.ID, name)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		//
		// Create a new installation with the same name
		//
		response2, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response2)
		assert.Equal(t, name, response2.Integration.Metadata.Name)
		assert.NotEqual(t, integrationID, response2.Integration.Metadata.Id, "New integration should have different ID")

		//
		// Verify we can find the new installation by name
		//
		newIntegration, err := models.FindIntegrationByName(r.Organization.ID, name)
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
		response1, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response1)
		assert.Equal(t, name, response1.Integration.Metadata.Name)

		//
		// Create integration with same name in second organization
		//
		response2, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, org2.ID.String(), "github", name, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response2)
		assert.Equal(t, name, response2.Integration.Metadata.Name)
		assert.NotEqual(t, response1.Integration.Metadata.Id, response2.Integration.Metadata.Id)

		//
		// Verify both integrations exist and can be found by name in their respective organizations
		//
		integration1, err := models.FindIntegrationByName(r.Organization.ID, name)
		require.NoError(t, err)
		assert.Equal(t, response1.Integration.Metadata.Id, integration1.ID.String())

		integration2, err := models.FindIntegrationByName(org2.ID, name)
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
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "integration nonexistent-app not found")
	})

	t.Run("sync fails -> integration created in error state", func(t *testing.T) {
		//
		// Register a test integration that always fails on Sync
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			return errors.New("oops")
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
		integration, err := models.FindIntegrationByName(r.Organization.ID, name)
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

	t.Run("successful integration -> integration created in ready state", func(t *testing.T) {
		//
		// Register a test integration that succeeds on Sync
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
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
		integration, err := models.FindIntegrationByName(r.Organization.ID, name)
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
}
