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
