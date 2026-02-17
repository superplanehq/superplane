package harness

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__ExtractPipelineWebhookEvent__PrefersExpectedPaths(t *testing.T) {
	payload := map[string]any{
		"eventType":          "PIPELINE_END",
		"pipelineIdentifier": "top-level-pipeline",
		"status":             "RUNNING",
		"eventData": map[string]any{
			"planExecutionId":    "exec-event-data",
			"pipelineIdentifier": "event-data-pipeline",
			"nodeStatus":         "completed",
		},
		"data": map[string]any{
			"planExecutionId":    "exec-data",
			"pipelineIdentifier": "data-pipeline",
			"status":             "FAILED",
		},
	}

	event := extractPipelineWebhookEvent(payload)

	assert.Equal(t, "exec-event-data", event.ExecutionID)
	assert.Equal(t, "event-data-pipeline", event.PipelineIdentifier)
	assert.Equal(t, "completed", event.Status)
	assert.Equal(t, "PIPELINE_END", event.EventType)
}

func Test__ExtractPipelineWebhookEvent__FallsBackToRecursiveSearch(t *testing.T) {
	payload := map[string]any{
		"wrapper": map[string]any{
			"inner": map[string]any{
				"execution_id": "exec-1",
				"pipeline_id":  "deploy",
				"event":        "PipelineEnd",
				"details": map[string]any{
					"executionStatus": "FAILED",
				},
			},
		},
	}

	event := extractPipelineWebhookEvent(payload)

	assert.Equal(t, "exec-1", event.ExecutionID)
	assert.Equal(t, "deploy", event.PipelineIdentifier)
	assert.Equal(t, "FAILED", event.Status)
	assert.Equal(t, "PipelineEnd", event.EventType)
}

func Test__IsPipelineCompletedEventType(t *testing.T) {
	assert.True(t, isPipelineCompletedEventType("PIPELINE_END"))
	assert.True(t, isPipelineCompletedEventType("PipelineEnd"))
	assert.True(t, isPipelineCompletedEventType("pipeline-end"))
	assert.True(t, isPipelineCompletedEventType("pipeline.completed"))
	assert.False(t, isPipelineCompletedEventType("STAGE_END"))
	assert.False(t, isPipelineCompletedEventType(""))
}

func Test__CanonicalStatus__TreatsErroredAsFailedTerminal(t *testing.T) {
	assert.Equal(t, "failed", canonicalStatus("ERRORED"))
	assert.True(t, isTerminalStatus("ERRORED"))
}
