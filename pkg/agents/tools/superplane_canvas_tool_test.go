package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func TestResolveCustomToolAutoLayout_DefaultsGraphUpdatesToFullCanvas(t *testing.T) {
	layout := resolveCustomToolAutoLayout(nil, true)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_FULL_CANVAS, layout.Scope)
	assert.Empty(t, layout.NodeIds)
}

func TestResolveCustomToolAutoLayout_SkipsConsoleOnlyUpdates(t *testing.T) {
	assert.Nil(t, resolveCustomToolAutoLayout(nil, false))
}

func TestResolveCustomToolAutoLayout_PreservesExplicitSettings(t *testing.T) {
	layout := resolveCustomToolAutoLayout(&superPlaneCanvasAutoLayoutInput{
		Scope:   "connected_component",
		NodeIDs: []string{"node-1"},
	}, true)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT, layout.Scope)
	assert.Equal(t, []string{"node-1"}, layout.NodeIds)
}

func TestSummarizeNodes_UsesYamlComponentFieldName(t *testing.T) {
	summary := summarizeNodes([]models.Node{
		{
			ID:   "node-1",
			Name: "Notify",
			Type: "TYPE_ACTION",
			Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "slack.sendTextMessage"}},
		},
	}, 20)

	require.Len(t, summary, 1)
	assert.Equal(t, "slack.sendTextMessage", summary[0].Component)

	data, err := json.Marshal(summary[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), `"component":"slack.sendTextMessage"`)
	assert.NotContains(t, string(data), `"ref"`)
}
