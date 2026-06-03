package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func dashboardDraftVersionID(t *testing.T, canvasID, userID uuid.UUID) string {
	t.Helper()

	draft, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvasID, userID, "", nil, nil)
	require.NoError(t, err)
	return draft.ID
}

func Test__GetCanvasDashboard(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := GetCanvasDashboard(ctx, "not-a-uuid", uuid.New().String(), "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := GetCanvasDashboard(ctx, orgID, "bad-canvas", "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := GetCanvasDashboard(ctx, orgID, uuid.New().String(), "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("empty dashboard when none stored", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		resp, err := GetCanvasDashboard(ctx, orgID, canvas.ID.String(), "")
		require.NoError(t, err)
		require.NotNil(t, resp.GetDashboard())
		assert.Equal(t, canvas.ID.String(), resp.GetDashboard().GetCanvasId())
		assert.Empty(t, resp.GetDashboard().GetPanels())
		assert.Empty(t, resp.GetDashboard().GetLayout())
	})

	t.Run("returns stored panels and layout", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		minW, minH := 2, 1
		_, err := models.UpsertCanvasDashboard(canvas.ID, []models.DashboardPanel{
			{ID: "p1", Type: "markdown", Content: map[string]any{"body": "hi", "nested": []any{1, map[string]any{"k": "v"}}}},
		}, []models.DashboardLayoutItem{
			{I: "p1", X: 0, Y: 0, W: 4, H: 2, MinW: &minW, MinH: &minH},
		})
		require.NoError(t, err)

		resp, err := GetCanvasDashboard(ctx, orgID, canvas.ID.String(), "")
		require.NoError(t, err)
		d := resp.GetDashboard()
		require.Len(t, d.GetPanels(), 1)
		assert.Equal(t, "p1", d.GetPanels()[0].GetId())
		assert.Equal(t, "markdown", d.GetPanels()[0].GetType())
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

func Test__toStructpbCompatible(t *testing.T) {
	in := map[string]any{
		"arr": []any{float64(1), "two"},
		"nested": map[string]any{
			"k": true,
		},
	}
	out := toStructpbCompatible(in)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	arr, ok := m["arr"].([]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), arr[0])
}
