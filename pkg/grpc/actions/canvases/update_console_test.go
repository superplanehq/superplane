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
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

func updateConsoleFromProto(
	ctx context.Context,
	organizationID, canvasID, versionID string,
	panels []*pb.Console_Panel,
	layout []*pb.Console_LayoutItem,
) (*models.CanvasVersion, error) {
	modelPanels, err := deserializeConsolePanels(panels)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}
	modelLayout := deserializeConsoleLayout(layout)
	return UpdateConsole(ctx, organizationID, canvasID, versionID, modelPanels, modelLayout, false)
}

func Test__UpdateConsole(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := updateConsoleFromProto(ctx, "not-a-uuid", uuid.New().String(), "", nil, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := updateConsoleFromProto(ctx, orgID, "bad", "", nil, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := updateConsoleFromProto(ctx, orgID, uuid.New().String(), "", nil, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("panel content must be an object", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		strVal, err := structpb.NewValue("not-an-object")
		require.NoError(t, err)
		_, err = updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "x", Type: "markdown", Content: strVal},
		}, []*pb.Console_LayoutItem{{I: "x", X: 0, Y: 0, W: 1, H: 1}})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: panel id required", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "", Type: "markdown"},
		}, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: panel type required", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: ""},
		}, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: duplicate panel id", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "dup", Type: "markdown"},
			{Id: "dup", Type: "markdown"},
		}, nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: layout i required", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: "markdown"},
		}, []*pb.Console_LayoutItem{{I: "", X: 0, Y: 0, W: 1, H: 1}})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: duplicate layout id", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: "markdown"},
		}, []*pb.Console_LayoutItem{
			{I: "p", X: 0, Y: 0, W: 1, H: 1},
			{I: "p", X: 1, Y: 0, W: 1, H: 1},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: layout references unknown panel", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: "markdown"},
		}, []*pb.Console_LayoutItem{{I: "other", X: 0, Y: 0, W: 1, H: 1}})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: layout w/h must be positive", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: "markdown"},
		}, []*pb.Console_LayoutItem{{I: "p", X: 0, Y: 0, W: 0, H: 1}})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: layout x/y must be non-negative", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: "markdown"},
		}, []*pb.Console_LayoutItem{{I: "p", X: -1, Y: 0, W: 1, H: 1}})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: too many panels", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		panels := make([]*pb.Console_Panel, 0, MaxConsolePanels+1)
		layout := make([]*pb.Console_LayoutItem, 0, MaxConsolePanels+1)
		for i := range MaxConsolePanels + 1 {
			id := uuid.New().String()
			panels = append(panels, &pb.Console_Panel{Id: id, Type: "markdown"})
			layout = append(layout, &pb.Console_LayoutItem{I: id, X: int32(i), Y: 0, W: 1, H: 1})
		}
		_, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", panels, layout)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("validation: panels payload too large", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		huge := strings.Repeat("x", MaxConsolePayloadBytes+1)
		content, err := structpb.NewValue(map[string]any{"body": huge})
		require.NoError(t, err)
		_, err = updateConsoleFromProto(ctx, orgID, canvas.ID.String(), "", []*pb.Console_Panel{
			{Id: "p", Type: "markdown", Content: content},
		}, []*pb.Console_LayoutItem{{I: "p", X: 0, Y: 0, W: 1, H: 1}})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("persists and returns console", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		versionID := consoleDraftVersionID(t, canvas.ID, r.User)
		content, err := structpb.NewValue(map[string]any{"body": "hello"})
		require.NoError(t, err)
		resp, err := updateConsoleFromProto(ctx, orgID, canvas.ID.String(), versionID, []*pb.Console_Panel{
			{Id: "a", Type: "markdown", Content: content},
			{Id: "b", Type: "markdown"},
		}, []*pb.Console_LayoutItem{
			{I: "a", X: 0, Y: 0, W: 2, H: 2},
			{I: "b", X: 2, Y: 0, W: 2, H: 2},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		yamlText, err := consoleYAMLFromVersion(resp)
		require.NoError(t, err)
		doc, err := models.ConsoleFromYML([]byte(yamlText))
		require.NoError(t, err)
		require.Len(t, doc.Spec.Panels, 2)
		assert.NotNil(t, doc.Spec.Panels[0].Content)
		assert.NotNil(t, doc.Spec.Panels[1].Content)
		require.Len(t, doc.Spec.Layout, 2)
		assert.Equal(t, versionID, resp.ID.String())

		got, err := ReadRepositorySpecFile(ctx, orgID, canvas.ID.String(), versionID, ConsoleYAMLRepositoryPath)
		require.NoError(t, err)
		gotDoc, err := models.ConsoleFromYML([]byte(got))
		require.NoError(t, err)
		assert.Len(t, gotDoc.Spec.Panels, 2)
		assert.Len(t, gotDoc.Spec.Layout, 2)

		live, err := ReadRepositorySpecFile(ctx, orgID, canvas.ID.String(), "", ConsoleYAMLRepositoryPath)
		require.NoError(t, err)
		liveDoc, err := models.ConsoleFromYML([]byte(live))
		require.NoError(t, err)
		assert.Empty(t, liveDoc.Spec.Panels)
		assert.Empty(t, liveDoc.Spec.Layout)
	})
}

func consoleDraftVersionID(t *testing.T, canvasID, userID uuid.UUID) string {
	t.Helper()

	draft, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvasID, userID, "", nil, nil)
	require.NoError(t, err)
	return draft.ID.String()
}
