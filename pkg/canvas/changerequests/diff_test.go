package changerequests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestComputeCanvasChangeRequestDiff_NoConflictForIdenticalNodeAddition(t *testing.T) {
	node := models.Node{
		ID:            "node-a",
		Name:          "Node A",
		Type:          models.NodeTypeComponent,
		Configuration: map[string]any{"foo": "bar"},
		Position:      models.Position{X: 10, Y: 20},
	}

	changed, conflicting := ComputeCanvasChangeRequestDiff(
		nil,
		nil,
		[]models.Node{node},
		nil,
		[]models.Node{node},
		nil,
	)

	assert.Equal(t, []string{"node-a"}, changed)
	assert.Empty(t, conflicting)
}

func TestComputeCanvasChangeRequestDiff_NoConflictForIdenticalNodeUpdate(t *testing.T) {
	baseNode := models.Node{
		ID:            "node-a",
		Name:          "Node A",
		Type:          models.NodeTypeComponent,
		Configuration: map[string]any{"foo": "old"},
	}
	updatedNode := models.Node{
		ID:            "node-a",
		Name:          "Node A Updated",
		Type:          models.NodeTypeComponent,
		Configuration: map[string]any{"foo": "new"},
	}

	changed, conflicting := ComputeCanvasChangeRequestDiff(
		[]models.Node{baseNode},
		nil,
		[]models.Node{updatedNode},
		nil,
		[]models.Node{updatedNode},
		nil,
	)

	assert.Equal(t, []string{"node-a"}, changed)
	assert.Empty(t, conflicting)
}

func TestComputeCanvasChangeRequestDiff_NoConflictForIdenticalEdgeAddition(t *testing.T) {
	nodeA := models.Node{ID: "node-a", Name: "Node A", Type: models.NodeTypeComponent}
	nodeB := models.Node{ID: "node-b", Name: "Node B", Type: models.NodeTypeComponent}
	edge := models.Edge{SourceID: "node-a", TargetID: "node-b", Channel: "default"}

	changed, conflicting := ComputeCanvasChangeRequestDiff(
		[]models.Node{nodeA, nodeB},
		nil,
		[]models.Node{nodeA, nodeB},
		[]models.Edge{edge},
		[]models.Node{nodeA, nodeB},
		[]models.Edge{edge},
	)

	assert.ElementsMatch(t, []string{"node-a", "node-b"}, changed)
	assert.Empty(t, conflicting)
}

func TestComputeCanvasChangeRequestDiff_ConflictForDifferentNodeUpdate(t *testing.T) {
	baseNode := models.Node{
		ID:            "node-a",
		Name:          "Node A",
		Type:          models.NodeTypeComponent,
		Configuration: map[string]any{"foo": "base"},
	}
	liveNode := models.Node{
		ID:            "node-a",
		Name:          "Node A Live",
		Type:          models.NodeTypeComponent,
		Configuration: map[string]any{"foo": "live"},
	}
	versionNode := models.Node{
		ID:            "node-a",
		Name:          "Node A Version",
		Type:          models.NodeTypeComponent,
		Configuration: map[string]any{"foo": "version"},
	}

	changed, conflicting := ComputeCanvasChangeRequestDiff(
		[]models.Node{baseNode},
		nil,
		[]models.Node{liveNode},
		nil,
		[]models.Node{versionNode},
		nil,
	)

	assert.Equal(t, []string{"node-a"}, changed)
	assert.Equal(t, []string{"node-a"}, conflicting)
}
