package me

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func Test_ListUserPermissions(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()

	//
	// Assign viewer role to user, and prepare context with user ID and organization ID
	//
	require.NoError(t, r.AuthService.AssignRole(r.User.String(), models.RoleOrgViewer, orgID, models.DomainTypeOrganization))
	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(
			"x-organization-id", orgID,
			"x-user-id", r.User.String(),
		),
	)

	t.Run("no user in context", func(t *testing.T) {
		_, err := GetUser(context.Background(), r.AuthService, false)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("permissions not included in response", func(t *testing.T) {
		resp, err := GetUser(ctx, r.AuthService, false)
		require.NoError(t, err)
		assert.NotNil(t, resp.User)
		assert.Empty(t, resp.User.Permissions)
	})

	t.Run("includes permissions", func(t *testing.T) {
		resp, err := GetUser(ctx, r.AuthService, true)
		require.NoError(t, err)
		assert.NotNil(t, resp.User)
		assert.NotEmpty(t, resp.User.Permissions)
		assert.ElementsMatch(t, resp.User.Permissions, getExpectedPermissions([]string{
			"org",
			"roles",
			"groups",
			"members",
			"canvases",
			"blueprints",
			"service_accounts",
			"agents",
		}))
	})
}

func getExpectedPermissions(resources []string) []*pbAuth.Permission {
	permissions := make([]*pbAuth.Permission, 0, len(resources))
	for _, resource := range resources {
		permissions = append(permissions, &pbAuth.Permission{
			Resource:   resource,
			Action:     "read",
			DomainType: actions.DomainTypeToProto(models.DomainTypeOrganization),
		})
	}
	return permissions
}
