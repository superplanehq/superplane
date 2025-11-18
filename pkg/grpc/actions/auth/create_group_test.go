package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"github.com/superplanehq/superplane/test/support"
)

func Test_CreateGroup(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("successful group creation", func(t *testing.T) {
		req := &pb.CreateGroupRequest{
			Group: &pb.Group{
				Metadata: &pb.Group_Metadata{
					Name: "test-group",
				},
				Spec: &pb.Group_Spec{
					Role:        models.RoleOrgAdmin,
					DisplayName: "test-group",
					Description: "test-group",
				},
			},
		}

		resp, err := CreateGroup(ctx, "org", orgID, req.Group, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if group was created
		groups, err := r.AuthService.GetGroups(orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.Contains(t, groups, "test-group")
		assert.Len(t, groups, 1)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.CreateGroupRequest{
			Group: &pb.Group{
				Metadata: &pb.Group_Metadata{
					Name: "",
				},
				Spec: &pb.Group_Spec{
					Role:        models.RoleOrgAdmin,
					DisplayName: "test-group",
					Description: "test-group",
				},
			},
		}

		_, err := CreateGroup(ctx, "org", orgID, req.Group, r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})
}
