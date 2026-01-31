package organizations

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__DeleteIntegration(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("delete existing integration -> success", func(t *testing.T) {
		//
		// Register a test integration
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create installation
		//
		installResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := installResponse.Integration.Metadata.Id

		//
		// Verify integration is visible in the list
		//
		listResponse, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		found := slices.ContainsFunc(listResponse.Integrations, func(integration *pb.Integration) bool {
			return integration.Metadata.Id == integrationID
		})
		assert.True(t, found, "Integration should be visible before deletion")

		//
		// Delete the integration
		//
		deleteResponse, err := DeleteIntegration(ctx, r.Organization.ID.String(), integrationID)
		require.NoError(t, err)
		require.NotNil(t, deleteResponse)

		//
		// Verify integration is no longer visible in the list
		//
		listResponse, err = ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		found = slices.ContainsFunc(listResponse.Integrations, func(integration *pb.Integration) bool {
			return integration.Metadata.Id == integrationID
		})
		assert.False(t, found, "Deleted integration should not be visible in list")

		//
		// Verify integration is soft-deleted in the database
		//
		integration, err := models.FindMaybeDeletedIntegrationInTransaction(database.Conn(), uuid.MustParse(integrationID))
		require.NoError(t, err)
		assert.True(t, integration.DeletedAt.Valid)
	})

	t.Run("invalid organization ID -> error", func(t *testing.T) {
		//
		// Try to delete with an invalid organization ID
		//
		_, err := DeleteIntegration(ctx, "invalid-uuid", uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid organization ID")
	})

	t.Run("invalid integration ID -> error", func(t *testing.T) {
		//
		// Try to delete with an invalid integration ID
		//
		_, err := DeleteIntegration(ctx, r.Organization.ID.String(), "invalid-uuid")
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid integration ID")
	})

	t.Run("non-existent integration -> error", func(t *testing.T) {
		//
		// Try to delete a non-existent integration
		//
		_, err := DeleteIntegration(ctx, r.Organization.ID.String(), uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "integration not found")
	})

	t.Run("integration from different organization -> error", func(t *testing.T) {
		//
		// Register a test application
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		})

		//
		// Create a second organization
		//
		org2, err := models.CreateOrganization("org-2", "")
		require.NoError(t, err)

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create integration in first organization
		//
		installResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := installResponse.Integration.Metadata.Id

		//
		// Try to delete using the second organization's ID
		//
		_, err = DeleteIntegration(ctx, org2.ID.String(), integrationID)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("delete twice -> error on second attempt", func(t *testing.T) {
		//
		// Register a test integration
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create integration
		//
		installResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := installResponse.Integration.Metadata.Id

		//
		// Delete the integration (first time)
		//
		_, err = DeleteIntegration(ctx, r.Organization.ID.String(), integrationID)
		require.NoError(t, err)

		//
		// Try to delete again (should fail)
		//
		_, err = DeleteIntegration(ctx, r.Organization.ID.String(), integrationID)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "integration not found")
	})

	t.Run("delete modifies integration name to prevent name conflicts -> success", func(t *testing.T) {
		//
		// Register a test integration
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create integration
		//
		installResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := installResponse.Integration.Metadata.Id

		//
		// Delete the integration
		//
		_, err = DeleteIntegration(ctx, r.Organization.ID.String(), integrationID)
		require.NoError(t, err)

		//
		// Verify the integration name has been modified
		//
		integration, err := models.FindMaybeDeletedIntegrationInTransaction(database.Conn(), uuid.MustParse(integrationID))
		require.NoError(t, err)
		assert.True(t, integration.DeletedAt.Valid)
		assert.NotEqual(t, name, integration.InstallationName)
		assert.Contains(t, integration.InstallationName, "deleted-", "Integration name should be prefixed with 'deleted-'")
	})
}
