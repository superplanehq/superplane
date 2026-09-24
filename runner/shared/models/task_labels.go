package models

import "strings"

// Well-known task label keys for origin attribution (alerts / Dash0).
const (
	LabelCanvasID       = "canvas_id"
	LabelOrganizationID = "organization_id"
	LabelCanvasName     = "canvas_name"
	LabelNodeName       = "node_name"
)

const maxLabelValueLen = 128

var originLabelKeys = []string{
	LabelCanvasID,
	LabelOrganizationID,
	LabelCanvasName,
	LabelNodeName,
}

// NormalizeOriginLabels keeps only known origin keys, trimmed and length-capped.
func NormalizeOriginLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(originLabelKeys))
	for _, key := range originLabelKeys {
		if v := normalizeLabelValue(in[key]); v != "" {
			out[key] = v
		}
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
