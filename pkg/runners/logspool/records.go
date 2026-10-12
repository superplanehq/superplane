package logspool

import (
	"encoding/json"
	"strings"
)

func encodeRecord(line []byte) []byte {
	message := strings.TrimSpace(string(line))
	if message == "" {
		return nil
	}
	record := lineRecord(message)
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil
	}
	return append(encoded, '\n')
}

func lineRecord(message string) any {
	if record, ok := controlRecord(message); ok {
		return record
	}
	return map[string]any{"type": "line", "text": message}
}

func controlRecord(message string) (map[string]any, bool) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(message), &envelope); err != nil {
		return nil, false
	}
	switch envelope.Type {
	case "cmd_start":
		return commandStartRecord(message)
	case "cmd_end":
		return commandEndRecord(message)
	case "tool_start":
		return toolStartRecord(message)
	case "tool_end":
		return toolEndRecord(message)
	default:
		return nil, false
	}
}

func commandStartRecord(message string) (map[string]any, bool) {
	var record struct {
		Index     int    `json:"index"`
		Text      string `json:"text"`
		Kind      string `json:"kind"`
		Preview   string `json:"preview"`
		StartedAt *int64 `json:"started_at"`
	}
	if err := json.Unmarshal([]byte(message), &record); err != nil || record.Index < 0 {
		return nil, false
	}
	output := map[string]any{
		"type":  "cmd_start",
		"index": record.Index,
		"text":  record.Text,
	}
	if kind := strings.TrimSpace(record.Kind); kind != "" {
		output["kind"] = kind
	}
	if preview := strings.TrimSpace(record.Preview); preview != "" {
		output["preview"] = preview
	}
	if record.StartedAt != nil && *record.StartedAt >= 0 {
		output["started_at"] = *record.StartedAt
	}
	return output, true
}

func commandEndRecord(message string) (map[string]any, bool) {
	var record struct {
		Index      int    `json:"index"`
		Status     string `json:"status"`
		DurationMS int64  `json:"duration_ms"`
	}
	if err := json.Unmarshal([]byte(message), &record); err != nil ||
		record.Index < 0 ||
		record.DurationMS < 0 ||
		!validStatus(record.Status) {
		return nil, false
	}
	return map[string]any{
		"type":        "cmd_end",
		"index":       record.Index,
		"status":      record.Status,
		"duration_ms": record.DurationMS,
	}, true
}

func toolStartRecord(message string) (map[string]any, bool) {
	var record struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Text      string `json:"text"`
		StartedAt *int64 `json:"started_at"`
	}
	if err := json.Unmarshal([]byte(message), &record); err != nil {
		return nil, false
	}
	output := map[string]any{
		"type": "tool_start",
		"kind": strings.TrimSpace(record.Kind),
		"text": record.Text,
	}
	if id := strings.TrimSpace(record.ID); id != "" {
		output["id"] = id
	}
	if record.StartedAt != nil && *record.StartedAt >= 0 {
		output["started_at"] = *record.StartedAt
	}
	return output, true
}

func toolEndRecord(message string) (map[string]any, bool) {
	var record struct {
		ID         string `json:"id"`
		Kind       string `json:"kind"`
		Status     string `json:"status"`
		DurationMS int64  `json:"duration_ms"`
	}
	if err := json.Unmarshal([]byte(message), &record); err != nil ||
		record.DurationMS < 0 ||
		!validStatus(record.Status) {
		return nil, false
	}
	output := map[string]any{
		"type":        "tool_end",
		"status":      record.Status,
		"duration_ms": record.DurationMS,
	}
	if id := strings.TrimSpace(record.ID); id != "" {
		output["id"] = id
	}
	if kind := strings.TrimSpace(record.Kind); kind != "" {
		output["kind"] = kind
	}
	return output, true
}

func validStatus(status string) bool {
	return status == "passed" || status == "failed"
}
