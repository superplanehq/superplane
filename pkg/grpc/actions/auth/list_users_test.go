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

func TestGetCanvasUsers(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("empty canvas", func(t *testing.T) {
		resp, err := ListUsers(context.Background(), orgID, models.DomainTypeCanvas, canvasID, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Users, 0)
	})

	t.Run("non-empty canvas", func(t *testing.T) {
		userID1 := uuid.New().String()
		userID2 := uuid.New().String()

		err := r.AuthService.AssignRole(userID1, "canvas_admin", canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		err = r.AuthService.AssignRole(userID2, "canvas_viewer", canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		resp, err := ListUsers(context.Background(), orgID, models.DomainTypeCanvas, canvasID, r.AuthService)
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
				assert.Equal(t, canvasID, roleAssignment.DomainId)
			}
		}
	})
}
