package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestComputeCanvasChangeRequestDiffDetectsConcurrentNodeEdits(t *testing.T) {
	baseNodes := []models.Node{{
		ID:   "node-1",
		Name: "Initial Name",
		Type: models.NodeTypeComponent,
		Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
	}}
	liveNodes := []models.Node{{
		ID:   "node-1",
		Name: "Draft Two",
		Type: models.NodeTypeComponent,
		Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
	}}
	versionNodes := []models.Node{{
		ID:   "node-1",
		Name: "Draft One",
		Type: models.NodeTypeComponent,
		Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
	}}

	diff := computeCanvasChangeRequestDiff(baseNodes, nil, liveNodes, nil, versionNodes, nil)
	assert.Contains(t, diff.ConflictingNodeIDs, "node-1")
}
