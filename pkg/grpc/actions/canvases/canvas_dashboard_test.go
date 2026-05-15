package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__GetCanvasDashboard(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := GetCanvasDashboard(ctx, "not-a-uuid", uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := GetCanvasDashboard(ctx, orgID, "bad-canvas")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := GetCanvasDashboard(ctx, orgID, uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("empty dashboard when none stored", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		resp, err := GetCanvasDashboard(ctx, orgID, canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, resp.GetDashboard())
		assert.Equal(t, canvas.ID.String(), resp.GetDashboard().GetCanvasId())
		assert.Empty(t, resp.GetDashboard().GetPanels())
		assert.Empty(t, resp.GetDashboard().GetLayout())
		assert.Nil(t, resp.GetDashboard().GetUpdatedAt())
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

		resp, err := GetCanvasDashboard(ctx, orgID, canvas.ID.String())
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

func Test__UpdateCanvasDashboard(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := UpdateCanvasDashboard(ctx, "not-a-uuid", uuid.New().String(), nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := UpdateCanvasDashboard(ctx, orgID, "bad", nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := UpdateCanvasDashboard(ctx, orgID, uuid.New().String(), nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("template canvas is read-only", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		require.NoError(t, database.Conn().Model(&models.Canvas{}).
			Where("id = ?", canvas.ID).
			Update("is_template", true).Error)

		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "a", Type: "markdown"},
		}, []*pb.DashboardLayoutItem{{I: "a", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
	})

	t.Run("panel content must be an object", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		strVal, err := structpb.NewValue("not-an-object")
		require.NoError(t, err)
		_, err = UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "x", Type: "markdown", Content: strVal},
		}, []*pb.DashboardLayoutItem{{I: "x", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: panel id required", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "", Type: "markdown"},
		}, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: panel type required", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "p", Type: ""},
		}, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: duplicate panel id", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "dup", Type: "markdown"},
			{Id: "dup", Type: "markdown"},
		}, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: layout i required", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "p", Type: "markdown"},
		}, []*pb.DashboardLayoutItem{{I: "", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: duplicate layout id", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "p", Type: "markdown"},
		}, []*pb.DashboardLayoutItem{
			{I: "p", X: 0, Y: 0, W: 1, H: 1},
			{I: "p", X: 1, Y: 0, W: 1, H: 1},
		})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: layout references unknown panel", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "p", Type: "markdown"},
		}, []*pb.DashboardLayoutItem{{I: "other", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: layout w/h must be positive", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "p", Type: "markdown"},
		}, []*pb.DashboardLayoutItem{{I: "p", X: 0, Y: 0, W: 0, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: layout x/y must be non-negative", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "p", Type: "markdown"},
		}, []*pb.DashboardLayoutItem{{I: "p", X: -1, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: too many panels", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		panels := make([]*pb.DashboardPanel, 0, MaxDashboardPanels+1)
		layout := make([]*pb.DashboardLayoutItem, 0, MaxDashboardPanels+1)
		for i := range MaxDashboardPanels + 1 {
			id := uuid.New().String()
			panels = append(panels, &pb.DashboardPanel{Id: id, Type: "markdown"})
			layout = append(layout, &pb.DashboardLayoutItem{I: id, X: int32(i), Y: 0, W: 1, H: 1})
		}
		_, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), panels, layout)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: panels payload too large", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		huge := strings.Repeat("x", MaxDashboardPayloadBytes+1)
		content, err := structpb.NewValue(map[string]any{"body": huge})
		require.NoError(t, err)
		_, err = UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "p", Type: "markdown", Content: content},
		}, []*pb.DashboardLayoutItem{{I: "p", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("persists and returns dashboard", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		content, err := structpb.NewValue(map[string]any{"body": "hello"})
		require.NoError(t, err)
		resp, err := UpdateCanvasDashboard(ctx, orgID, canvas.ID.String(), []*pb.DashboardPanel{
			{Id: "a", Type: "markdown", Content: content},
			{Id: "b", Type: "markdown"},
		}, []*pb.DashboardLayoutItem{
			{I: "a", X: 0, Y: 0, W: 2, H: 2},
			{I: "b", X: 2, Y: 0, W: 2, H: 2},
		})
		require.NoError(t, err)
		d := resp.GetDashboard()
		require.Len(t, d.GetPanels(), 2)
		assert.NotNil(t, d.GetPanels()[0].GetContent())
		assert.NotNil(t, d.GetPanels()[1].GetContent())
		require.Len(t, d.GetLayout(), 2)

		got, err := GetCanvasDashboard(ctx, orgID, canvas.ID.String())
		require.NoError(t, err)
		assert.Len(t, got.GetDashboard().GetPanels(), 2)
		assert.Len(t, got.GetDashboard().GetLayout(), 2)
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
