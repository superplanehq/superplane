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
)

func Test__ListOrganizations(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("user can list their own organization", func(t *testing.T) {
		res, err := ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Organizations, 1)
		require.NotNil(t, res.Organizations[0].Metadata)
		assert.Equal(t, r.Organization.ID.String(), res.Organizations[0].Metadata.Id)
		assert.Equal(t, r.Organization.Name, res.Organizations[0].Metadata.Name)
		assert.Equal(t, r.Organization.DisplayName, res.Organizations[0].Metadata.DisplayName)
		assert.Equal(t, r.Organization.Description, res.Organizations[0].Metadata.Description)
		assert.Equal(t, r.Organization.CreatedBy.String(), res.Organizations[0].Metadata.CreatedBy)
		assert.NotNil(t, res.Organizations[0].Metadata.CreatedAt)
		assert.NotNil(t, res.Organizations[0].Metadata.UpdatedAt)
	})

	t.Run("user only sees organizations they have access to", func(t *testing.T) {
		user1ID := uuid.New()
		user2ID := uuid.New()

		org1 := support.CreateOrganization(t, r, user1ID)
		org2 := support.CreateOrganization(t, r, user2ID)

		// User1 should only see org1
		ctx := authentication.SetUserIdInMetadata(context.Background(), user1ID.String())
		res, err := ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Organizations, 1)
		assert.Equal(t, org1.ID.String(), res.Organizations[0].Metadata.Id)
		assert.Equal(t, org1.Name, res.Organizations[0].Metadata.Name)

		// User2 should only see org2
		ctx = authentication.SetUserIdInMetadata(context.Background(), user2ID.String())
		res, err = ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Organizations, 1)
		assert.Equal(t, org2.ID.String(), res.Organizations[0].Metadata.Id)
		assert.Equal(t, org2.Name, res.Organizations[0].Metadata.Name)
	})

	t.Run("user with no organization access sees empty list", func(t *testing.T) {
		ctx = authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		res, err := ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Organizations, 0, "User with no organization access should see empty list")
	})

	t.Run("user sees all organizations where they have any role", func(t *testing.T) {
		userID := uuid.New()
		otherUserID := uuid.New()

		org1, err := models.CreateOrganization(userID, "owned-org", "User Owned Organization", "Organization owned by user")
		require.NoError(t, err)

		org2, err := models.CreateOrganization(otherUserID, "member-org", "User Member Organization", "Organization where user is member")
		require.NoError(t, err)

		org3, err := models.CreateOrganization(otherUserID, "no-access-org", "No Access Organization", "Organization with no access")
		require.NoError(t, err)

		require.NoError(t, r.AuthService.SetupOrganizationRoles(org1.ID.String()))
		require.NoError(t, r.AuthService.SetupOrganizationRoles(org2.ID.String()))
		require.NoError(t, r.AuthService.SetupOrganizationRoles(org3.ID.String()))
		require.NoError(t, r.AuthService.AssignRole(userID.String(), models.RoleOrgOwner, org1.ID.String(), models.DomainTypeOrganization))
		require.NoError(t, r.AuthService.AssignRole(userID.String(), models.RoleOrgViewer, org2.ID.String(), models.DomainTypeOrganization))
		require.NoError(t, r.AuthService.AssignRole(otherUserID.String(), models.RoleOrgOwner, org3.ID.String(), models.DomainTypeOrganization))

		// user should see org1 and org2, but not org3
		ctx := context.Background()
		ctx = authentication.SetUserIdInMetadata(ctx, userID.String())

		res, err := ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Organizations, 2, "User should see organizations where they have any role")

		orgIDs := make([]string, len(res.Organizations))
		for i, org := range res.Organizations {
			orgIDs[i] = org.Metadata.Id
		}

		assert.Contains(t, orgIDs, org1.ID.String(), "Should include organization where user is owner")
		assert.Contains(t, orgIDs, org2.ID.String(), "Should include organization where user is viewer")
		assert.NotContains(t, orgIDs, org3.ID.String(), "Should not include organization where user has no role")
	})
}
