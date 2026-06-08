package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__UpdateConsole(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := UpdateConsole(ctx, "not-a-uuid", uuid.New().String(), "", nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := UpdateConsole(ctx, orgID, "bad", "", nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := UpdateConsole(ctx, orgID, uuid.New().String(), "", nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("template canvas is read-only", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		require.NoError(t, database.Conn().Model(&models.Canvas{}).
			Where("id = ?", canvas.ID).
			Update("is_template", true).Error)

		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "a", Type: pb.Console_Panel_MARKDOWN},
		}, []*pb.Console_LayoutItem{{I: "a", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
	})

	t.Run("panel content must be an object", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		strVal, err := structpb.NewValue("not-an-object")
		require.NoError(t, err)
		_, err = UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "x", Type: pb.Console_Panel_MARKDOWN, Content: strVal},
		}, []*pb.Console_LayoutItem{{I: "x", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: panel id required", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "", Type: pb.Console_Panel_MARKDOWN},
		}, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: panel type unspecified is rejected fail-closed", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		// TYPE_UNSPECIFIED is the proto3 zero value. The wire boundary
		// helper `panelTypeToModel` must surface it as a clear
		// `InvalidArgument` instead of silently defaulting the panel to
		// markdown — that's the whole point of the fail-closed mapping.
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: pb.Console_Panel_TYPE_UNSPECIFIED},
		}, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "panel type is required")
	})

	t.Run("validation: duplicate panel id", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "dup", Type: pb.Console_Panel_MARKDOWN},
			{Id: "dup", Type: pb.Console_Panel_MARKDOWN},
		}, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: layout i required", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: pb.Console_Panel_MARKDOWN},
		}, []*pb.Console_LayoutItem{{I: "", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: duplicate layout id", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: pb.Console_Panel_MARKDOWN},
		}, []*pb.Console_LayoutItem{
			{I: "p", X: 0, Y: 0, W: 1, H: 1},
			{I: "p", X: 1, Y: 0, W: 1, H: 1},
		})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: layout references unknown panel", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: pb.Console_Panel_MARKDOWN},
		}, []*pb.Console_LayoutItem{{I: "other", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: layout w/h must be positive", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: pb.Console_Panel_MARKDOWN},
		}, []*pb.Console_LayoutItem{{I: "p", X: 0, Y: 0, W: 0, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: layout x/y must be non-negative", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: pb.Console_Panel_MARKDOWN},
		}, []*pb.Console_LayoutItem{{I: "p", X: -1, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: too many panels", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		panels := make([]*pb.Console_Panel, 0, MaxConsolePanels+1)
		layout := make([]*pb.Console_LayoutItem, 0, MaxConsolePanels+1)
		for i := range MaxConsolePanels + 1 {
			id := uuid.New().String()
			panels = append(panels, &pb.Console_Panel{Id: id, Type: pb.Console_Panel_MARKDOWN})
			layout = append(layout, &pb.Console_LayoutItem{I: id, X: int32(i), Y: 0, W: 1, H: 1})
		}
		_, err := UpdateConsole(ctx, orgID, canvas.ID.String(), "", panels, layout)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("validation: panels payload too large", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		huge := strings.Repeat("x", MaxConsolePayloadBytes+1)
		content, err := structpb.NewValue(map[string]any{"body": huge})
		require.NoError(t, err)
		_, err = UpdateConsole(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: pb.Console_Panel_MARKDOWN, Content: content},
		}, []*pb.Console_LayoutItem{{I: "p", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("persists and returns console", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		versionID := consoleDraftVersionID(t, canvas.ID, r.User)
		content, err := structpb.NewValue(map[string]any{"body": "hello"})
		require.NoError(t, err)
		resp, err := UpdateConsole(ctx, orgID, canvas.ID.String(), versionID, []*pb.Console_Panel{
			{Id: "a", Type: pb.Console_Panel_MARKDOWN, Content: content},
			{Id: "b", Type: pb.Console_Panel_MARKDOWN},
		}, []*pb.Console_LayoutItem{
			{I: "a", X: 0, Y: 0, W: 2, H: 2},
			{I: "b", X: 2, Y: 0, W: 2, H: 2},
		})
		require.NoError(t, err)
		d := resp.GetConsole()
		require.Len(t, d.GetPanels(), 2)
		assert.NotNil(t, d.GetPanels()[0].GetContent())
		assert.NotNil(t, d.GetPanels()[1].GetContent())
		// Round-trip: the enum survives serialize -> persist -> deserialize.
		assert.Equal(t, pb.Console_Panel_MARKDOWN, d.GetPanels()[0].GetType())
		assert.Equal(t, pb.Console_Panel_MARKDOWN, d.GetPanels()[1].GetType())
		require.Len(t, d.GetLayout(), 2)
		assert.Equal(t, versionID, d.GetVersionId())

		got, err := GetConsole(ctx, orgID, canvas.ID.String(), versionID)
		require.NoError(t, err)
		assert.Len(t, got.GetConsole().GetPanels(), 2)
		assert.Len(t, got.GetConsole().GetLayout(), 2)
		assert.Equal(t, pb.Console_Panel_MARKDOWN, got.GetConsole().GetPanels()[0].GetType())

		live, err := GetConsole(ctx, orgID, canvas.ID.String(), "")
		require.NoError(t, err)
		assert.Empty(t, live.GetConsole().GetPanels())
		assert.Empty(t, live.GetConsole().GetLayout())
	})
}

func consoleDraftVersionID(t *testing.T, canvasID, userID uuid.UUID) string {
	t.Helper()

	draft, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvasID, userID, "", nil, nil)
	require.NoError(t, err)
	return draft.ID.String()
}
