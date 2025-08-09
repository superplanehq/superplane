package canvases

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateCanvas(t *testing.T) {
	r := support.Setup(t)
	user := uuid.New()
	ctx := authentication.SetUserIdInMetadata(context.Background(), user.String())

	t.Run("name still not used -> canvas is created", func(t *testing.T) {
		response, err := CreateCanvas(ctx, &protos.CreateCanvasRequest{
			OrganizationId: r.Organization.ID.String(),
			Canvas:         &protos.Canvas{Metadata: &protos.Canvas_Metadata{Name: "test"}},
		}, r.AuthService)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		assert.NotEmpty(t, response.Canvas.Metadata.Id)
		assert.NotEmpty(t, response.Canvas.Metadata.CreatedAt)
		assert.Equal(t, "test", response.Canvas.Metadata.Name)
	})

	t.Run("name already used -> error", func(t *testing.T) {
		_, err := CreateCanvas(ctx, &protos.CreateCanvasRequest{
			Canvas:         &protos.Canvas{Metadata: &protos.Canvas_Metadata{Name: "test"}},
			OrganizationId: r.Organization.ID.String(),
		}, r.AuthService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
