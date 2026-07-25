package models

import "strings"

// Well-known task label keys for origin attribution (Dash0 alerts).
const (
	LabelCanvasName = "canvas_name"
	LabelNodeName   = "node_name"
)

const maxLabelValueLen = 128

// NormalizeOriginLabels keeps only canvas_name / node_name, trimmed and length-capped.
func NormalizeOriginLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, 2)
	if v := normalizeLabelValue(in[LabelCanvasName]); v != "" {
		out[LabelCanvasName] = v
	}
	if v := normalizeLabelValue(in[LabelNodeName]); v != "" {
		out[LabelNodeName] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeLabelValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if len(v) > maxLabelValueLen {
		return v[:maxLabelValueLen]
	}
	return v
}
