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

func Test__GetRecentWorkflowRuns__Setup(t *testing.T) {
	c := &GetRecentWorkflowRuns{}

	t.Run("missing project slug -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug": "",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "project slug is required")
	})

	t.Run("valid project slug -> success", func(t *testing.T) {
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
				"projectSlug": "gh/org/my-repo",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
	})
}

func Test__GetRecentWorkflowRuns__Execute(t *testing.T) {
	c := &GetRecentWorkflowRuns{}

	t.Run("fetches insights workflows -> emits to default channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"items":[{"name":"build","metrics":{"success_rate":0.95},"window_start":"2024-01-01T00:00:00Z","window_end":"2024-02-01T00:00:00Z"}]}`)),
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
				"projectSlug": "gh/org/my-repo",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "circleci.insights.workflows", executionState.Type)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/insights/gh/org/my-repo/workflows")
	})
}
