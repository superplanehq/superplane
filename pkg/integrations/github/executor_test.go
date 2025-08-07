package github_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/github"
)

func Test_GitHubExecutor_Validate(t *testing.T) {
	resourceManager := &github.GitHubResourceManager{}
	resource := &github.Repository{
		ID:             123456789,
		RepositoryName: "test-repo",
	}

	executor, err := github.NewGitHubExecutor(resourceManager, resource)
	require.NoError(t, err)

	t.Run("workflow is required", func(t *testing.T) {
		spec, err := json.Marshal(&github.ExecutorSpec{
			Workflow: "",
			Ref:      "main",
		})
		require.NoError(t, err)

		err = executor.Validate(context.Background(), spec)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflow is required")
	})

	t.Run("ref is required", func(t *testing.T) {
		spec, err := json.Marshal(&github.ExecutorSpec{
			Workflow: "ci.yml",
			Ref:      "",
		})
		require.NoError(t, err)

		err = executor.Validate(context.Background(), spec)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ref is required")
	})

	t.Run("invalid JSON spec", func(t *testing.T) {
		err := executor.Validate(context.Background(), []byte("invalid json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error unmarshaling spec data")
	})
}
