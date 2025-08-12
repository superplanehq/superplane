package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ListGroups(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	require.NoError(t, r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group-1", models.RoleOrgAdmin, "Test Group 1", "A test group"))
	require.NoError(t, r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group-2", models.RoleOrgViewer, "Test Group 2", "Another test group"))

	t.Run("successful list groups", func(t *testing.T) {
		resp, err := ListGroups(ctx, models.DomainTypeOrganization, orgID, r.AuthService)
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

	t.Run("successful canvas groups list", func(t *testing.T) {
		require.NoError(t, r.AuthService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, "canvas-group-1", models.RoleCanvasAdmin, "Canvas Group 1", "A canvas group"))
		require.NoError(t, r.AuthService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, "canvas-group-2", models.RoleCanvasViewer, "Canvas Group 2", "Another canvas group"))

		resp, err := ListGroups(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Groups, 2)

		for _, group := range resp.Groups {
			assert.NotEmpty(t, group.Metadata.Name)
			assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, group.Metadata.DomainType)
			assert.Equal(t, r.Canvas.ID.String(), group.Metadata.DomainId)
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
		require.NoError(t, r.AuthService.AddUserToGroup(orgID, "org", "test-user-1", "test-group-1"))

		resp, err := ListGroups(ctx, models.DomainTypeOrganization, orgID, r.AuthService)
		require.NoError(t, err)

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
