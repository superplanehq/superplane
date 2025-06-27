package canvases

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateCanvas(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	user := uuid.New()
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(context.Background(), user.String())
	org, err := models.CreateOrganization(user, "test", "test")
	require.NoError(t, err)

	t.Run("name still not used -> canvas is created", func(t *testing.T) {
		// Create a Canvas with nested metadata structure
		canvas := &protos.Canvas{
			Metadata: &protos.Canvas_Metadata{
				Name: "test",
			},
		}

		response, err := CreateCanvas(ctx, &protos.CreateCanvasRequest{
			Canvas:         canvas,
			OrganizationId: org.ID.String(),
		}, authService)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		assert.NotEmpty(t, response.Canvas.Metadata.Id)
		assert.NotEmpty(t, response.Canvas.Metadata.CreatedAt)
		assert.Equal(t, "test", response.Canvas.Metadata.Name)
	})

	t.Run("name already used -> error", func(t *testing.T) {
		// Create a Canvas with nested metadata structure
		canvas := &protos.Canvas{
			Metadata: &protos.Canvas_Metadata{
				Name: "test",
			},
		}

		_, err := CreateCanvas(ctx, &protos.CreateCanvasRequest{
			Canvas:         canvas,
			OrganizationId: org.ID.String(),
		}, authService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
