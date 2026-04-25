package canvases

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func nodeComponent(id, name, component string) openapi_client.SuperplaneComponentsNode {
	n := openapi_client.NewSuperplaneComponentsNode()
	n.SetId(id)
	n.SetName(name)
	comp := openapi_client.NewNodeComponentRef()
	comp.SetName(component)
	n.SetComponent(*comp)
	pos := openapi_client.NewComponentsPosition()
	pos.SetX(0)
	pos.SetY(0)
	n.SetPosition(*pos)
	return *n
}

func edge(source, target, channel string) openapi_client.SuperplaneComponentsEdge {
	e := openapi_client.NewSuperplaneComponentsEdge()
	e.SetSourceId(source)
	e.SetTargetId(target)
	e.SetChannel(channel)
	return *e
}

func TestBuildCanvasChangesetFromSpecsEmpty(t *testing.T) {
	cs, err := buildCanvasChangesetFromSpecs(nil, nil, nil, nil)
	require.NoError(t, err)
	require.Empty(t, cs.GetChanges())
}

func TestBuildCanvasChangesetFromSpecsAddNodeAndEdge(t *testing.T) {
	a := nodeComponent("a", "A", "noop")
	b := nodeComponent("b", "B", "noop")
	current := []openapi_client.SuperplaneComponentsNode{}
	currentEdges := []openapi_client.SuperplaneComponentsEdge{}
	proposed := []openapi_client.SuperplaneComponentsNode{a, b}
	proposedEdges := []openapi_client.SuperplaneComponentsEdge{
		edge("a", "b", "default"),
	}

	cs, err := buildCanvasChangesetFromSpecs(current, currentEdges, proposed, proposedEdges)
	require.NoError(t, err)
	changes := cs.GetChanges()
	require.GreaterOrEqual(t, len(changes), 3)

	var addCount, edgeCount int
	for _, ch := range changes {
		switch ch.GetType() {
		case openapi_client.CANVASCHANGESETCHANGETYPE_ADD_NODE:
			addCount++
		case openapi_client.CANVASCHANGESETCHANGETYPE_ADD_EDGE:
			edgeCount++
		}
	}
	require.Equal(t, 2, addCount)
	require.Equal(t, 1, edgeCount)
}

func TestCollectCanvasNodeIssues(t *testing.T) {
	n := nodeComponent("n1", "One", "noop")
	errMsg := "bad config"
	n.SetErrorMessage(errMsg)
	warnMsg := "heads up"
	n.SetWarningMessage(warnMsg)

	spec := openapi_client.NewCanvasesCanvasSpec()
	spec.SetNodes([]openapi_client.SuperplaneComponentsNode{n})

	issues := collectCanvasNodeIssues(*spec)
	require.Len(t, issues, 2)
	require.Equal(t, "error", issues[0].Kind)
	require.Equal(t, "warning", issues[1].Kind)
}
