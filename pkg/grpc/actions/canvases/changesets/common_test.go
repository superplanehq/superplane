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

func TestCheckForCycles_NamesTheNodesInTheCycle(t *testing.T) {
	nodes := []models.Node{
		{ID: "node-a", Name: "Deploy to Droplet", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
		{ID: "node-b", Name: "Deploy Failed", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
	}
	edges := []models.Edge{
		{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
		{SourceID: "node-b", TargetID: "node-a", Channel: "default"},
	}

	err := changesets.CheckForCycles(nodes, edges)

	require.Error(t, err)
	require.Equal(t, "graph contains a cycle: Deploy to Droplet -> Deploy Failed -> Deploy to Droplet", err.Error())
}

func TestCheckForCycles_NamesLongerCycles(t *testing.T) {
	nodes := []models.Node{
		{ID: "a", Name: "Build", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
		{ID: "b", Name: "Test", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
		{ID: "c", Name: "Deploy", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
	}
	edges := []models.Edge{
		{SourceID: "a", TargetID: "b", Channel: "default"},
		{SourceID: "b", TargetID: "c", Channel: "default"},
		{SourceID: "c", TargetID: "a", Channel: "default"},
	}

	err := changesets.CheckForCycles(nodes, edges)

	require.Error(t, err)
	require.Equal(t, "graph contains a cycle: Build -> Test -> Deploy -> Build", err.Error())
}

func TestCheckForCycles_ReportsUnnamedNodesByID(t *testing.T) {
	nodes := []models.Node{
		{ID: "node-a", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
		{ID: "node-b", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
	}
	edges := []models.Edge{
		{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
		{SourceID: "node-b", TargetID: "node-a", Channel: "default"},
	}

	err := changesets.CheckForCycles(nodes, edges)

	require.Error(t, err)
	require.Equal(t, "graph contains a cycle: node-a -> node-b -> node-a", err.Error())
}

func TestCheckForCycles_ReportsOnlyTheCyclicNodes(t *testing.T) {
	nodes := []models.Node{
		{ID: "trigger", Name: "Start", Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}},
		{ID: "a", Name: "Build", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
		{ID: "b", Name: "Test", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}},
	}
	edges := []models.Edge{
		{SourceID: "trigger", TargetID: "a", Channel: "default"},
		{SourceID: "a", TargetID: "b", Channel: "default"},
		{SourceID: "b", TargetID: "a", Channel: "default"},
	}

	err := changesets.CheckForCycles(nodes, edges)

	require.Error(t, err)
	require.Equal(t, "graph contains a cycle: Build -> Test -> Build", err.Error())
}
