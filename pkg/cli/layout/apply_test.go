package layout

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestApplyToCanvasSpecRepositionsConnectedNodes(t *testing.T) {
	nodeA := openapi_client.NewSuperplaneComponentsNode()
	nodeA.SetId("a")
	nodeA.SetName("A")
	nodeA.SetComponent("noop")
	nodeA.SetType(openapi_client.COMPONENTSNODETYPE_TYPE_ACTION)
	posA := openapi_client.NewComponentsPosition()
	posA.SetX(100)
	posA.SetY(200)
	nodeA.SetPosition(*posA)

	nodeB := openapi_client.NewSuperplaneComponentsNode()
	nodeB.SetId("b")
	nodeB.SetName("B")
	nodeB.SetComponent("noop")
	nodeB.SetType(openapi_client.COMPONENTSNODETYPE_TYPE_ACTION)
	posB := openapi_client.NewComponentsPosition()
	posB.SetX(100)
	posB.SetY(200)
	nodeB.SetPosition(*posB)

	edge := openapi_client.NewComponentsEdge()
	edge.SetSourceId("a")
	edge.SetTargetId("b")
	edge.SetChannel("default")

	spec := openapi_client.NewCanvasesCanvasSpec()
	spec.SetNodes([]openapi_client.SuperplaneComponentsNode{*nodeA, *nodeB})
	spec.SetEdges([]openapi_client.ComponentsEdge{*edge})

	autoLayout := DefaultAutoLayout()
	require.NoError(t, ApplyToCanvasSpec(spec, &autoLayout))

	nodes := spec.GetNodes()
	require.Len(t, nodes, 2)
	pos0 := nodes[0].GetPosition()
	pos1 := nodes[1].GetPosition()
	require.NotEqual(t, pos0.GetX(), pos1.GetX())
}

func TestResolveUpdateAutoLayoutRejectsFlagsWithFileAutoLayout(t *testing.T) {
	fileAutoLayout := DefaultAutoLayout()
	_, err := ResolveUpdateAutoLayout(true, &fileAutoLayout, "horizontal", "", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "autoLayout")
}

func TestResolveUpdateAutoLayoutDefaultsToFullCanvasHorizontal(t *testing.T) {
	autoLayout, err := ResolveUpdateAutoLayout(false, nil, "", "", nil)
	require.NoError(t, err)
	require.NotNil(t, autoLayout)
	require.Equal(t, openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL, autoLayout.GetAlgorithm())
	require.Equal(t, openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS, autoLayout.GetScope())
	require.Empty(t, autoLayout.GetNodeIds())
}

func TestResolveUpdateAutoLayoutDisableViaFlags(t *testing.T) {
	autoLayout, err := ResolveUpdateAutoLayout(true, nil, "disable", "", nil)
	require.NoError(t, err)
	require.Nil(t, autoLayout)
}
