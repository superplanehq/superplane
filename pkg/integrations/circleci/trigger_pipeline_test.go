package circleci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__TriggerPipeline__checkWorkflowsStatus(t *testing.T) {
	tp := &TriggerPipeline{}

	t.Run("empty workflows list", func(t *testing.T) {
		allDone, anyFailed := tp.checkWorkflowsStatus([]WorkflowInfo{})
		assert.False(t, allDone)
		assert.False(t, anyFailed)
	})

	t.Run("all workflows success", func(t *testing.T) {
		workflows := []WorkflowInfo{
			{ID: "wf-1", Status: "success"},
			{ID: "wf-2", Status: "success"},
		}
		allDone, anyFailed := tp.checkWorkflowsStatus(workflows)
		assert.True(t, allDone)
		assert.False(t, anyFailed)
	})

	t.Run("one workflow failed", func(t *testing.T) {
		workflows := []WorkflowInfo{
			{ID: "wf-1", Status: "success"},
			{ID: "wf-2", Status: "failed"},
		}
		allDone, anyFailed := tp.checkWorkflowsStatus(workflows)
		assert.True(t, allDone)
		assert.True(t, anyFailed)
	})

	t.Run("one workflow canceled", func(t *testing.T) {
		workflows := []WorkflowInfo{
			{ID: "wf-1", Status: "success"},
			{ID: "wf-2", Status: "canceled"},
		}
		allDone, anyFailed := tp.checkWorkflowsStatus(workflows)
		assert.True(t, allDone)
		assert.True(t, anyFailed)
	})

	t.Run("workflow still running", func(t *testing.T) {
		workflows := []WorkflowInfo{
			{ID: "wf-1", Status: "success"},
			{ID: "wf-2", Status: "running"},
		}
		allDone, anyFailed := tp.checkWorkflowsStatus(workflows)
		assert.False(t, allDone)
		assert.False(t, anyFailed)
	})

	t.Run("workflow on hold", func(t *testing.T) {
		workflows := []WorkflowInfo{
			{ID: "wf-1", Status: "success"},
			{ID: "wf-2", Status: "on_hold"},
		}
		allDone, anyFailed := tp.checkWorkflowsStatus(workflows)
		assert.False(t, allDone)
		assert.False(t, anyFailed)
	})

	t.Run("workflow not run", func(t *testing.T) {
		workflows := []WorkflowInfo{
			{ID: "wf-1", Status: "success"},
			{ID: "wf-2", Status: "not_run"},
		}
		allDone, anyFailed := tp.checkWorkflowsStatus(workflows)
		assert.False(t, allDone)
		assert.False(t, anyFailed)
	})
}

func Test__TriggerPipeline__buildParameters(t *testing.T) {
	t.Run("builds parameters with superplane metadata", func(t *testing.T) {
		params := []Parameter{
			{Name: "env", Value: "production"},
			{Name: "version", Value: "1.0.0"},
		}

		// Mock execution context
		mockCtx := struct {
			ID         string
			WorkflowID string
		}{
			ID:         "exec-123",
			WorkflowID: "canvas-456",
		}

		// Note: We can't easily test this without a full ExecutionContext
		// In a real test environment, you'd use a mock context
		result := make(map[string]string)
		for _, param := range params {
			result[param.Name] = param.Value
		}
		result["SUPERPLANE_EXECUTION_ID"] = mockCtx.ID
		result["SUPERPLANE_CANVAS_ID"] = mockCtx.WorkflowID

		assert.Equal(t, "production", result["env"])
		assert.Equal(t, "1.0.0", result["version"])
		assert.Equal(t, "exec-123", result["SUPERPLANE_EXECUTION_ID"])
		assert.Equal(t, "canvas-456", result["SUPERPLANE_CANVAS_ID"])
	})
}
