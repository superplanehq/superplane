package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ListUsers(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)

	err := authService.SetupCanvasRoles(r.Canvas.ID.String())
	require.NoError(t, err)

	t.Run("empty canvas", func(t *testing.T) {
		resp, err := ListUsers(context.Background(), models.DomainTypeCanvas, r.Canvas.ID.String(), authService)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Users, 0)
	})

	t.Run("canvas with users", func(t *testing.T) {
		err = authService.AssignRole(uuid.New().String(), "canvas_admin", r.Canvas.ID.String(), models.DomainTypeCanvas)
		require.NoError(t, err)

		err = authService.AssignRole(uuid.New().String(), "canvas_viewer", r.Canvas.ID.String(), models.DomainTypeCanvas)
		require.NoError(t, err)

		resp, err := ListUsers(context.Background(), models.DomainTypeCanvas, r.Canvas.ID.String(), authService)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Users, 2)

		for _, user := range resp.Users {
			assert.NotEmpty(t, user.Metadata.Id)
			assert.NotEmpty(t, user.Status.RoleAssignments)
			assert.False(t, user.Status.IsActive)

			for _, roleAssignment := range user.Status.RoleAssignments {
				assert.NotEmpty(t, roleAssignment.RoleName)
				assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, roleAssignment.DomainType)
				assert.Equal(t, r.Canvas.ID.String(), roleAssignment.DomainId)
			}
		}
	})
}
