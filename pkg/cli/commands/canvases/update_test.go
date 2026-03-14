package canvases

import (
	"reflect"
	"testing"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestBuildDefaultAutoLayoutUsesFullCanvas(t *testing.T) {
	autoLayout := buildDefaultAutoLayout()

	if autoLayout.GetAlgorithm() != openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL {
		t.Fatalf("expected horizontal auto-layout, got %s", autoLayout.GetAlgorithm())
	}
	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS {
		t.Fatalf("expected full-canvas scope, got %s", autoLayout.GetScope())
	}
	if autoLayout.HasNodeIds() {
		t.Fatalf("expected no node ids for default full-canvas strategy, got %v", autoLayout.GetNodeIds())
	}
}

func TestParseAutoLayoutDefaultsAlgorithmToHorizontal(t *testing.T) {
	autoLayout, err := parseAutoLayout("", "connected-component", []string{"node-1", " node-2 ", "node-1"})
	if err != nil {
		t.Fatalf("parseAutoLayout returned error: %v", err)
	}
	if autoLayout == nil {
		t.Fatalf("expected autoLayout to be set")
	}
	if autoLayout.GetAlgorithm() != openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL {
		t.Fatalf("expected horizontal auto-layout, got %s", autoLayout.GetAlgorithm())
	}
	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT {
		t.Fatalf("expected connected-component scope, got %s", autoLayout.GetScope())
	}
	if !reflect.DeepEqual(autoLayout.GetNodeIds(), []string{"node-1", "node-2"}) {
		t.Fatalf("expected node ids [node-1 node-2], got %v", autoLayout.GetNodeIds())
	}
}

func TestParseAutoLayoutDisable(t *testing.T) {
	autoLayout, err := parseAutoLayout("disable", "", nil)
	if err != nil {
		t.Fatalf("parseAutoLayout returned error: %v", err)
	}
	if autoLayout != nil {
		t.Fatalf("expected nil autoLayout when disabled, got %#v", autoLayout)
	}
}

func TestParseAutoLayoutDisableRejectsScopeOrNodes(t *testing.T) {
	if _, err := parseAutoLayout("disable", "connected-component", nil); err == nil {
		t.Fatalf("expected error when scope is set together with disable")
	}
	if _, err := parseAutoLayout("disable", "", []string{"node-1"}); err == nil {
		t.Fatalf("expected error when node ids are set together with disable")
	}
}
