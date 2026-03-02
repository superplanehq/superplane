package canvases

import (
	"reflect"
	"testing"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestBuildDefaultAutoLayoutUsesConnectedComponentForChangedFlowNodes(t *testing.T) {
	current := testCanvas(
		[]openapi_client.ComponentsNode{
			testNode("trigger", openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER, map[string]any{"repository": "a/repo"}),
			testNode("component", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, map[string]any{"ref": "main"}),
			testNode("widget", openapi_client.COMPONENTSNODETYPE_TYPE_WIDGET, nil),
		},
		[]openapi_client.ComponentsEdge{
			testEdge("trigger", "component", "default"),
		},
	)
	next := testCanvas(
		[]openapi_client.ComponentsNode{
			testNode("trigger", openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER, map[string]any{"repository": "a/repo"}),
			testNode("component", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, map[string]any{"ref": "release"}),
			testNode("widget", openapi_client.COMPONENTSNODETYPE_TYPE_WIDGET, nil),
		},
		[]openapi_client.ComponentsEdge{
			testEdge("trigger", "component", "default"),
		},
	)

	autoLayout := buildDefaultAutoLayout(current, next)

	if autoLayout.GetAlgorithm() != openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL {
		t.Fatalf("expected horizontal auto-layout, got %s", autoLayout.GetAlgorithm())
	}
	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT {
		t.Fatalf("expected connected-component scope, got %s", autoLayout.GetScope())
	}
	if !reflect.DeepEqual(autoLayout.GetNodeIds(), []string{"component"}) {
		t.Fatalf("expected changed node ids [component], got %v", autoLayout.GetNodeIds())
	}
}

func TestBuildDefaultAutoLayoutUsesChangedEdgeEndpoints(t *testing.T) {
	current := testCanvas(
		[]openapi_client.ComponentsNode{
			testNode("trigger", openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER, nil),
			testNode("component-a", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, nil),
			testNode("component-b", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, nil),
		},
		[]openapi_client.ComponentsEdge{
			testEdge("trigger", "component-a", "default"),
		},
	)
	next := testCanvas(
		[]openapi_client.ComponentsNode{
			testNode("trigger", openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER, nil),
			testNode("component-b", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, nil),
			testNode("component-a", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, nil),
		},
		[]openapi_client.ComponentsEdge{
			testEdge("trigger", "component-a", "default"),
			testEdge("component-a", "component-b", "default"),
		},
	)

	autoLayout := buildDefaultAutoLayout(current, next)

	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT {
		t.Fatalf("expected connected-component scope, got %s", autoLayout.GetScope())
	}
	if !reflect.DeepEqual(autoLayout.GetNodeIds(), []string{"component-b", "component-a"}) {
		t.Fatalf("expected node ids ordered by updated canvas [component-b component-a], got %v", autoLayout.GetNodeIds())
	}
}

func TestBuildDefaultAutoLayoutUsesNeighborWhenNodeWasRemoved(t *testing.T) {
	current := testCanvas(
		[]openapi_client.ComponentsNode{
			testNode("trigger", openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER, nil),
			testNode("component-a", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, nil),
			testNode("component-b", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, nil),
		},
		[]openapi_client.ComponentsEdge{
			testEdge("trigger", "component-a", "default"),
			testEdge("component-a", "component-b", "default"),
		},
	)
	next := testCanvas(
		[]openapi_client.ComponentsNode{
			testNode("trigger", openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER, nil),
			testNode("component-a", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, nil),
		},
		[]openapi_client.ComponentsEdge{
			testEdge("trigger", "component-a", "default"),
		},
	)

	autoLayout := buildDefaultAutoLayout(current, next)

	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT {
		t.Fatalf("expected connected-component scope, got %s", autoLayout.GetScope())
	}
	if !reflect.DeepEqual(autoLayout.GetNodeIds(), []string{"component-a"}) {
		t.Fatalf("expected node ids [component-a], got %v", autoLayout.GetNodeIds())
	}
}

func TestBuildDefaultAutoLayoutIgnoresWarningAndErrorDifferences(t *testing.T) {
	currentNode := testNode("component", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, map[string]any{"name": "deploy"})
	currentError := "invalid config"
	currentWarning := "duplicate name"
	currentNode.ErrorMessage = &currentError
	currentNode.WarningMessage = &currentWarning

	nextNode := testNode("component", openapi_client.COMPONENTSNODETYPE_TYPE_COMPONENT, map[string]any{"name": "deploy"})

	current := testCanvas([]openapi_client.ComponentsNode{currentNode}, nil)
	next := testCanvas([]openapi_client.ComponentsNode{nextNode}, nil)

	autoLayout := buildDefaultAutoLayout(current, next)

	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS {
		t.Fatalf("expected full-canvas scope when only warning/error changes, got %s", autoLayout.GetScope())
	}
	if autoLayout.HasNodeIds() {
		t.Fatalf("expected no node ids when canvas content is unchanged, got %v", autoLayout.GetNodeIds())
	}
}

func testCanvas(nodes []openapi_client.ComponentsNode, edges []openapi_client.ComponentsEdge) openapi_client.CanvasesCanvas {
	spec := openapi_client.CanvasesCanvasSpec{}
	spec.SetNodes(nodes)
	spec.SetEdges(edges)

	canvas := openapi_client.CanvasesCanvas{}
	canvas.SetSpec(spec)
	return canvas
}

func testNode(id string, nodeType openapi_client.ComponentsNodeType, configuration map[string]any) openapi_client.ComponentsNode {
	node := openapi_client.ComponentsNode{}
	node.SetId(id)
	node.SetType(nodeType)
	if configuration != nil {
		node.SetConfiguration(configuration)
	}

	return node
}

func testEdge(sourceID string, targetID string, channel string) openapi_client.ComponentsEdge {
	edge := openapi_client.ComponentsEdge{}
	edge.SetSourceId(sourceID)
	edge.SetTargetId(targetID)
	edge.SetChannel(channel)
	return edge
}
