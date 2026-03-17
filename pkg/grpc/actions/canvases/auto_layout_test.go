package canvases

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func TestApplyCanvasAutoLayoutStacksDisconnectedComponentsVertically(t *testing.T) {
	nodes := []models.Node{
		{ID: "component-a-1", Type: models.NodeTypeComponent, Position: models.Position{X: 0, Y: 0}},
		{ID: "component-a-2", Type: models.NodeTypeComponent, Position: models.Position{X: 300, Y: 0}},
		{ID: "component-b-1", Type: models.NodeTypeComponent, Position: models.Position{X: 0, Y: 500}},
		{ID: "component-b-2", Type: models.NodeTypeComponent, Position: models.Position{X: 300, Y: 500}},
	}
	edges := []models.Edge{
		{SourceID: "component-a-1", TargetID: "component-a-2", Channel: "default"},
		{SourceID: "component-b-1", TargetID: "component-b-2", Channel: "default"},
	}

	autoLayout := &pb.CanvasAutoLayout{
		Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
		Scope:     pb.CanvasAutoLayout_SCOPE_FULL_CANVAS,
	}

	updatedNodes, updatedEdges, err := applyCanvasAutoLayout(nodes, edges, autoLayout, nil)
	require.NoError(t, err)
	require.Equal(t, edges, updatedEdges)

	nodesByID := mapLayoutNodesByID(updatedNodes)
	a1 := nodesByID["component-a-1"]
	a2 := nodesByID["component-a-2"]
	b1 := nodesByID["component-b-1"]
	b2 := nodesByID["component-b-2"]

	componentAMaxY := maxInt(a1.Position.Y+180, a2.Position.Y+180)
	componentBMinY := minInt(b1.Position.Y, b2.Position.Y)
	require.Greater(t, componentBMinY, componentAMaxY)

	componentAMinX := minInt(a1.Position.X, a2.Position.X)
	componentBMinX := minInt(b1.Position.X, b2.Position.X)
	require.LessOrEqual(t, int(math.Abs(float64(componentAMinX-componentBMinX))), 1)
}

func TestApplyCanvasAutoLayoutPacksIsolatedNodesBelowConnectedComponent(t *testing.T) {
	nodes := []models.Node{
		{ID: "component-a-1", Type: models.NodeTypeComponent, Position: models.Position{X: 0, Y: 0}},
		{ID: "component-a-2", Type: models.NodeTypeComponent, Position: models.Position{X: 300, Y: 0}},
		{ID: "isolated", Type: models.NodeTypeComponent, Position: models.Position{X: 0, Y: 500}},
	}
	edges := []models.Edge{
		{SourceID: "component-a-1", TargetID: "component-a-2", Channel: "default"},
	}

	autoLayout := &pb.CanvasAutoLayout{
		Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
		Scope:     pb.CanvasAutoLayout_SCOPE_FULL_CANVAS,
	}

	updatedNodes, _, err := applyCanvasAutoLayout(nodes, edges, autoLayout, nil)
	require.NoError(t, err)

	nodesByID := mapLayoutNodesByID(updatedNodes)
	a1 := nodesByID["component-a-1"]
	a2 := nodesByID["component-a-2"]
	isolated := nodesByID["isolated"]

	componentAMaxY := maxInt(a1.Position.Y+180, a2.Position.Y+180)
	require.Greater(t, isolated.Position.Y, componentAMaxY)

	componentAMinX := minInt(a1.Position.X, a2.Position.X)
	require.LessOrEqual(t, int(math.Abs(float64(componentAMinX-isolated.Position.X))), 1)
}

func TestApplyCanvasAutoLayoutDoesNotPushTerminalNodeBackWithParallelEdges(t *testing.T) {
	nodes := []models.Node{
		{ID: "github-onpullrequest-on-pr-closed-cleanup-irrew6", Type: models.NodeTypeTrigger, Position: models.Position{X: -48, Y: -448}},
		{ID: "readmemory-read-machine-by-pr--cleanup--o50tk6", Type: models.NodeTypeComponent, Position: models.Position{X: 552, Y: -448}},
		{ID: "hetzner-deleteserver-terminate-hetzner-server--cleanup--bsc35t", Type: models.NodeTypeComponent, Position: models.Position{X: 1152, Y: -448}},
		{ID: "component-node-uu1k5g", Type: models.NodeTypeComponent, Position: models.Position{X: 1752, Y: -588}},
		{ID: "deletememory-delete-machine-mapping--cleanup--xo61py", Type: models.NodeTypeComponent, Position: models.Position{X: 2352, Y: -448}},
		{ID: "trigger-creation-of-new-machine-trigger-creation-of-new-machine-2-9w7rn9", Type: models.NodeTypeComponent, Position: models.Position{X: 1752, Y: -308}},
	}

	edges := []models.Edge{
		{SourceID: "github-onpullrequest-on-pr-closed-cleanup-irrew6", TargetID: "readmemory-read-machine-by-pr--cleanup--o50tk6", Channel: "default"},
		{SourceID: "readmemory-read-machine-by-pr--cleanup--o50tk6", TargetID: "hetzner-deleteserver-terminate-hetzner-server--cleanup--bsc35t", Channel: "found"},
		{SourceID: "hetzner-deleteserver-terminate-hetzner-server--cleanup--bsc35t", TargetID: "component-node-uu1k5g", Channel: "default"},
		{SourceID: "component-node-uu1k5g", TargetID: "deletememory-delete-machine-mapping--cleanup--xo61py", Channel: "default"},
		{SourceID: "deletememory-delete-machine-mapping--cleanup--xo61py", TargetID: "trigger-creation-of-new-machine-trigger-creation-of-new-machine-2-9w7rn9", Channel: "deleted"},
		{SourceID: "deletememory-delete-machine-mapping--cleanup--xo61py", TargetID: "trigger-creation-of-new-machine-trigger-creation-of-new-machine-2-9w7rn9", Channel: "notFound"},
	}

	autoLayout := &pb.CanvasAutoLayout{
		Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
		Scope:     pb.CanvasAutoLayout_SCOPE_FULL_CANVAS,
	}

	updatedNodes, _, err := applyCanvasAutoLayout(nodes, edges, autoLayout, nil)
	require.NoError(t, err)

	nodesByID := mapLayoutNodesByID(updatedNodes)
	source := nodesByID["deletememory-delete-machine-mapping--cleanup--xo61py"]
	target := nodesByID["trigger-creation-of-new-machine-trigger-creation-of-new-machine-2-9w7rn9"]

	require.Greater(t, target.Position.X, source.Position.X)
}

func mapLayoutNodesByID(nodes []models.Node) map[string]models.Node {
	result := make(map[string]models.Node, len(nodes))
	for _, node := range nodes {
		result[node.ID] = node
	}
	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
