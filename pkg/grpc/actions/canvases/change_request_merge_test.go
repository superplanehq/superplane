package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestMergeCanvasVersionIntoLivePreservesUnchangedLiveChanges(t *testing.T) {
	baseNodes := []models.Node{
		{ID: "node-a", Name: "A", Type: models.NodeTypeComponent},
		{ID: "node-b", Name: "B", Type: models.NodeTypeComponent},
	}
	baseEdges := []models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "default"}}

	liveNodes := []models.Node{
		{ID: "node-a", Name: "A", Type: models.NodeTypeComponent},
		{ID: "node-b", Name: "B from live", Type: models.NodeTypeComponent},
		{ID: "node-c", Name: "C from live", Type: models.NodeTypeComponent},
	}
	liveEdges := []models.Edge{
		{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
		{SourceID: "node-b", TargetID: "node-c", Channel: "default"},
	}

	versionNodes := []models.Node{
		{ID: "node-a", Name: "A from version", Type: models.NodeTypeComponent},
		{ID: "node-b", Name: "B", Type: models.NodeTypeComponent},
	}
	versionEdges := []models.Edge{
		{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
		{SourceID: "node-a", TargetID: "node-c", Channel: "default"},
	}

	changed := []string{"node-a"}
	mergedNodes, mergedEdges := mergeCanvasVersionIntoLive(
		baseNodes,
		baseEdges,
		liveNodes,
		liveEdges,
		versionNodes,
		versionEdges,
		changed,
	)

	assert.Len(t, mergedNodes, 3)
	assert.Equal(t, "A from version", findNodeName(mergedNodes, "node-a"))
	assert.Equal(t, "B from live", findNodeName(mergedNodes, "node-b"))
	assert.Equal(t, "C from live", findNodeName(mergedNodes, "node-c"))

	assert.ElementsMatch(t, []models.Edge{
		{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
		{SourceID: "node-a", TargetID: "node-c", Channel: "default"},
		{SourceID: "node-b", TargetID: "node-c", Channel: "default"},
	}, mergedEdges)
}

func TestMergeCanvasVersionIntoLiveDeletesRemovedNodes(t *testing.T) {
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

	versionNodes := []models.Node{{ID: "node-b", Name: "B", Type: models.NodeTypeComponent}}
	versionEdges := []models.Edge{}

	changed := []string{"node-a"}
	mergedNodes, mergedEdges := mergeCanvasVersionIntoLive(
		baseNodes,
		baseEdges,
		liveNodes,
		liveEdges,
		versionNodes,
		versionEdges,
		changed,
	)

	assert.Len(t, mergedNodes, 1)
	assert.Equal(t, "node-b", mergedNodes[0].ID)
	assert.Empty(t, mergedEdges)
}

func findNodeName(nodes []models.Node, nodeID string) string {
	for _, node := range nodes {
		if node.ID == nodeID {
			return node.Name
		}
	}

	return ""
}
