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

func Test__ValidateCanvas(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("returns unauthenticated when user not in context", func(t *testing.T) {
		resp, err := ValidateCanvas(context.Background(), r.Registry, r.Organization.ID, &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "test"},
			Spec:     &pb.Canvas_Spec{Nodes: []*componentpb.Node{}, Edges: []*componentpb.Edge{}},
		})
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.Nil(t, resp)
	})

	t.Run("returns invalid argument when canvas is nil", func(t *testing.T) {
		resp, err := ValidateCanvas(ctx, r.Registry, r.Organization.ID, nil)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, resp)
	})

	t.Run("returns invalid argument when canvas name is blank", func(t *testing.T) {
		resp, err := ValidateCanvas(ctx, r.Registry, r.Organization.ID, &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "   "},
			Spec:     &pb.Canvas_Spec{Nodes: []*componentpb.Node{}, Edges: []*componentpb.Edge{}},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Equal(t, "canvas name is required", status.Convert(err).Message())
		assert.Nil(t, resp)
	})

	t.Run("validates empty canvas without persisting anything", func(t *testing.T) {
		var countBefore int64
		require.NoError(t, database.Conn().Model(&models.CanvasVersion{}).Count(&countBefore).Error)

		resp, err := ValidateCanvas(ctx, r.Registry, r.Organization.ID, &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "empty-canvas"},
			Spec:     &pb.Canvas_Spec{Nodes: []*componentpb.Node{}, Edges: []*componentpb.Edge{}},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Version)
		assert.Empty(t, resp.Version.Spec.Nodes)
		assert.Empty(t, resp.Version.Spec.Edges)

		var countAfter int64
		require.NoError(t, database.Conn().Model(&models.CanvasVersion{}).Count(&countAfter).Error)
		assert.Equal(t, countBefore, countAfter, "no canvas versions should be persisted")
	})

	t.Run("validates canvas with nodes without persisting anything", func(t *testing.T) {
		var countBefore int64
		require.NoError(t, database.Conn().Model(&models.CanvasVersion{}).Count(&countBefore).Error)

		resp, err := ValidateCanvas(ctx, r.Registry, r.Organization.ID, &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "test-canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{Id: "node-a", Name: "Node A", Component: "noop"},
					{Id: "node-b", Name: "Node B", Component: "noop"},
				},
				Edges: []*componentpb.Edge{
					{SourceId: "node-a", TargetId: "node-b", Channel: "default"},
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Version)
		require.Len(t, resp.Version.Spec.Nodes, 2)
		require.Len(t, resp.Version.Spec.Edges, 1)

		nodeA := findProtoNode(resp.Version.Spec.Nodes, "node-a")
		require.NotNil(t, nodeA)
		assert.Equal(t, "Node A", nodeA.Name)
		assert.Nil(t, nodeA.ErrorMessage)

		nodeB := findProtoNode(resp.Version.Spec.Nodes, "node-b")
		require.NotNil(t, nodeB)
		assert.Equal(t, "Node B", nodeB.Name)
		assert.Nil(t, nodeB.ErrorMessage)

		edge := resp.Version.Spec.Edges[0]
		assert.Equal(t, "node-a", edge.SourceId)
		assert.Equal(t, "node-b", edge.TargetId)
		assert.Equal(t, "default", edge.Channel)

		var countAfter int64
		require.NoError(t, database.Conn().Model(&models.CanvasVersion{}).Count(&countAfter).Error)
		assert.Equal(t, countBefore, countAfter, "no canvas versions should be persisted")
	})

	t.Run("returns invalid argument for canvas with cycles", func(t *testing.T) {
		resp, err := ValidateCanvas(ctx, r.Registry, r.Organization.ID, &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "cyclic-canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{Id: "node-a", Name: "Node A", Component: "noop"},
					{Id: "node-b", Name: "Node B", Component: "noop"},
				},
				Edges: []*componentpb.Edge{
					{SourceId: "node-a", TargetId: "node-b", Channel: "default"},
					{SourceId: "node-b", TargetId: "node-a", Channel: "default"},
				},
			},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, resp)
	})
}
