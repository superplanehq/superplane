package layout

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyLayout_NilLayoutIsNoOp(t *testing.T) {
	nodes := []N{{ID: "node-1", Position: Position{X: 10, Y: 20}}}
	edges := []E{{SourceID: "node-1", TargetID: "node-2"}}

	updatedNodes, updatedEdges, err := ApplyLayout(nodes, edges, nil)
	require.NoError(t, err)
	assert.Equal(t, nodes, updatedNodes)
	assert.Equal(t, edges, updatedEdges)
}

func TestApplyLayout_RejectsUnsupportedAlgorithm(t *testing.T) {
	nodes := []N{{ID: "node-1", Position: Position{X: 0, Y: 0}}}
	autoLayout := &AutoLayout{Algorithm: "vertical"}

	_, _, err := ApplyLayout(nodes, nil, autoLayout)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported layout algorithm")
}

func TestApplyLayout_AcceptsHorizontalAlias(t *testing.T) {
	nodes := []N{
		{ID: "source", Position: Position{X: 0, Y: 0}},
		{ID: "target", Position: Position{X: 500, Y: 0}},
	}
	edges := []E{{SourceID: "source", TargetID: "target", Channel: "default"}}
	autoLayout := &AutoLayout{Algorithm: "HORIZONTAL", Scope: ScopeFullCanvas}

	updatedNodes, _, err := ApplyLayout(nodes, edges, autoLayout)
	require.NoError(t, err)
	require.Len(t, updatedNodes, 2)
}

func TestApplyLayout_RejectsUnknownSeedNode(t *testing.T) {
	nodes := []N{{ID: "node-1", Position: Position{X: 0, Y: 0}}}
	autoLayout := &AutoLayout{
		Algorithm: AlgorithmHorizontal,
		NodeIDs:   []string{"missing-node"},
	}

	_, _, err := ApplyLayout(nodes, nil, autoLayout)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown node")
}

func TestApplyLayout_RejectsUnsupportedScope(t *testing.T) {
	nodes := []N{{ID: "node-1", Position: Position{X: 0, Y: 0}}}
	autoLayout := &AutoLayout{
		Algorithm: AlgorithmHorizontal,
		Scope:     "SCOPE_SELECTION",
	}

	_, _, err := ApplyLayout(nodes, nil, autoLayout)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported auto layout scope")
}

func TestApplyLayout_EmptyNodesIsNoOp(t *testing.T) {
	autoLayout := &AutoLayout{Algorithm: AlgorithmHorizontal, Scope: ScopeFullCanvas}

	updatedNodes, updatedEdges, err := ApplyLayout(nil, nil, autoLayout)
	require.NoError(t, err)
	assert.Nil(t, updatedNodes)
	assert.Nil(t, updatedEdges)
}

func TestApplyLayout_SkipsNodesWithoutIDs(t *testing.T) {
	nodes := []N{
		{ID: "", Position: Position{X: 0, Y: 0}},
		{ID: "node-1", Position: Position{X: 100, Y: 100}},
	}
	autoLayout := &AutoLayout{Algorithm: AlgorithmHorizontal, Scope: ScopeFullCanvas}

	updatedNodes, _, err := ApplyLayout(nodes, nil, autoLayout)
	require.NoError(t, err)
	assert.Equal(t, 100, updatedNodes[1].Position.X)
}

func TestApplyLayout_LayoutsDisconnectedNodesWithoutEdges(t *testing.T) {
	nodes := []N{
		{ID: "node-a", Position: Position{X: 0, Y: 0}},
		{ID: "node-b", Position: Position{X: 0, Y: 400}},
	}
	autoLayout := &AutoLayout{Algorithm: AlgorithmHorizontal, Scope: ScopeFullCanvas}

	updatedNodes, _, err := ApplyLayout(nodes, nil, autoLayout)
	require.NoError(t, err)

	nodesByID := mapLayoutNodesByID(updatedNodes)
	assert.NotEqual(t, 400, nodesByID["node-b"].Position.Y)
	assert.Less(t, nodesByID["node-a"].Position.Y, nodesByID["node-b"].Position.Y)
}

func TestApplyLayout_ConnectedComponentScopeLimitsLayout(t *testing.T) {
	nodes := []N{
		{ID: "component-a-1", Position: Position{X: 0, Y: 0}},
		{ID: "component-a-2", Position: Position{X: 300, Y: 0}},
		{ID: "component-b-1", Position: Position{X: 0, Y: 900}},
	}
	edges := []E{
		{SourceID: "component-a-1", TargetID: "component-a-2", Channel: "default"},
	}
	autoLayout := &AutoLayout{
		Algorithm: AlgorithmHorizontal,
		Scope:     ScopeConnectedComponent,
		NodeIDs:   []string{"component-a-1"},
	}

	updatedNodes, _, err := ApplyLayout(nodes, edges, autoLayout)
	require.NoError(t, err)

	nodesByID := mapLayoutNodesByID(updatedNodes)
	assert.Equal(t, Position{X: 0, Y: 900}, nodesByID["component-b-1"].Position)
	assert.NotEqual(t, Position{X: 300, Y: 0}, nodesByID["component-a-2"].Position)
}

func TestApplyLayout_ConnectedComponentScopeWithoutSeedsLayoutsAllNodes(t *testing.T) {
	nodes := []N{
		{ID: "component-a-1", Position: Position{X: 0, Y: 0}},
		{ID: "component-a-2", Position: Position{X: 300, Y: 0}},
		{ID: "component-b-1", Position: Position{X: 0, Y: 900}},
	}
	edges := []E{
		{SourceID: "component-a-1", TargetID: "component-a-2", Channel: "default"},
	}
	autoLayout := &AutoLayout{
		Algorithm: AlgorithmHorizontal,
		Scope:     ScopeConnectedComponent,
	}

	updatedNodes, _, err := ApplyLayout(nodes, edges, autoLayout)
	require.NoError(t, err)

	nodesByID := mapLayoutNodesByID(updatedNodes)
	assert.NotEqual(t, Position{X: 0, Y: 900}, nodesByID["component-b-1"].Position)
}

func TestApplyCanvasAutoLayoutStacksDisconnectedComponentsVertically(t *testing.T) {
	nodes := []N{
		{ID: "component-a-1", Type: "component", Position: Position{X: 0, Y: 0}},
		{ID: "component-a-2", Type: "component", Position: Position{X: 300, Y: 0}},
		{ID: "component-b-1", Type: "component", Position: Position{X: 0, Y: 500}},
		{ID: "component-b-2", Type: "component", Position: Position{X: 300, Y: 500}},
	}
	edges := []E{
		{SourceID: "component-a-1", TargetID: "component-a-2", Channel: "default"},
		{SourceID: "component-b-1", TargetID: "component-b-2", Channel: "default"},
	}

	autoLayout := &AutoLayout{
		Algorithm: AlgorithmHorizontal,
		Scope:     ScopeFullCanvas,
	}

	updatedNodes, updatedEdges, err := ApplyLayout(nodes, edges, autoLayout)
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
	nodes := []N{
		{ID: "component-a-1", Type: "component", Position: Position{X: 0, Y: 0}},
		{ID: "component-a-2", Type: "component", Position: Position{X: 300, Y: 0}},
		{ID: "isolated", Type: "component", Position: Position{X: 0, Y: 500}},
	}
	edges := []E{
		{SourceID: "component-a-1", TargetID: "component-a-2", Channel: "default"},
	}

	autoLayout := &AutoLayout{
		Algorithm: AlgorithmHorizontal,
		Scope:     ScopeFullCanvas,
	}

	updatedNodes, _, err := ApplyLayout(nodes, edges, autoLayout)
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
	nodes := []N{
		{ID: "github-onpullrequest-on-pr-closed-cleanup-irrew6", Type: "trigger", Position: Position{X: -48, Y: -448}},
		{ID: "readmemory-read-machine-by-pr--cleanup--o50tk6", Type: "component", Position: Position{X: 552, Y: -448}},
		{ID: "hetzner-deleteserver-terminate-hetzner-server--cleanup--bsc35t", Type: "component", Position: Position{X: 1152, Y: -448}},
		{ID: "component-node-uu1k5g", Type: "component", Position: Position{X: 1752, Y: -588}},
		{ID: "deletememory-delete-machine-mapping--cleanup--xo61py", Type: "component", Position: Position{X: 2352, Y: -448}},
		{ID: "trigger-creation-of-new-machine-trigger-creation-of-new-machine-2-9w7rn9", Type: "component", Position: Position{X: 1752, Y: -308}},
	}

	edges := []E{
		{SourceID: "github-onpullrequest-on-pr-closed-cleanup-irrew6", TargetID: "readmemory-read-machine-by-pr--cleanup--o50tk6", Channel: "default"},
		{SourceID: "readmemory-read-machine-by-pr--cleanup--o50tk6", TargetID: "hetzner-deleteserver-terminate-hetzner-server--cleanup--bsc35t", Channel: "found"},
		{SourceID: "hetzner-deleteserver-terminate-hetzner-server--cleanup--bsc35t", TargetID: "component-node-uu1k5g", Channel: "default"},
		{SourceID: "component-node-uu1k5g", TargetID: "deletememory-delete-machine-mapping--cleanup--xo61py", Channel: "default"},
		{SourceID: "deletememory-delete-machine-mapping--cleanup--xo61py", TargetID: "trigger-creation-of-new-machine-trigger-creation-of-new-machine-2-9w7rn9", Channel: "deleted"},
		{SourceID: "deletememory-delete-machine-mapping--cleanup--xo61py", TargetID: "trigger-creation-of-new-machine-trigger-creation-of-new-machine-2-9w7rn9", Channel: "notFound"},
	}

	autoLayout := &AutoLayout{
		Algorithm: AlgorithmHorizontal,
		Scope:     ScopeFullCanvas,
	}

	updatedNodes, _, err := ApplyLayout(nodes, edges, autoLayout)
	require.NoError(t, err)

	nodesByID := mapLayoutNodesByID(updatedNodes)
	source := nodesByID["deletememory-delete-machine-mapping--cleanup--xo61py"]
	target := nodesByID["trigger-creation-of-new-machine-trigger-creation-of-new-machine-2-9w7rn9"]

	require.Greater(t, target.Position.X, source.Position.X)
}

func TestApplyCanvasAutoLayoutPreservesForwardFlowForLoops(t *testing.T) {
	nodes := []N{
		{ID: "start", Type: "component", Position: Position{X: 0, Y: 0}},
		{ID: "process", Type: "component", Position: Position{X: 600, Y: 0}},
		{ID: "check", Type: "component", Position: Position{X: 1200, Y: 0}},
	}
	edges := []E{
		{SourceID: "start", TargetID: "process", Channel: "default"},
		{SourceID: "process", TargetID: "check", Channel: "default"},
		{SourceID: "check", TargetID: "start", Channel: "repeat"},
	}

	autoLayout := &AutoLayout{
		Algorithm: AlgorithmHorizontal,
		Scope:     ScopeFullCanvas,
	}

	updatedNodes, updatedEdges, err := ApplyLayout(nodes, edges, autoLayout)
	require.NoError(t, err)
	require.Equal(t, edges, updatedEdges)

	nodesByID := mapLayoutNodesByID(updatedNodes)
	require.Less(t, nodesByID["start"].Position.X, nodesByID["process"].Position.X)
	require.Less(t, nodesByID["process"].Position.X, nodesByID["check"].Position.X)
}

func mapLayoutNodesByID(nodes []N) map[string]N {
	result := make(map[string]N, len(nodes))
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
