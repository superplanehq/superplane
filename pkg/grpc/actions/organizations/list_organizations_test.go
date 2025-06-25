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
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func Test__ListOrganizations(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	authService := auth.SetupTestAuthService(t)

	t.Run("user can list their own organization", func(t *testing.T) {
		userID := uuid.New()
		ctx := context.Background()
		ctx = authentication.SetUserIdInMetadata(ctx, userID.String())

		organization, err := models.CreateOrganization(userID, "test-org", "Test Organization")
		require.NoError(t, err)
		authService.SetupOrganizationRoles(organization.ID.String())
		authService.AssignRole(userID.String(), authorization.RoleOrgOwner, organization.ID.String(), authorization.DomainOrg)

		res, err := ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, authService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Organizations, 1)
		require.NotNil(t, res.Organizations[0].Metadata)
		assert.Equal(t, organization.ID.String(), res.Organizations[0].Metadata.Id)
		assert.Equal(t, organization.Name, res.Organizations[0].Metadata.Name)
		assert.Equal(t, organization.DisplayName, res.Organizations[0].Metadata.DisplayName)
		assert.Equal(t, organization.CreatedBy.String(), res.Organizations[0].Metadata.CreatedBy)
		assert.NotNil(t, res.Organizations[0].Metadata.CreatedAt)
		assert.NotNil(t, res.Organizations[0].Metadata.UpdatedAt)
	})

	t.Run("user only sees organizations they have access to", func(t *testing.T) {
		user1ID := uuid.New()
		user2ID := uuid.New()

		org1, err := models.CreateOrganization(user1ID, "user1-org", "User 1 Organization")
		require.NoError(t, err)

		org2, err := models.CreateOrganization(user2ID, "user2-org", "User 2 Organization")
		require.NoError(t, err)

		authService.SetupOrganizationRoles(org1.ID.String())
		authService.SetupOrganizationRoles(org2.ID.String())

		authService.AssignRole(user1ID.String(), authorization.RoleOrgOwner, org1.ID.String(), authorization.DomainOrg)

		authService.AssignRole(user2ID.String(), authorization.RoleOrgOwner, org2.ID.String(), authorization.DomainOrg)

		// User1 should only see org1
		ctx1 := context.Background()
		ctx1 = authentication.SetUserIdInMetadata(ctx1, user1ID.String())

		res1, err := ListOrganizations(ctx1, &protos.ListOrganizationsRequest{}, authService)
		require.NoError(t, err)
		require.NotNil(t, res1)
		require.Len(t, res1.Organizations, 1, "User1 should only see their own organization")
		assert.Equal(t, org1.ID.String(), res1.Organizations[0].Metadata.Id)
		assert.Equal(t, org1.Name, res1.Organizations[0].Metadata.Name)

		// User2 should only see org2
		ctx2 := context.Background()
		ctx2 = authentication.SetUserIdInMetadata(ctx2, user2ID.String())

		res2, err := ListOrganizations(ctx2, &protos.ListOrganizationsRequest{}, authService)
		require.NoError(t, err)
		require.NotNil(t, res2)
		require.Len(t, res2.Organizations, 1, "User2 should only see their own organization")
		assert.Equal(t, org2.ID.String(), res2.Organizations[0].Metadata.Id)
		assert.Equal(t, org2.Name, res2.Organizations[0].Metadata.Name)
	})

	t.Run("user with no organization access sees empty list", func(t *testing.T) {
		// user with no orgs
		userID := uuid.New()

		otherUserID := uuid.New()
		organization, err := models.CreateOrganization(otherUserID, "other-org", "Other User Organization")
		require.NoError(t, err)
		authService.SetupOrganizationRoles(organization.ID.String())
		authService.AssignRole(otherUserID.String(), authorization.RoleOrgOwner, organization.ID.String(), authorization.DomainOrg)

		ctx := context.Background()
		ctx = authentication.SetUserIdInMetadata(ctx, userID.String())

		res, err := ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, authService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Organizations, 0, "User with no organization access should see empty list")
	})

	t.Run("user sees all organizations where they have any role", func(t *testing.T) {
		userID := uuid.New()
		otherUserID := uuid.New()

		org1, err := models.CreateOrganization(userID, "owned-org", "User Owned Organization")
		require.NoError(t, err)

		org2, err := models.CreateOrganization(otherUserID, "member-org", "User Member Organization")
		require.NoError(t, err)

		org3, err := models.CreateOrganization(otherUserID, "no-access-org", "No Access Organization")
		require.NoError(t, err)

		authService.SetupOrganizationRoles(org1.ID.String())
		authService.SetupOrganizationRoles(org2.ID.String())
		authService.SetupOrganizationRoles(org3.ID.String())

		authService.AssignRole(userID.String(), authorization.RoleOrgOwner, org1.ID.String(), authorization.DomainOrg)

		authService.AssignRole(userID.String(), authorization.RoleOrgViewer, org2.ID.String(), authorization.DomainOrg)

		authService.AssignRole(otherUserID.String(), authorization.RoleOrgOwner, org3.ID.String(), authorization.DomainOrg)

		// user should see org1 and org2, but not org3
		ctx := context.Background()
		ctx = authentication.SetUserIdInMetadata(ctx, userID.String())

		res, err := ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, authService)
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
