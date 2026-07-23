package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Run Workflow component emits the workflow-run object flat under `data`
// (both the workflow_run webhook path and the poll fallback). The example
// output must match that shape so expressions copied from it (e.g. data.id)
// work against real payloads.
//
// Regression test for https://github.com/superplanehq/superplane/issues/6264
func Test__RunWorkflow__ExampleOutputMatchesEmittedShape(t *testing.T) {
	example := (&RunWorkflow{}).ExampleOutput()

	data, ok := example["data"].(map[string]any)
	require.True(t, ok, "example output must contain a data object")

	// Fields must be flat under data, not nested under data.workflow_run.
	_, nested := data["workflow_run"]
	assert.False(t, nested, "example output must not nest fields under data.workflow_run")

	for _, key := range []string{"id", "status", "conclusion", "html_url"} {
		_, present := data[key]
		assert.Truef(t, present, "example output data must contain %q", key)
	}

	// The example data keys must be a subset of what the webhook path emits,
	// which is the raw workflow_run object returned by metadataFromPayload.
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

	for key := range data {
		_, present := emitted[key]
		assert.Truef(t, present, "example output data key %q is not present in the emitted payload", key)
	}
}
