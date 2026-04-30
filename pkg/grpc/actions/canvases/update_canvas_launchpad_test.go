package canvases

import (
	"context"
	"testing"

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

func newMarkdownPanel(t *testing.T, id, body string) *pb.LaunchpadPanel {
	t.Helper()
	content, err := structpb.NewValue(map[string]any{"body": body})
	require.NoError(t, err)
	return &pb.LaunchpadPanel{Id: id, Type: "markdown", Content: content}
}

func Test__UpdateCanvasLaunchpad(t *testing.T) {
	r := support.Setup(t)

	t.Run("creates a new launchpad row on first call", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		panels := []*pb.LaunchpadPanel{newMarkdownPanel(t, "p1", "# Hello")}
		layout := []*pb.LaunchpadLayoutItem{{I: "p1", X: 0, Y: 0, W: 6, H: 4}}

		resp, err := UpdateCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String(), panels, layout)
		require.NoError(t, err)
		require.NotNil(t, resp.Launchpad)
		require.Len(t, resp.Launchpad.Panels, 1)

		stored, err := models.FindCanvasLaunchpadInTransaction(database.Conn(), canvas.ID)
		require.NoError(t, err)
		require.Len(t, stored.Panels.Data(), 1)
		assert.Equal(t, "p1", stored.Panels.Data()[0].ID)
	})

	t.Run("replaces panels and layout on subsequent call", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		_, err := UpdateCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String(),
			[]*pb.LaunchpadPanel{newMarkdownPanel(t, "p1", "first")},
			[]*pb.LaunchpadLayoutItem{{I: "p1", X: 0, Y: 0, W: 4, H: 3}},
		)
		require.NoError(t, err)

		_, err = UpdateCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String(),
			[]*pb.LaunchpadPanel{
				newMarkdownPanel(t, "p2", "second"),
				newMarkdownPanel(t, "p3", "third"),
			},
			[]*pb.LaunchpadLayoutItem{
				{I: "p2", X: 0, Y: 0, W: 6, H: 4},
				{I: "p3", X: 6, Y: 0, W: 6, H: 4},
			},
		)
		require.NoError(t, err)

		stored, err := models.FindCanvasLaunchpadInTransaction(database.Conn(), canvas.ID)
		require.NoError(t, err)
		require.Len(t, stored.Panels.Data(), 2)
		ids := []string{stored.Panels.Data()[0].ID, stored.Panels.Data()[1].ID}
		assert.ElementsMatch(t, []string{"p2", "p3"}, ids)
	})

	t.Run("rejects layout item that does not reference any panel", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		_, err := UpdateCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String(),
			[]*pb.LaunchpadPanel{newMarkdownPanel(t, "p1", "x")},
			[]*pb.LaunchpadLayoutItem{{I: "ghost", X: 0, Y: 0, W: 4, H: 3}},
		)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("rejects duplicate panel ids", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		_, err := UpdateCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String(),
			[]*pb.LaunchpadPanel{
				newMarkdownPanel(t, "p1", "a"),
				newMarkdownPanel(t, "p1", "b"),
			},
			[]*pb.LaunchpadLayoutItem{},
		)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("rejects zero-size layout item", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		_, err := UpdateCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String(),
			[]*pb.LaunchpadPanel{newMarkdownPanel(t, "p1", "x")},
			[]*pb.LaunchpadLayoutItem{{I: "p1", X: 0, Y: 0, W: 0, H: 4}},
		)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("rejects template canvases", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(t, database.Conn().Model(&models.Canvas{}).
			Where("id = ?", canvas.ID).
			Update("is_template", true).Error)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := UpdateCanvasLaunchpad(ctx, r.Organization.ID.String(), canvas.ID.String(),
			[]*pb.LaunchpadPanel{newMarkdownPanel(t, "p1", "x")},
			[]*pb.LaunchpadLayoutItem{{I: "p1", X: 0, Y: 0, W: 4, H: 3}},
		)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
	})

	t.Run("rejects invalid canvas id", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := UpdateCanvasLaunchpad(ctx, r.Organization.ID.String(), "not-a-uuid", nil, nil)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})
}
