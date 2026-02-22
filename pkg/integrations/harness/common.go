package harness

import (
	"sort"
	"strings"
)

type pipelineWebhookEvent struct {
	ExecutionID        string
	PipelineIdentifier string
	Status             string
	EventType          string
}

func normalizeStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func canonicalStatus(status string) string {
	normalized := normalizeStatus(status)

	switch normalized {
	case "success", "completed":
		return "succeeded"
	case "error", "errored":
		return "failed"
	case "cancelled", "canceled", "stopped", "rejected":
		return "aborted"
	default:
		return normalized
	}
}

func isTerminalStatus(status string) bool {
	normalized := canonicalStatus(status)
	return isCanonicalTerminalStatus(normalized)
}

func isCanonicalTerminalStatus(status string) bool {
	switch status {
	case "succeeded", "failed", "aborted", "expired":
		return true
	default:
		return false
	}
}

func isSuccessStatus(status string) bool {
	return canonicalStatus(status) == "succeeded"
}

func isPipelineCompletedEventType(eventType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(eventType))
	if normalized == "" {
		return false
	}

	replacer := strings.NewReplacer("_", "", "-", "", ".", "", " ", "")
	normalized = replacer.Replace(normalized)

	switch normalized {
	case "pipelineend", "pipelinecompleted":
		return true
	default:
		return false
	}
}

func extractPipelineWebhookEvent(payload map[string]any) pipelineWebhookEvent {
	event := pipelineWebhookEvent{}

	event.ExecutionID = firstNonEmpty(
		findStringByPaths(payload,
			[]string{"eventData", "planExecutionId"},
			[]string{"eventData", "executionId"},
			[]string{"eventData", "execution_id"},
			[]string{"data", "planExecutionId"},
			[]string{"data", "executionId"},
			[]string{"data", "execution_id"},
			[]string{"planExecutionId"},
			[]string{"executionId"},
			[]string{"execution_id"},
		),
		findStringRecursive(payload, []string{"planExecutionId", "executionId", "execution_id"}, 0),
	)
	event.PipelineIdentifier = firstNonEmpty(
		findStringByPaths(payload,
			[]string{"eventData", "pipelineIdentifier"},
			[]string{"eventData", "pipelineId"},
			[]string{"eventData", "pipeline_id"},
			[]string{"data", "pipelineIdentifier"},
			[]string{"data", "pipelineId"},
			[]string{"data", "pipeline_id"},
			[]string{"pipelineIdentifier"},
			[]string{"pipelineId"},
			[]string{"pipeline_id"},
		),
		findStringRecursive(payload, []string{"pipelineIdentifier", "pipelineId", "pipeline_id"}, 0),
	)
	event.Status = firstNonEmpty(
		findStringByPaths(payload,
			[]string{"eventData", "nodeStatus"},
			[]string{"eventData", "status"},
			[]string{"eventData", "pipelineStatus"},
			[]string{"eventData", "planExecutionStatus"},
			[]string{"eventData", "executionStatus"},
			[]string{"data", "nodeStatus"},
			[]string{"data", "status"},
			[]string{"data", "pipelineStatus"},
			[]string{"data", "planExecutionStatus"},
			[]string{"data", "executionStatus"},
			[]string{"nodeStatus"},
			[]string{"status"},
			[]string{"pipelineStatus"},
			[]string{"planExecutionStatus"},
			[]string{"executionStatus"},
		),
		findStringRecursive(payload, []string{"status", "pipelineStatus", "planExecutionStatus", "executionStatus", "nodeStatus"}, 0),
	)
	event.EventType = firstNonEmpty(
		findStringByPaths(payload,
			[]string{"eventType"},
			[]string{"eventData", "eventType"},
			[]string{"data", "eventType"},
			[]string{"type"},
			[]string{"event"},
		),
		findStringRecursive(payload, []string{"eventType", "type", "event"}, 0),
	)

	return event
}

func findStringByPaths(input map[string]any, paths ...[]string) string {
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}

		if text := readString(readAnyPath(input, path...)); text != "" {
			return text
		}
	}

	return ""
}

func findStringRecursive(input any, keys []string, depth int) string {
	if depth > 5 {
		return ""
	}

	switch value := input.(type) {
	case map[string]any:
		for _, key := range keys {
			if found, ok := value[key]; ok {
				if text := readString(found); text != "" {
					return text
				}
			}
		}

		nestedKeys := make([]string, 0, len(value))
		for key := range value {
			nestedKeys = append(nestedKeys, key)
		}
		sort.Strings(nestedKeys)

		for _, key := range nestedKeys {
			nested := value[key]
			if text := findStringRecursive(nested, keys, depth+1); text != "" {
				return text
			}
		}
	case []any:
		for _, item := range value {
			if text := findStringRecursive(item, keys, depth+1); text != "" {
				return text
			}
		}
	}

	return ""
}
