package circleci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__RunPipeline__buildParameters(t *testing.T) {
	t.Run("builds parameters map", func(t *testing.T) {
		tp := &RunPipeline{}
		params := []Parameter{
			{Name: "env", Value: "production"},
			{Name: "version", Value: "1.0.0"},
		}

		result := tp.buildParameters(params)

		assert.Equal(t, "production", result["env"])
		assert.Equal(t, "1.0.0", result["version"])
		assert.Len(t, result, 2)
	})
}

func Test__RunPipeline__evaluateWorkflows(t *testing.T) {
	tp := &RunPipeline{}

	workflows := func(statuses ...string) []WorkflowResponse {
		result := make([]WorkflowResponse, 0, len(statuses))
		for _, status := range statuses {
			result = append(result, WorkflowResponse{Status: status})
		}

		return result
	}

	t.Run("single successful workflow succeeds", func(t *testing.T) {
		assert.Equal(t, pipelineSucceeded, tp.evaluateWorkflows(workflows("success")))
	})

	t.Run("single failed workflow fails", func(t *testing.T) {
		assert.Equal(t, pipelineFailed, tp.evaluateWorkflows(workflows("failed")))
	})

	t.Run("all workflows successful succeeds", func(t *testing.T) {
		assert.Equal(t, pipelineSucceeded, tp.evaluateWorkflows(workflows("success", "success", "success")))
	})

	// The defect in #6925: workflows[0] succeeded, so the pipeline was reported
	// on the success channel even though a later workflow failed.
	t.Run("later workflow failing fails the pipeline", func(t *testing.T) {
		assert.Equal(t, pipelineFailed, tp.evaluateWorkflows(workflows("success", "failed")))
	})

	t.Run("failure anywhere fails the pipeline", func(t *testing.T) {
		assert.Equal(t, pipelineFailed, tp.evaluateWorkflows(workflows("success", "success", "canceled")))
		assert.Equal(t, pipelineFailed, tp.evaluateWorkflows(workflows("failed", "success")))
	})

	// The second half of #6925: workflows[0] reached a final status, so a result
	// was emitted while another workflow was still running.
	t.Run("later workflow still running keeps polling", func(t *testing.T) {
		assert.Equal(t, pipelineRunning, tp.evaluateWorkflows(workflows("success", "running")))
	})

	t.Run("running takes precedence over an earlier failure", func(t *testing.T) {
		assert.Equal(t, pipelineRunning, tp.evaluateWorkflows(workflows("failed", "running")))
	})

	t.Run("on_hold is not final", func(t *testing.T) {
		assert.Equal(t, pipelineRunning, tp.evaluateWorkflows(workflows("success", "on_hold")))
	})

	t.Run("every failure status is treated as a failure", func(t *testing.T) {
		for _, status := range []string{"failed", "canceled", "error", "failing", "unauthorized"} {
			assert.Equal(t, pipelineFailed, tp.evaluateWorkflows(workflows("success", status)), status)
		}
	})

	t.Run("no workflows is not a failure", func(t *testing.T) {
		assert.Equal(t, pipelineSucceeded, tp.evaluateWorkflows(workflows()))
	})
}
