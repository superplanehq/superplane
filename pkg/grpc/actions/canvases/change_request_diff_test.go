package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestComputeCanvasChangeRequestDiffDetectsConflictingNodeIDs(t *testing.T) {
	baseNodes := []models.Node{
		{ID: "node-a", Name: "A", Type: models.NodeTypeComponent},
		{ID: "node-b", Name: "B", Type: models.NodeTypeComponent},
	}
	baseEdges := []models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}}

	liveNodes := []models.Node{
		{ID: "node-a", Name: "A", Type: models.NodeTypeComponent},
		{ID: "node-b", Name: "B from live", Type: models.NodeTypeComponent},
	}
	liveEdges := []models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}}

	versionNodes := []models.Node{
		{ID: "node-a", Name: "A from version", Type: models.NodeTypeComponent},
		{ID: "node-b", Name: "B from version", Type: models.NodeTypeComponent},
	}
	versionEdges := []models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}}

	diff := computeCanvasChangeRequestDiff(baseNodes, baseEdges, liveNodes, liveEdges, versionNodes, versionEdges)

	assert.ElementsMatch(t, []string{"node-a", "node-b"}, diff.ChangedNodeIDs)
	assert.ElementsMatch(t, []string{"node-b"}, diff.ConflictingNodeIDs)
}
