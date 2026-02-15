package harness

import (
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
	case "error":
		return "failed"
	case "cancelled", "canceled", "stopped", "rejected":
		return "aborted"
	default:
		return normalized
	}
}

func isTerminalStatus(status string) bool {
	normalized := canonicalStatus(status)
	switch normalized {
	case "succeeded", "failed", "aborted", "expired":
		return true
	default:
		return false
	}
}

func isSuccessStatus(status string) bool {
	return canonicalStatus(status) == "succeeded"
}

func extractPipelineWebhookEvent(payload map[string]any) pipelineWebhookEvent {
	event := pipelineWebhookEvent{}

	event.ExecutionID = firstNonEmpty(
		findStringRecursive(payload, []string{"planExecutionId", "executionId", "execution_id"}, 0),
	)
	event.PipelineIdentifier = firstNonEmpty(
		findStringRecursive(payload, []string{"pipelineIdentifier", "pipelineId", "pipeline_id"}, 0),
	)
	event.Status = firstNonEmpty(
		findStringRecursive(payload, []string{"status", "pipelineStatus", "planExecutionStatus", "executionStatus", "nodeStatus"}, 0),
	)
	event.EventType = firstNonEmpty(
		findStringRecursive(payload, []string{"eventType", "type", "event"}, 0),
	)

	return event
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

		for _, nested := range value {
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
