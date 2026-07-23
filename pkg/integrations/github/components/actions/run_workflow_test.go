package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression test for #6264: the Run Workflow example output must expose the
// workflow-run fields flat under `data` (data.id, data.status, ...), matching
// what both emit paths (workflow_run webhook and poll fallback) actually emit.
// It must NOT nest them under `data.workflow_run`.
func Test__RunWorkflow__ExampleOutputMatchesEmittedShape(t *testing.T) {
	example := (&RunWorkflow{}).ExampleOutput()

	require.Equal(t, WorkflowPayloadType, example["type"])
	require.Contains(t, example, "timestamp")

	data, ok := example["data"].(map[string]any)
	require.True(t, ok, "example output must have a `data` object")

	// The example must not re-introduce the incorrect nesting.
	assert.NotContains(t, data, "workflow_run",
		"example output must emit the run flat at `data`, not nested under `data.workflow_run`")

	// The documented, stable contract fields must be present at the top of `data`.
	for _, field := range []string{"id", "status", "conclusion", "html_url"} {
		assert.Contains(t, data, field, "expected flat field data.%s", field)
	}

	// Tie the example to the real webhook emit path: the object emitted by
	// metadataFromPayload (used by HandleWebhook) must contain the same fields.
	payload := map[string]any{
		"action": "completed",
		"workflow_run": map[string]any{
			"id":         float64(9001),
			"status":     "completed",
			"conclusion": "success",
			"html_url":   "https://github.com/acme/widgets/actions/runs/9001",
		},
	}

	_, emitted, err := metadataFromPayload(payload)
	require.NoError(t, err)
	for field := range data {
		assert.Contains(t, emitted, field,
			"example field data.%s must exist in the emitted payload", field)
	}
}
