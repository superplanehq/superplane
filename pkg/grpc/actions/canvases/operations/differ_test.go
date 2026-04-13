package operations

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func TestDifferDiff_MixedOperations(t *testing.T) {
	differ := NewDiffer(
		[]models.Node{
			{
				ID:            "node-a",
				Name:          "Node A",
				Configuration: map[string]any{"foo": "before"},
			},
			{
				ID:            "node-b",
				Name:          "Node B",
				Configuration: map[string]any{"bar": "value"},
			},
		},
		[]models.Edge{
			{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
			{SourceID: "node-b", TargetID: "node-a", Channel: "secondary"},
		},
		[]models.Node{
			{
				ID:            "node-a",
				Name:          "Node A Updated",
				Configuration: map[string]any{"foo": "after"},
			},
			{
				ID:            "node-c",
				Name:          "Node C",
				Configuration: map[string]any{"baz": "value"},
			},
		},
		[]models.Edge{
			{SourceID: "node-a", TargetID: "node-c", Channel: "default"},
		},
	)

	operations, err := differ.Diff()
	require.NoError(t, err)
	require.Len(t, operations, 6)

	require.Equal(t, pb.CanvasUpdateOperation_DISCONNECT_NODES, operations[0].Type)
	require.Equal(t, "node-a", operations[0].Source.Id)
	require.Equal(t, "node-b", operations[0].Target.Id)
	require.Equal(t, "default", operations[0].Source.Channel)
	require.Equal(t, "default", operations[0].Target.Channel)

	require.Equal(t, pb.CanvasUpdateOperation_DISCONNECT_NODES, operations[1].Type)
	require.Equal(t, "node-b", operations[1].Source.Id)
	require.Equal(t, "node-a", operations[1].Target.Id)
	require.Equal(t, "secondary", operations[1].Source.Channel)
	require.Equal(t, "secondary", operations[1].Target.Channel)

	require.Equal(t, pb.CanvasUpdateOperation_DELETE_NODE, operations[2].Type)
	require.Equal(t, "node-b", operations[2].Target.Id)

	require.Equal(t, pb.CanvasUpdateOperation_ADD_NODE, operations[3].Type)
	require.Equal(t, "node-c", operations[3].Target.Id)
	require.Equal(t, "Node C", operations[3].Target.Name)
	require.Equal(t, "value", operations[3].Target.Configuration.AsMap()["baz"])

	require.Equal(t, pb.CanvasUpdateOperation_UPDATE_NODE, operations[4].Type)
	require.Equal(t, "node-a", operations[4].Target.Id)
	require.Equal(t, "Node A Updated", operations[4].Target.Name)
	require.Equal(t, "after", operations[4].Target.Configuration.AsMap()["foo"])

	require.Equal(t, pb.CanvasUpdateOperation_CONNECT_NODES, operations[5].Type)
	require.Equal(t, "node-a", operations[5].Source.Id)
	require.Equal(t, "node-c", operations[5].Target.Id)
	require.Equal(t, "default", operations[5].Source.Channel)
	require.Equal(t, "default", operations[5].Target.Channel)
}

func TestDifferDiff_NoChanges(t *testing.T) {
	nodes := []models.Node{
		{
			ID:            "node-a",
			Name:          "Node A",
			Configuration: map[string]any{"foo": "bar"},
		},
	}
	edges := []models.Edge{
		{SourceID: "node-a", TargetID: "node-a", Channel: "self"},
	}

	differ := NewDiffer(nodes, edges, nodes, edges)

	operations, err := differ.Diff()
	require.NoError(t, err)
	require.Empty(t, operations)
}

func TestDifferDiff_InvalidNodeConfiguration(t *testing.T) {
	differ := NewDiffer(
		nil,
		nil,
		[]models.Node{
			{
				ID:            "node-a",
				Name:          "Node A",
				Configuration: map[string]any{"invalid": func() {}},
			},
		},
		nil,
	)

	operations, err := differ.Diff()
	require.Error(t, err)
	require.Nil(t, operations)
}
