package actions

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateCanvas(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	orgID := uuid.New()
	user := uuid.New()

	t.Run("name still not used -> canvas is created", func(t *testing.T) {
		response, err := CreateCanvas(context.Background(), &protos.CreateCanvasRequest{
			OrganizationId: orgID.String(),
			RequesterId:    user.String(),
			Name:           "test",
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		assert.NotEmpty(t, response.Canvas.Id)
		assert.NotEmpty(t, response.Canvas.CreatedAt)
		assert.Equal(t, "test", response.Canvas.Name)
		assert.Equal(t, orgID.String(), response.Canvas.OrganizationId)
	})

	t.Run("name already used -> error", func(t *testing.T) {
		_, err := CreateCanvas(context.Background(), &protos.CreateCanvasRequest{
			OrganizationId: orgID.String(),
			RequesterId:    user.String(),
			Name:           "test",
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
