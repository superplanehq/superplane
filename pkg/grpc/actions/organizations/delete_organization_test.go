package organizations

import (
	"context"
	"errors"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DeleteOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("organization does not exist -> error", func(t *testing.T) {
		_, err := DeleteOrganization(ctx, r.AuthService, uuid.New().String())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "organization not found", s.Message())
	})

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := DeleteOrganization(context.Background(), r.AuthService, r.Organization.ID.String())
		require.Error(t, err)
		assert.ErrorContains(t, err, "user not authenticated")
	})

	t.Run("organization is deleted", func(t *testing.T) {
		response, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindOrganizationByID(r.Organization.ID.String())
		assert.Error(t, err)
	})
}

func Test__DeleteOrganization_TransactionRollback(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("auth service failure rolls back organization soft-deletion", func(t *testing.T) {
		//
		// Verify organization exists and is not soft-deleted
		//
		foundOrg, err := models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.False(t, foundOrg.DeletedAt.Valid)

		//
		// Use an authentication service that fails
		//
		mockAuth := &mockAuthService{
			Authorization: r.AuthService,
			Error:         errors.New("ooops"),
		}

		//
		// Try to delete organization
		// It should fail due to destroy organization error.
		//
		_, err = DeleteOrganization(ctx, mockAuth, r.Organization.ID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ooops")

		//
		// Verify organization is NOT soft-deleted after transaction rollback
		//
		foundOrg, err = models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.False(t, foundOrg.DeletedAt.Valid)
	})
}
