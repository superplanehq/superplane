package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListCanvases(t *testing.T) {
	r := support.Setup(t)
	authService := auth.SetupTestAuthService(t)

	user := models.User{
		ID: r.User,
	}

	t.Run("no organization ID -> list all canvases", func(t *testing.T) {
		ctx := context.Background()
		ctx = authentication.SetUserIdInMetadata(ctx, user.ID.String())

		authService.SetupCanvasRoles(r.Canvas.ID.String())
		authService.AssignRole(user.ID.String(), authorization.RoleCanvasOwner, r.Canvas.ID.String(), authorization.DomainCanvas)

		res, err := ListCanvases(ctx, &protos.ListCanvasesRequest{}, authService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Canvases, 1)
		require.NotNil(t, res.Canvases[0].Metadata)
		assert.Equal(t, r.Canvas.ID.String(), res.Canvases[0].Metadata.Id)
		assert.Equal(t, r.Canvas.Name, res.Canvases[0].Metadata.Name)
		assert.Equal(t, r.Canvas.CreatedBy.String(), res.Canvases[0].Metadata.CreatedBy)
		assert.NotNil(t, res.Canvases[0].Metadata.CreatedAt)
	})

	t.Run("with organization ID -> list canvases from organization", func(t *testing.T) {
		ctx := context.Background()
		ctx = authentication.SetUserIdInMetadata(ctx, user.ID.String())

		authService.SetupCanvasRoles(r.Canvas.ID.String())
		authService.AssignRole(user.ID.String(), authorization.RoleCanvasOwner, r.Canvas.ID.String(), authorization.DomainCanvas)

		res, err := ListCanvases(ctx, &protos.ListCanvasesRequest{
			OrganizationId: r.Organization.ID.String(),
		}, authService)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Canvases, 1)
		require.NotNil(t, res.Canvases[0].Metadata)
		assert.Equal(t, r.Canvas.ID.String(), res.Canvases[0].Metadata.Id)
		assert.Equal(t, r.Canvas.Name, res.Canvases[0].Metadata.Name)
		assert.Equal(t, r.Canvas.CreatedBy.String(), res.Canvases[0].Metadata.CreatedBy)
		assert.NotNil(t, res.Canvases[0].Metadata.CreatedAt)
	})

	t.Run("Organization with no canvases -> empty list", func(t *testing.T) {
		ctx := context.Background()
		ctx = authentication.SetUserIdInMetadata(ctx, user.ID.String())

		res, err := ListCanvases(ctx, &protos.ListCanvasesRequest{
			OrganizationId: uuid.New().String(),
		}, authService)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Canvases)
	})
}
