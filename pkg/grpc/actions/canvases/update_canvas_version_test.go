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
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateCanvasVersion(t *testing.T) {
	r := support.Setup(t)

	t.Run("versioning enabled at canvas level and no version id -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(
			t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("canvas_versioning_enabled", false).Error,
		)
		require.NoError(
			t,
			database.Conn().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("canvas_versioning_enabled", true).Error,
		)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"",
			testPbCanvas(canvas.Name),
			nil,
			"",
		)

		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
		assert.Contains(t, s.Message(), "canvas versioning is enabled")
	})

	t.Run("versioning enabled at org level and no version id -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(
			t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("canvas_versioning_enabled", true).Error,
		)
		require.NoError(
			t,
			database.Conn().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("canvas_versioning_enabled", false).Error,
		)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"",
			testPbCanvas(canvas.Name),
			nil,
			"",
		)

		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
		assert.Contains(t, s.Message(), "canvas versioning is enabled")
	})

	t.Run("versioning disabled and version id provided -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(
			t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("canvas_versioning_enabled", false).Error,
		)
		require.NoError(
			t,
			database.Conn().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("canvas_versioning_enabled", false).Error,
		)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			canvas.LiveVersionID.String(),
			testPbCanvas(canvas.Name),
			nil,
			"",
		)

		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
		assert.Contains(t, s.Message(), "canvas versioning is disabled")
	})

	t.Run("versioning disabled and no version id -> updates live canvas", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(
			t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("canvas_versioning_enabled", false).Error,
		)
		require.NoError(
			t,
			database.Conn().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("canvas_versioning_enabled", false).Error,
		)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		response, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"",
			testPbCanvas(canvas.Name),
			nil,
			"",
		)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Version)
	})

	t.Run("versioning enabled and valid draft version id -> updates draft", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(
			t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("canvas_versioning_enabled", false).Error,
		)
		require.NoError(
			t,
			database.Conn().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("canvas_versioning_enabled", true).Error,
		)

		draftVersion, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvas.ID, r.User, nil, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		response, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			draftVersion.ID.String(),
			testPbCanvas(canvas.Name),
			nil,
			"",
		)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Version)
	})
}

func testPbCanvas(name string) *pb.Canvas {
	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: name,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}
}
