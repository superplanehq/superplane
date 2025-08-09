package organizations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DeleteOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := DeleteOrganization(context.Background(), &protos.DeleteOrganizationRequest{
			IdOrName: r.Organization.ID.String(),
		}, r.AuthService)

		assert.ErrorContains(t, err, "user not authenticated")
	})

	t.Run("organization does not exist -> error", func(t *testing.T) {
		_, err := DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: uuid.New().String(),
		}, r.AuthService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "organization not found", s.Message())
	})

	t.Run("delete organization by ID -> success", func(t *testing.T) {
		response, err := DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: r.Organization.ID.String(),
		}, r.AuthService)

		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindOrganizationByID(r.Organization.ID.String())
		assert.Error(t, err)
	})

	t.Run("delete organization by name -> success", func(t *testing.T) {
		organization, err := models.CreateOrganization("test-org-delete-2", "Test Organization Delete 2", "Organization to be deleted by name")
		require.NoError(t, err)
		r.AuthService.SetupOrganizationRoles(organization.ID.String())

		response, err := DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: organization.Name,
		}, r.AuthService)

		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindOrganizationByName(organization.Name)
		assert.Error(t, err)
	})

	t.Run("empty id_or_name -> error", func(t *testing.T) {
		_, err := DeleteOrganization(ctx, &protos.DeleteOrganizationRequest{
			IdOrName: "",
		}, r.AuthService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "id_or_name is required", s.Message())
	})
}
