package circleci

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetTestMetrics__Setup(t *testing.T) {
	c := &GetTestMetrics{}

	t.Run("missing project slug -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "",
				"workflowName": "build",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "project slug is required")
	})

	t.Run("missing workflow name -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/org/my-repo",
				"workflowName": "",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "workflow name is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"proj-123","name":"my-repo","slug":"gh/org/my-repo"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token-123",
			},
		}

		metadataCtx := &contexts.MetadataContext{Metadata: map[string]any{}}

		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/org/my-repo",
				"workflowName": "build",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
	})
}

func Test__GetTestMetrics__Execute(t *testing.T) {
	c := &GetTestMetrics{}

	t.Run("fetches test metrics -> emits to default channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"average_test_count":150,"most_failed_tests":[],"most_failed_tests_extra":0,"slowest_tests":[],"slowest_tests_extra":0,"total_test_runs":200,"test_runs":[]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token-123",
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/org/my-repo",
				"workflowName": "build",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "circleci.insights.test-metrics", executionState.Type)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/insights/gh/org/my-repo/workflows/build/test-metrics")
	})
}
