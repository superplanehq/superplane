package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
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

func TestGetOrganizationUsersMultipleRoles(t *testing.T) {
	authService := SetupTestAuthService(t)
	
	// Create test organization ID
	orgID := uuid.New().String()
	
	// Setup organization roles
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create test user ID
	userID := uuid.New().String()

	// Assign multiple roles to the same user
	err = authService.AssignRole(userID, "org_admin", orgID, authorization.DomainOrg)
	require.NoError(t, err)
	
	err = authService.AssignRole(userID, "org_viewer", orgID, authorization.DomainOrg)
	require.NoError(t, err)

	// Test getting organization users
	req := &pb.GetOrganizationUsersRequest{
		OrganizationId: orgID,
	}

	resp, err := GetOrganizationUsers(context.Background(), req, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)
	
	// Should have 1 user with multiple roles
	assert.Len(t, resp.Users, 1)
	assert.Len(t, resp.Users[0].RoleAssignments, 2)
	
	// Check that both roles are present
	roleNames := make(map[string]bool)
	for _, roleAssignment := range resp.Users[0].RoleAssignments {
		roleNames[roleAssignment.RoleName] = true
	}
	
	assert.True(t, roleNames["org_admin"])
	assert.True(t, roleNames["org_viewer"])
}