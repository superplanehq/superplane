package deployments

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__validateCreateDeploymentConfig(t *testing.T) {
	t.Parallel()

	t.Run("accepts non-empty ref and environment", func(t *testing.T) {
		t.Parallel()
		err := validateCreateDeploymentConfig(CreateDeploymentConfiguration{
			Ref:         "feature/foo",
			Environment: "preview-pr-1",
		})
		require.NoError(t, err)
	})

	t.Run("trims ref and environment", func(t *testing.T) {
		t.Parallel()
		err := validateCreateDeploymentConfig(CreateDeploymentConfiguration{
			Ref:         "  main  ",
			Environment: "  prod  ",
		})
		require.NoError(t, err)
	})

	t.Run("rejects empty ref", func(t *testing.T) {
		t.Parallel()
		err := validateCreateDeploymentConfig(CreateDeploymentConfiguration{
			Ref:         "",
			Environment: "env",
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "ref is required")
	})

	t.Run("rejects whitespace-only ref", func(t *testing.T) {
		t.Parallel()
		err := validateCreateDeploymentConfig(CreateDeploymentConfiguration{
			Ref:         "   ",
			Environment: "env",
		})
		require.Error(t, err)
	})

	t.Run("rejects empty environment", func(t *testing.T) {
		t.Parallel()
		err := validateCreateDeploymentConfig(CreateDeploymentConfiguration{
			Ref:         "main",
			Environment: "",
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "environment is required")
	})
}

func Test__normalizeRequiredContexts(t *testing.T) {
	t.Parallel()

	assert.Empty(t, normalizeRequiredContexts(nil))
	assert.Empty(t, normalizeRequiredContexts([]string{}))
	assert.Empty(t, normalizeRequiredContexts([]string{"", "  ", "\t"}))

	assert.Equal(t, []string{"ci/build"}, normalizeRequiredContexts([]string{"ci/build"}))
	assert.Equal(t, []string{"ci/build", "ci/lint"}, normalizeRequiredContexts([]string{"  ci/build  ", "ci/lint"}))
	assert.Equal(t, []string{"ci/build"}, normalizeRequiredContexts([]string{"ci/build", "ci/build"}))
}

func Test__newGitHubDeploymentRequest(t *testing.T) {
	t.Parallel()

	t.Run("sets empty required_contexts when contexts empty", func(t *testing.T) {
		t.Parallel()
		req := newGitHubDeploymentRequest(CreateDeploymentConfiguration{
			Ref:         "feat/x",
			Environment: "preview",
		})
		require.NotNil(t, req.RequiredContexts)
		assert.Empty(t, *req.RequiredContexts)
		assert.Equal(t, "feat/x", *req.Ref)
		assert.Equal(t, "preview", *req.Environment)
	})

	t.Run("sends specific required contexts", func(t *testing.T) {
		t.Parallel()
		req := newGitHubDeploymentRequest(CreateDeploymentConfiguration{
			Ref:              "main",
			Environment:      "prod",
			RequiredContexts: []string{"  ci/build  ", "ci/lint", "ci/build"},
		})
		require.NotNil(t, req.RequiredContexts)
		assert.Equal(t, []string{"ci/build", "ci/lint"}, *req.RequiredContexts)
	})

	t.Run("includes optional description and task", func(t *testing.T) {
		t.Parallel()
		req := newGitHubDeploymentRequest(CreateDeploymentConfiguration{
			Ref:         "main",
			Environment: "staging",
			Description: "ship it",
			Task:        "deploy",
		})
		require.NotNil(t, req.Description)
		assert.Equal(t, "ship it", *req.Description)
		require.NotNil(t, req.Task)
		assert.Equal(t, "deploy", *req.Task)
	})

	t.Run("trims ref environment description and task", func(t *testing.T) {
		t.Parallel()
		req := newGitHubDeploymentRequest(CreateDeploymentConfiguration{
			Ref:         "  r  ",
			Environment: "  e  ",
			Description: "  d  ",
			Task:        "  t  ",
		})
		assert.Equal(t, "r", *req.Ref)
		assert.Equal(t, "e", *req.Environment)
		require.NotNil(t, req.Description)
		assert.Equal(t, "d", *req.Description)
		require.NotNil(t, req.Task)
		assert.Equal(t, "t", *req.Task)
	})

	t.Run("sets boolean flags on request", func(t *testing.T) {
		t.Parallel()
		req := newGitHubDeploymentRequest(CreateDeploymentConfiguration{
			Ref:                   "main",
			Environment:           "x",
			AutoMerge:             true,
			TransientEnvironment:  true,
			ProductionEnvironment: true,
		})
		require.NotNil(t, req.AutoMerge)
		assert.True(t, *req.AutoMerge)
		require.NotNil(t, req.TransientEnvironment)
		assert.True(t, *req.TransientEnvironment)
		require.NotNil(t, req.ProductionEnvironment)
		assert.True(t, *req.ProductionEnvironment)
	})
}
