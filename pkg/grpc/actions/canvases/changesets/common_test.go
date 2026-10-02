package changesets_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestCheckForCycles_AllowsFeedbackIntoLoop(t *testing.T) {
	nodes := []models.Node{
		{ID: "trigger", Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}},
		{ID: "loop", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "loop"}}},
		{ID: "worker", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
	}
	edges := []models.Edge{
		{SourceID: "trigger", TargetID: "loop", Channel: "default"},
		{SourceID: "loop", TargetID: "worker", Channel: "next"},
		{SourceID: "worker", TargetID: "loop", Channel: "default"},
	}

	require.NoError(t, changesets.CheckForCycles(nodes, edges))
}

func TestCheckForCycles_RejectsCyclesWithoutLoop(t *testing.T) {
	nodes := []models.Node{
		{ID: "node-a", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
		{ID: "node-b", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
	}
	edges := []models.Edge{
		{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
		{SourceID: "node-b", TargetID: "node-a", Channel: "default"},
	}

	require.Error(t, changesets.CheckForCycles(nodes, edges))
}

func regularNode(id string) models.Node {
	return models.Node{ID: id, Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}}
}

func loopNode(id string) models.Node {
	return models.Node{ID: id, Ref: models.NodeRef{Component: &models.ComponentRef{Name: "loop"}}}
}

func edge(source, target string) models.Edge {
	return models.Edge{SourceID: source, TargetID: target, Channel: "default"}
}

func TestCheckForCycles(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []models.Node
		edges     []models.Edge
		wantCycle bool
	}{
		{
			name:      "linear graph is acyclic",
			nodes:     []models.Node{regularNode("a"), regularNode("b"), regularNode("c")},
			edges:     []models.Edge{edge("a", "b"), edge("b", "c")},
			wantCycle: false,
		},
		{
			name:  "diamond is acyclic",
			nodes: []models.Node{regularNode("a"), regularNode("b"), regularNode("c"), regularNode("d")},
			edges: []models.Edge{
				edge("a", "b"), edge("a", "c"), edge("b", "d"), edge("c", "d"),
			},
			wantCycle: false,
		},
		{
			name:      "self loop on a regular component is a cycle",
			nodes:     []models.Node{regularNode("a")},
			edges:     []models.Edge{edge("a", "a")},
			wantCycle: true,
		},
		{
			name:  "feedback edge into a loop node is allowed",
			nodes: []models.Node{regularNode("trigger"), loopNode("loop"), regularNode("worker")},
			edges: []models.Edge{
				edge("trigger", "loop"), edge("loop", "worker"), edge("worker", "loop"),
			},
			wantCycle: false,
		},
		{
			name:  "feedback edge from deeper in the loop body is allowed",
			nodes: []models.Node{loopNode("loop"), regularNode("w1"), regularNode("w2")},
			edges: []models.Edge{
				edge("loop", "w1"), edge("w1", "w2"), edge("w2", "loop"),
			},
			wantCycle: false,
		},
		{
			// A loop and a single body node, with no separate trigger feeding the
			// loop. The exemption covers the feedback edge, so this is a loop rather
			// than a cycle.
			name:  "loop and its body alone are allowed",
			nodes: []models.Node{regularNode("a"), loopNode("loop")},
			edges: []models.Edge{
				edge("loop", "a"), edge("a", "loop"),
			},
			wantCycle: false,
		},
		{
			// The exemption is what makes a loop possible, so it holds however many
			// nodes the body spans.
			name:  "loop with a multi node body is allowed",
			nodes: []models.Node{regularNode("a"), loopNode("loop"), regularNode("b")},
			edges: []models.Edge{
				edge("loop", "b"), edge("b", "a"), edge("a", "loop"),
			},
			wantCycle: false,
		},
		{
			name:  "cycle between regular components downstream of a loop is a cycle",
			nodes: []models.Node{loopNode("loop"), regularNode("a"), regularNode("b")},
			edges: []models.Edge{
				edge("loop", "a"), edge("a", "b"), edge("b", "a"),
			},
			wantCycle: true,
		},
		{
			name:      "edge referencing an unknown node does not report a cycle",
			nodes:     []models.Node{regularNode("a")},
			edges:     []models.Edge{edge("a", "ghost")},
			wantCycle: false,
		},
		{
			// Counting visited nodes and comparing against len(nodes) breaks down
			// once an edge points at an id that is not in nodes: the unknown target
			// is visited too, which can pad the count back up to len(nodes) and hide
			// a real cycle elsewhere in the graph.
			name:  "unknown node in an edge does not mask a real cycle",
			nodes: []models.Node{regularNode("a"), regularNode("b"), regularNode("c")},
			edges: []models.Edge{
				edge("c", "ghost-1"), edge("c", "ghost-2"), edge("a", "b"), edge("b", "a"),
			},
			wantCycle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := changesets.CheckForCycles(tt.nodes, tt.edges)
			if tt.wantCycle {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
