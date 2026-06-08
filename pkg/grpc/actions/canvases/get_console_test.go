package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__GetConsole(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := GetConsole(ctx, "not-a-uuid", uuid.New().String(), "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := GetConsole(ctx, orgID, "bad-canvas", "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := GetConsole(ctx, orgID, uuid.New().String(), "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("empty console when none stored", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		resp, err := GetConsole(ctx, orgID, canvas.ID.String(), "")
		require.NoError(t, err)
		require.NotNil(t, resp.GetConsole())
		assert.Equal(t, canvas.ID.String(), resp.GetConsole().GetCanvasId())
		assert.Empty(t, resp.GetConsole().GetPanels())
		assert.Empty(t, resp.GetConsole().GetLayout())
	})

	t.Run("returns stored panels and layout", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		minW, minH := 2, 1

		_, err := models.UpsertCanvasVersionConsole(canvas.ID, []models.ConsolePanel{
			{ID: "p1", Type: "markdown", Content: map[string]any{"body": "hi", "nested": []any{1, map[string]any{"k": "v"}}}},
		}, []models.ConsoleLayoutItem{
			{I: "p1", X: 0, Y: 0, W: 4, H: 2, MinW: &minW, MinH: &minH},
		})

		require.NoError(t, err)

		resp, err := GetConsole(ctx, orgID, canvas.ID.String(), "")
		require.NoError(t, err)
		d := resp.GetConsole()
		require.Len(t, d.GetPanels(), 1)
		assert.Equal(t, "p1", d.GetPanels()[0].GetId())
		// The string stored in the model maps to the proto enum on the wire.
		assert.Equal(t, pb.Console_Panel_MARKDOWN, d.GetPanels()[0].GetType())
		require.NotNil(t, d.GetPanels()[0].GetContent())
		require.Len(t, d.GetLayout(), 1)
		assert.Equal(t, "p1", d.GetLayout()[0].GetI())
		assert.EqualValues(t, 4, d.GetLayout()[0].GetW())
		require.NotNil(t, d.GetLayout()[0].MinW)
		require.NotNil(t, d.GetLayout()[0].MinH)
		assert.EqualValues(t, 2, *d.GetLayout()[0].MinW)
		assert.EqualValues(t, 1, *d.GetLayout()[0].MinH)
		require.NotNil(t, d.GetUpdatedAt())
	})
}
