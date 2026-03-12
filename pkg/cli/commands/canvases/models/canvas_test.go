package models

import "testing"

func TestParseCanvasPreservesPositionYFromUnquotedKey(t *testing.T) {
	raw := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  id: 4e9ae08d-0363-40d2-ba2c-5f6389a418d8
  name: advanced-scala-issue-plan-discord
spec:
  nodes:
    - id: manual-plan-start
      name: manual_plan_start
      type: TYPE_TRIGGER
      trigger:
        name: start
      configuration:
        templates:
          - name: Incident Report
            payload:
              incidentId: INC-1001
      position:
        x: 120
        y: 500
      paused: false
      isCollapsed: false
  edges:
    - sourceId: manual-plan-start
      targetId: manual-plan-start
      channel: default
`)

	resource, err := ParseCanvas(raw)
	if err != nil {
		t.Fatalf("ParseCanvas returned error: %v", err)
	}

	nodes := resource.Spec.GetNodes()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	position := nodes[0].GetPosition()
	if position.GetX() != 120 {
		t.Fatalf("expected x=120, got %d", position.GetX())
	}
	if position.GetY() != 500 {
		t.Fatalf("expected y=500, got %d", position.GetY())
	}

	edges := resource.Spec.GetEdges()
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].GetSourceId() != "manual-plan-start" {
		t.Fatalf("expected sourceId=manual-plan-start, got %q", edges[0].GetSourceId())
	}
	if edges[0].GetTargetId() != "manual-plan-start" {
		t.Fatalf("expected targetId=manual-plan-start, got %q", edges[0].GetTargetId())
	}
}
