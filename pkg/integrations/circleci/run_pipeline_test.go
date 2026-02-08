package circleci

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__RunPipeline__checkWorkflowsStatus(t *testing.T) {
	tp := &RunPipeline{}

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

func Test__RunPipeline__buildParameters(t *testing.T) {
	t.Run("builds parameters with superplane metadata", func(t *testing.T) {
		tp := &RunPipeline{}
		params := []Parameter{
			{Name: "env", Value: "production"},
			{Name: "version", Value: "1.0.0"},
		}

		mockCtx := core.ExecutionContext{
			ID:         uuid.MustParse("00000000-0000-0000-0000-000000000123"),
			WorkflowID: "canvas-456",
		}

		result := tp.buildParameters(mockCtx, params)

		assert.Equal(t, "production", result["env"])
		assert.Equal(t, "1.0.0", result["version"])
		assert.Equal(t, "00000000-0000-0000-0000-000000000123", result["SUPERPLANE_EXECUTION_ID"])
		assert.Equal(t, "canvas-456", result["SUPERPLANE_CANVAS_ID"])
	})
}
