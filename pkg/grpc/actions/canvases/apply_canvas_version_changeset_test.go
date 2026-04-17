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
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

func Test__ApplyCanvasVersionChangeset(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns unauthenticated without user metadata", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := ApplyCanvasVersionChangeset(
			context.Background(),
			r.Registry,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID,
			canvas.ID,
			*canvas.LiveVersionID,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns invalid argument for invalid user id", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		ctx := authentication.SetUserIdInMetadata(context.Background(), "not-a-uuid")

		_, err := ApplyCanvasVersionChangeset(
			ctx,
			r.Registry,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID,
			canvas.ID,
			*canvas.LiveVersionID,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("returns invalid argument for empty changeset", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		_, err := ApplyCanvasVersionChangeset(
			ctx,
			r.Registry,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID,
			canvas.ID,
			*canvas.LiveVersionID,
			&pb.CanvasChangeset{},
			nil,
		)

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "changeset is required")
	})

	t.Run("returns not found when version does not exist", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		_, err := ApplyCanvasVersionChangeset(
			ctx,
			r.Registry,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID,
			canvas.ID,
			uuid.New(),
			&pb.CanvasChangeset{
				Changes: []*pb.CanvasChangeset_Change{
					{
						Type: pb.CanvasChangeset_Change_ADD_NODE,
						Node: &pb.CanvasChangeset_Change_Node{
							Id: "node-a",
						},
					},
				},
			},
			nil,
		)

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("applies changeset and persists patched version", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				testCanvasNode("node-a", "Node A", map[string]any{"foo": "before"}),
				testCanvasNode("node-b", "Node B", map[string]any{"bar": "value"}),
			},
			[]models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}},
		)

		otherUser := support.CreateUser(t, r, r.Organization.ID)
		draftVersion := createCanvasDraftVersionFromLive(t, canvas.ID, *canvas.LiveVersionID, otherUser.ID)
		ctx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

		response, err := ApplyCanvasVersionChangeset(
			ctx,
			r.Registry,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID,
			canvas.ID,
			draftVersion.ID,
			&pb.CanvasChangeset{
				Changes: []*pb.CanvasChangeset_Change{
					{
						Type: pb.CanvasChangeset_Change_ADD_NODE,
						Node: &pb.CanvasChangeset_Change_Node{
							Id:            "node-c",
							Name:          "Node C",
							Block:         "noop",
							Configuration: structFromAnyMap(t, map[string]any{"baz": "value"}),
						},
					},
					{
						Type: pb.CanvasChangeset_Change_UPDATE_NODE,
						Node: &pb.CanvasChangeset_Change_Node{
							Id:            "node-a",
							Name:          "Node A Updated",
							Configuration: structFromAnyMap(t, map[string]any{"foo": "after"}),
						},
					},
					{
						Type: pb.CanvasChangeset_Change_DELETE_EDGE,
						Edge: &pb.CanvasChangeset_Change_Edge{
							SourceId: "node-a",
							TargetId: "node-b",
							Channel:  "default",
						},
					},
					{
						Type: pb.CanvasChangeset_Change_ADD_EDGE,
						Edge: &pb.CanvasChangeset_Change_Edge{
							SourceId: "node-a",
							TargetId: "node-c",
							Channel:  "default",
						},
					},
					{
						Type: pb.CanvasChangeset_Change_DELETE_NODE,
						Node: &pb.CanvasChangeset_Change_Node{Id: "node-b"},
					},
				},
			},
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Version)
		require.NotNil(t, response.Version.Spec)
		require.Len(t, response.Version.Spec.Nodes, 2)
		require.Len(t, response.Version.Spec.Edges, 1)

		nodeA := findProtoNode(response.Version.Spec.Nodes, "node-a")
		require.NotNil(t, nodeA)
		assert.Equal(t, "Node A Updated", nodeA.Name)
		assert.Equal(t, "after", nodeA.Configuration.AsMap()["foo"])

		nodeC := findProtoNode(response.Version.Spec.Nodes, "node-c")
		require.NotNil(t, nodeC)
		assert.Equal(t, "Node C", nodeC.Name)
		assert.Equal(t, "value", nodeC.Configuration.AsMap()["baz"])

		edge := response.Version.Spec.Edges[0]
		assert.Equal(t, "node-a", edge.SourceId)
		assert.Equal(t, "node-c", edge.TargetId)
		assert.Equal(t, "default", edge.Channel)

		version, findErr := models.FindCanvasVersion(canvas.ID, draftVersion.ID)
		require.NoError(t, findErr)
		require.NotNil(t, version.OwnerID)
		assert.Equal(t, otherUser.ID, *version.OwnerID)
		require.Len(t, version.Nodes, 2)
		require.Len(t, version.Edges, 1)
	})

	t.Run("returns invalid argument when operations produce invalid graph", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				testCanvasNode("node-a", "Node A", map[string]any{}),
				testCanvasNode("node-b", "Node B", map[string]any{}),
			},
			[]models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}},
		)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		draftVersion := createCanvasDraftVersionFromLive(t, canvas.ID, *canvas.LiveVersionID, r.User)

		_, err := ApplyCanvasVersionChangeset(
			ctx,
			r.Registry,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID,
			canvas.ID,
			draftVersion.ID,
			&pb.CanvasChangeset{
				Changes: []*pb.CanvasChangeset_Change{
					{
						Type: pb.CanvasChangeset_Change_ADD_EDGE,
						Edge: &pb.CanvasChangeset_Change_Edge{
							SourceId: "node-b",
							TargetId: "node-a",
							Channel:  "default",
						},
					},
				},
			},
			nil,
		)

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "graph contains a cycle")

		version, findErr := models.FindCanvasVersion(canvas.ID, draftVersion.ID)
		require.NoError(t, findErr)
		require.Len(t, version.Edges, 1)
		assert.Equal(t, "node-a", version.Edges[0].SourceID)
		assert.Equal(t, "node-b", version.Edges[0].TargetID)
	})

	t.Run("applies auto layout when requested", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				testCanvasNode("node-a", "Node A", map[string]any{"foo": "value"}),
				testCanvasNode("node-b", "Node B", map[string]any{"bar": "value"}),
			},
			nil,
		)

		draftVersion := createCanvasDraftVersionFromLive(t, canvas.ID, *canvas.LiveVersionID, r.User)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		response, err := ApplyCanvasVersionChangeset(
			ctx,
			r.Registry,
			r.AuthService,
			testWebhookBaseURL,
			r.Organization.ID,
			canvas.ID,
			draftVersion.ID,
			&pb.CanvasChangeset{
				Changes: []*pb.CanvasChangeset_Change{
					{
						Type: pb.CanvasChangeset_Change_ADD_EDGE,
						Edge: &pb.CanvasChangeset_Change_Edge{
							SourceId: "node-a",
							TargetId: "node-b",
							Channel:  "default",
						},
					},
				},
			},
			&pb.CanvasAutoLayout{
				Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
				Scope:     pb.CanvasAutoLayout_SCOPE_FULL_CANVAS,
			},
		)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Version)
		require.NotNil(t, response.Version.Spec)

		nodeA := findProtoNode(response.Version.Spec.Nodes, "node-a")
		nodeB := findProtoNode(response.Version.Spec.Nodes, "node-b")
		require.NotNil(t, nodeA)
		require.NotNil(t, nodeB)
		require.NotNil(t, nodeA.Position)
		require.NotNil(t, nodeB.Position)
		assert.Greater(t, nodeB.Position.X, nodeA.Position.X)

		version, findErr := models.FindCanvasVersion(canvas.ID, draftVersion.ID)
		require.NoError(t, findErr)

		storedNodeA := findModelNode(version.Nodes, "node-a")
		storedNodeB := findModelNode(version.Nodes, "node-b")
		require.NotNil(t, storedNodeA)
		require.NotNil(t, storedNodeB)
		assert.Greater(t, storedNodeB.Position.X, storedNodeA.Position.X)
	})
}

func testCanvasNode(id string, name string, configuration map[string]any) models.CanvasNode {
	return models.CanvasNode{
		NodeID:        id,
		Name:          name,
		Type:          models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
		Configuration: datatypes.NewJSONType(configuration),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(models.Position{X: 0, Y: 0}),
		State:         models.CanvasNodeStateReady,
	}
}

func structFromAnyMap(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()

	result, err := structpb.NewStruct(value)
	require.NoError(t, err)

	return result
}

func findProtoNode(nodes []*componentpb.Node, nodeID string) *componentpb.Node {
	for _, node := range nodes {
		if node.GetId() == nodeID {
			return node
		}
	}

	return nil
}

func findModelNode(nodes []models.Node, nodeID string) *models.Node {
	for i := range nodes {
		if nodes[i].ID == nodeID {
			return &nodes[i]
		}
	}

	return nil
}

func createCanvasDraftVersionFromLive(t *testing.T, canvasID uuid.UUID, liveVersionID uuid.UUID, userID uuid.UUID) *models.CanvasVersion {
	t.Helper()

	liveVersion, err := models.FindCanvasVersion(canvasID, liveVersionID)
	require.NoError(t, err)

	draftVersion, err := models.SaveCanvasDraftInTransaction(
		database.Conn(),
		canvasID,
		userID,
		liveVersion.Nodes,
		liveVersion.Edges,
	)
	require.NoError(t, err)

	return draftVersion
}
