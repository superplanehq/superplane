package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func TestGetOrganizationUsers(t *testing.T) {
	authService := SetupTestAuthService(t)

	// Create test organization ID
	orgID := uuid.New().String()

	// Setup organization roles
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create test user IDs
	userID1 := uuid.New().String()
	userID2 := uuid.New().String()

	// Assign roles to users
	err = authService.AssignRole(userID1, "org_admin", orgID, authorization.DomainOrg)
	require.NoError(t, err)

	err = authService.AssignRole(userID2, "org_viewer", orgID, authorization.DomainOrg)
	require.NoError(t, err)

	// Test getting organization users
	req := &pb.GetOrganizationUsersRequest{
		OrganizationId: orgID,
	}

	resp, err := GetOrganizationUsers(context.Background(), req, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have 2 users
	assert.Len(t, resp.Users, 2)

	// Check that users have role assignments
	for _, user := range resp.Users {
		assert.NotEmpty(t, user.UserId)
		assert.NotEmpty(t, user.RoleAssignments)

		// Check that is_active field is properly set
		// For test fallback users, should be false
		assert.False(t, user.IsActive)

		// Check role assignment details
		for _, roleAssignment := range user.RoleAssignments {
			assert.NotEmpty(t, roleAssignment.RoleName)
			assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, roleAssignment.DomainType)
			assert.Equal(t, orgID, roleAssignment.DomainId)
		}
	}
}

func TestGetOrganizationUsersEmptyOrganization(t *testing.T) {
	authService := SetupTestAuthService(t)

	// Create test organization ID
	orgID := uuid.New().String()

	// Setup organization roles (but don't assign any users)
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Test getting organization users
	req := &pb.GetOrganizationUsersRequest{
		OrganizationId: orgID,
	}

	resp, err := GetOrganizationUsers(context.Background(), req, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have 0 users
	assert.Len(t, resp.Users, 0)
}

func TestGetOrganizationUsersInvalidOrganizationId(t *testing.T) {
	authService := SetupTestAuthService(t)

	req := &pb.GetOrganizationUsersRequest{
		OrganizationId: "invalid-uuid",
	}

	resp, err := GetOrganizationUsers(context.Background(), req, authService)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid organization ID")
}

func TestGetOrganizationUsersWithActiveUser(t *testing.T) {
	authService := SetupTestAuthService(t)

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	user := &models.User{
		Name:     "Active User",
		IsActive: true,
	}
	err = user.Create()
	require.NoError(t, err)

	err = authService.AssignRole(user.ID.String(), "org_admin", orgID, authorization.DomainOrg)
	require.NoError(t, err)

	req := &pb.GetOrganizationUsersRequest{
		OrganizationId: orgID,
	}

	resp, err := GetOrganizationUsers(context.Background(), req, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Users, 1)

	activeUser := resp.Users[0]
	assert.Equal(t, user.ID.String(), activeUser.UserId)
	assert.True(t, activeUser.IsActive)
	assert.Equal(t, "Active User", activeUser.DisplayName)
	assert.NotEmpty(t, activeUser.RoleAssignments)

	assert.Equal(t, "org_admin", activeUser.RoleAssignments[0].RoleName)
	assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, activeUser.RoleAssignments[0].DomainType)
	assert.Equal(t, orgID, activeUser.RoleAssignments[0].DomainId)
}
