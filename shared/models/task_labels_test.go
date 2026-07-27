package models

import (
	"strings"
	"testing"
)

func TestNormalizeOriginLabels(t *testing.T) {
	got := NormalizeOriginLabels(map[string]string{
		LabelCanvasName: "  release-train  ",
		LabelNodeName:   "Run tests",
		"ignored":       "x",
	})
	if got[LabelCanvasName] != "release-train" || got[LabelNodeName] != "Run tests" || len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	if NormalizeOriginLabels(map[string]string{"other": "x"}) != nil {
		t.Fatal("expected nil for unknown keys only")
	}
	long := strings.Repeat("a", maxLabelValueLen+10)
	got = NormalizeOriginLabels(map[string]string{LabelCanvasName: long})
	if len(got[LabelCanvasName]) != maxLabelValueLen {
		t.Fatalf("len=%d", len(got[LabelCanvasName]))
	}
}
