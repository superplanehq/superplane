package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
)

func Test_ListGroups(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create some groups first
	err = authService.CreateGroup(orgID, "org", "test-group-1", models.RoleOrgAdmin)
	require.NoError(t, err)
	err = authService.CreateGroup(orgID, "org", "test-group-2", models.RoleOrgViewer)
	require.NoError(t, err)

	err = models.UpsertGroupMetadata("test-group-1", "org", orgID, "Test Group 1", "A test group")
	require.NoError(t, err)
	err = models.UpsertGroupMetadata("test-group-2", "org", orgID, "Test Group 2", "Another test group")
	require.NoError(t, err)

	t.Run("successful list groups", func(t *testing.T) {
		req := &pb.ListGroupsRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		resp, err := ListGroups(ctx, models.DomainTypeOrg, orgID, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Groups, 2)

		// Check that groups have the correct structure
		for _, group := range resp.Groups {
			assert.NotEmpty(t, group.Metadata.Name)
			assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, group.Metadata.DomainType)
			assert.Equal(t, orgID, group.Metadata.DomainId)
			assert.Contains(t, []string{"org_admin", "org_viewer"}, group.Spec.Role)
			assert.GreaterOrEqual(t, group.Status.MembersCount, int32(0))
			assert.NotEmpty(t, group.Metadata.CreatedAt)
			assert.NotEmpty(t, group.Metadata.UpdatedAt)
		}

		// Check specific group names
		groupNames := make([]string, len(resp.Groups))
		for i, group := range resp.Groups {
			groupNames[i] = group.Metadata.Name
		}
		assert.Contains(t, groupNames, "test-group-1")
		assert.Contains(t, groupNames, "test-group-2")
	})

	t.Run("invalid request - missing domain type", func(t *testing.T) {
		req := &pb.ListGroupsRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
		}

		_, err := ListGroups(ctx, models.DomainTypeOrg, orgID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("successful canvas groups list", func(t *testing.T) {
		canvasID := uuid.New().String()

		// Setup canvas roles and create canvas groups
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group-1", models.RoleCanvasAdmin)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group-2", models.RoleCanvasViewer)
		require.NoError(t, err)

		err = models.UpsertGroupMetadata("canvas-group-1", "canvas", canvasID, "Canvas Group 1", "A canvas group")
		require.NoError(t, err)
		err = models.UpsertGroupMetadata("canvas-group-2", "canvas", canvasID, "Canvas Group 2", "Another canvas group")
		require.NoError(t, err)

		req := &pb.ListGroupsRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
		}

		resp, err := ListGroups(ctx, models.DomainTypeCanvas, canvasID, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Groups, 2)

		// Check that groups have the correct structure
		for _, group := range resp.Groups {
			assert.NotEmpty(t, group.Metadata.Name)
			assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, group.Metadata.DomainType)
			assert.Equal(t, canvasID, group.Metadata.DomainId)
			assert.GreaterOrEqual(t, group.Status.MembersCount, int32(0))
			assert.NotEmpty(t, group.Metadata.CreatedAt)
			assert.NotEmpty(t, group.Metadata.UpdatedAt)
		}

		// Check specific group names
		groupNames := make([]string, len(resp.Groups))
		for i, group := range resp.Groups {
			groupNames[i] = group.Metadata.Name
		}
		assert.Contains(t, groupNames, "canvas-group-1")
		assert.Contains(t, groupNames, "canvas-group-2")
	})

	t.Run("groups with metadata have timestamps", func(t *testing.T) {
		// Create group metadata for one of the groups
		err := models.UpsertGroupMetadata("test-group-1", "org", orgID, "Test Group 1", "A test group")
		require.NoError(t, err)

		// Add a user to the group to test members count
		err = authService.AddUserToGroup(orgID, "org", "test-user-1", "test-group-1")
		require.NoError(t, err)

		req := &pb.ListGroupsRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		resp, err := ListGroups(ctx, models.DomainTypeOrg, orgID, req, authService)
		require.NoError(t, err)

		// Find the group with metadata
		var groupWithMetadata *pb.Group
		for _, group := range resp.Groups {
			if group.Metadata.Name == "test-group-1" {
				groupWithMetadata = group
				break
			}
		}

		require.NotNil(t, groupWithMetadata)
		assert.NotEmpty(t, groupWithMetadata.Metadata.CreatedAt)
		assert.NotEmpty(t, groupWithMetadata.Metadata.UpdatedAt)
		assert.Equal(t, int32(1), groupWithMetadata.Status.MembersCount)
		assert.Equal(t, "Test Group 1", groupWithMetadata.Spec.DisplayName)
		assert.Equal(t, "A test group", groupWithMetadata.Spec.Description)
	})
}
