package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ValidateCanvasVersionChangeset(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns patched version without persisting patched version", func(t *testing.T) {
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
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		draftVersion := createCanvasDraftVersionFromLive(t, canvas.ID, *canvas.LiveVersionID, r.User)

		versionBefore, err := models.FindCanvasVersion(canvas.ID, draftVersion.ID)
		require.NoError(t, err)
		require.Len(t, versionBefore.Nodes, 2)
		require.Len(t, versionBefore.Edges, 1)

		response, err := ValidateCanvasVersionChangeset(
			ctx,
			r.Registry,
			r.AuthService,
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
						Type: pb.CanvasChangeset_Change_DELETE_NODE,
						Node: &pb.CanvasChangeset_Change_Node{Id: "node-b"},
					},
					{
						Type: pb.CanvasChangeset_Change_ADD_EDGE,
						Edge: &pb.CanvasChangeset_Change_Edge{
							SourceId: "node-a",
							TargetId: "node-c",
							Channel:  "default",
						},
					},
				},
			},
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

		versionAfter, err := models.FindCanvasVersion(canvas.ID, draftVersion.ID)
		require.NoError(t, err)
		require.Len(t, versionAfter.Nodes, 2)
		require.Len(t, versionAfter.Edges, 1)

		var nodeAInDB *models.Node
		var nodeBInDB *models.Node
		for i := range versionAfter.Nodes {
			switch versionAfter.Nodes[i].ID {
			case "node-a":
				nodeAInDB = &versionAfter.Nodes[i]
			case "node-b":
				nodeBInDB = &versionAfter.Nodes[i]
			}
		}

		require.NotNil(t, nodeAInDB)
		assert.Equal(t, "Node A", nodeAInDB.Name)
		assert.Equal(t, "before", nodeAInDB.Configuration["foo"])

		require.NotNil(t, nodeBInDB)
		assert.Equal(t, "Node B", nodeBInDB.Name)
		assert.Equal(t, "value", nodeBInDB.Configuration["bar"])

		edgeInDB := versionAfter.Edges[0]
		assert.Equal(t, "node-a", edgeInDB.SourceID)
		assert.Equal(t, "node-b", edgeInDB.TargetID)
		assert.Equal(t, "default", edgeInDB.Channel)

		assert.Equal(t, versionBefore.UpdatedAt, versionAfter.UpdatedAt)
		assert.Equal(t, versionBefore.OwnerID, versionAfter.OwnerID)
	})
}
