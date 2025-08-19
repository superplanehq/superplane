package canvases

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DeleteCanvas(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	userID := uuid.New()
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := DeleteCanvas(context.Background(), uuid.New().String(), &protos.DeleteCanvasRequest{
			IdOrName: uuid.New().String(),
		}, authService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("delete canvas successfully", func(t *testing.T) {
		organization, err := models.CreateOrganization("test-org", "Test Organization", "")
		require.NoError(t, err)
		canvas, err := models.CreateCanvas(userID, organization.ID, "test", "test")
		require.NoError(t, err)
		err = authService.SetupCanvasRoles(canvas.ID.String())
		require.NoError(t, err)

		response, err := DeleteCanvas(context.Background(), organization.ID.String(), &protos.DeleteCanvasRequest{
			IdOrName: canvas.ID.String(),
		}, authService)

		require.NoError(t, err)
		require.NotNil(t, response)

		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvas.ID.String())
		require.NoError(t, err)
		require.Empty(t, roles)

		deletedCanvas, err := models.FindUnscopedCanvasByID(canvas.ID.String())
		require.NoError(t, err)
		assert.Contains(t, deletedCanvas.Name, "deleted-")
	})
}
