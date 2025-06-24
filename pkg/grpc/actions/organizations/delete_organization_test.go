package organizations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DeleteOrganization(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	userID := uuid.New()
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	ctx := context.Background()
	ctx = authentication.SetUserIdInMetadata(ctx, userID.String())

	t.Run("organization does not exist -> error", func(t *testing.T) {
		_, err := DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: uuid.New().String(),
		}, authService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "organization not found", s.Message())
	})

	t.Run("delete organization by ID -> success", func(t *testing.T) {
		organization, err := models.CreateOrganization(userID, "test-org-delete", "Test Organization Delete")
		require.NoError(t, err)
		authService.SetupOrganizationRoles(organization.ID.String())

		response, err := DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: organization.ID.String(),
		}, authService)

		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindOrganizationByID(organization.ID.String())
		assert.Error(t, err)
	})

	t.Run("delete organization by name -> success", func(t *testing.T) {
		organization, err := models.CreateOrganization(userID, "test-org-delete-2", "Test Organization Delete 2")
		require.NoError(t, err)
		authService.SetupOrganizationRoles(organization.ID.String())

		response, err := DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: organization.Name,
		}, authService)

		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindOrganizationByName(organization.Name)
		assert.Error(t, err)
	})

	t.Run("empty id_or_name -> error", func(t *testing.T) {
		_, err := DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: "",
		}, authService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "id_or_name is required", s.Message())
	})

	t.Run("invalid requester ID -> error", func(t *testing.T) {
		organization, err := models.CreateOrganization(userID, "test-org-delete-3", "Test Organization Delete 3")
		require.NoError(t, err)

		_, err = DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: organization.ID.String(),
		}, authService)

		assert.Error(t, err)
	})
}
