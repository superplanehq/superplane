package github_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/github"
)

func Test_GitHubEventHandler_Status(t *testing.T) {
	handler := &github.GitHubEventHandler{}

	t.Run("status from workflow run event", func(t *testing.T) {
		payload := `{
			"action": "completed",
			"workflow_run": {
				"id": 123456789,
				"status": "completed",
				"conclusion": "success"
			},
			"repository": {
				"full_name": "owner/repo"
			}
		}`

		resource, err := handler.Status("workflow_run", []byte(payload))
		require.NoError(t, err)
		assert.NotNil(t, resource)

		workflowRun, ok := resource.(*github.WorkflowRun)
		require.True(t, ok)
		assert.Equal(t, "owner/repo:123456789", workflowRun.Id())
		assert.Equal(t, "completed", workflowRun.Status)
		assert.Equal(t, "success", workflowRun.Conclusion)
		assert.Equal(t, "owner/repo", workflowRun.Repository)
		assert.True(t, workflowRun.Finished())
		assert.True(t, workflowRun.Successful())
	})

	t.Run("returns error for invalid payload", func(t *testing.T) {
		payload := `invalid json`

		resource, err := handler.Status("workflow_run", []byte(payload))
		assert.Error(t, err)
		assert.Nil(t, resource)
	})
}
