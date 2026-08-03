package models

import (
	"strings"
	"testing"
)

func TestNormalizeOriginLabels(t *testing.T) {
	got := NormalizeOriginLabels(map[string]string{
		LabelCanvasID:       "  11111111-1111-1111-1111-111111111111  ",
		LabelOrganizationID: "22222222-2222-2222-2222-222222222222",
		LabelCanvasName:     "  release-train  ",
		LabelNodeName:       "Run tests",
		"ignored":           "x",
	})
	if len(got) != 4 {
		t.Fatalf("got %#v", got)
	}
	if got[LabelCanvasID] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("canvas_id=%q", got[LabelCanvasID])
	}
	if got[LabelOrganizationID] != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("organization_id=%q", got[LabelOrganizationID])
	}
	if got[LabelCanvasName] != "release-train" || got[LabelNodeName] != "Run tests" {
		t.Fatalf("names=%#v", got)
	}
	if NormalizeOriginLabels(map[string]string{"other": "x"}) != nil {
		t.Fatal("expected nil for unknown keys only")
	}
	long := strings.Repeat("a", maxLabelValueLen+10)
	got = NormalizeOriginLabels(map[string]string{LabelCanvasName: long})
	if len(got[LabelCanvasName]) != maxLabelValueLen {
		t.Fatalf("len=%d", len(got[LabelCanvasName]))
	}
	got = NormalizeOriginLabels(map[string]string{LabelCanvasID: "c-1", LabelOrganizationID: "o-1"})
	if len(got) != 2 || got[LabelCanvasID] != "c-1" || got[LabelOrganizationID] != "o-1" {
		t.Fatalf("ids-only %#v", got)
	}
}
