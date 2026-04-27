package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func Test_RemoveUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("user not found -> error", func(t *testing.T) {
		_, err := RemoveUser(ctx, r.AuthService, orgID, uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("user found -> removes user from organization", func(t *testing.T) {
		//
		// Add new user to organization, and create new canvases for it.
		//
		newUser := support.CreateUser(t, r, r.Organization.ID)
		plainToken, err := crypto.Base64String(64)
		require.NoError(t, err)
		require.NoError(t, newUser.UpdateTokenHash(crypto.HashToken(plainToken)))

		//
		// Remove the user from the organization
		//
		_, err = RemoveUser(ctx, r.AuthService, orgID, newUser.ID.String())
		require.NoError(t, err)

		//
		// Verify the user is soft deleted, and no longer active.
		//
		user, err := models.FindMaybeDeletedUserByID(orgID, newUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, user.DeletedAt)
		_, err = models.FindActiveUserByID(orgID, newUser.ID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindActiveUserByEmail(orgID, newUser.GetEmail())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindActiveUserByTokenHash(newUser.TokenHash)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		require.Empty(t, user.TokenHash)

		//
		// Verify no organization roles exist anymore for that user anymore
		//
		roles, err := r.AuthService.GetUserRolesForOrg(newUser.ID.String(), orgID)
		require.NoError(t, err)
		require.Len(t, roles, 0)
	})

	t.Run("role removal failure -> user remains active with original roles", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)

		rolesBefore, err := r.AuthService.GetUserRolesForOrg(newUser.ID.String(), orgID)
		require.NoError(t, err)
		require.NotEmpty(t, rolesBefore)

		failingRole := rolesBefore[0].Name
		mockAuth := &removeUserAuthService{
			Authorization:   r.AuthService,
			failOnRoleName:  failingRole,
			removeRoleError: errors.New("forced remove role error"),
		}

		_, err = RemoveUser(ctx, mockAuth, orgID, newUser.ID.String())
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, s.Code())
		assert.Equal(t, "error removing role", s.Message())

		_, err = models.FindActiveUserByID(orgID, newUser.ID.String())
		require.NoError(t, err)

		rolesAfter, err := r.AuthService.GetUserRolesForOrg(newUser.ID.String(), orgID)
		require.NoError(t, err)
		require.NotEmpty(t, rolesAfter)

		roleNamesAfter := make([]string, 0, len(rolesAfter))
		for _, role := range rolesAfter {
			roleNamesAfter = append(roleNamesAfter, role.Name)
		}
		assert.Contains(t, roleNamesAfter, failingRole)
	})
}

type removeUserAuthService struct {
	authorization.Authorization
	failOnRoleName  string
	removeRoleError error
}

func (m *removeUserAuthService) RemoveRole(userID, role, domainID string, domainType string) error {
	if m.removeRoleError != nil && role == m.failOnRoleName {
		return m.removeRoleError
	}

	return m.Authorization.RemoveRole(userID, role, domainID, domainType)
}
