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
	component := &GetTestMetrics{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing project slug -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "",
				"workflowName": "build",
			},
		})

		require.ErrorContains(t, err, "project slug is required")
	})

	t.Run("missing workflow name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "",
			},
		})

		require.ErrorContains(t, err, "workflow name is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "build-test-deploy",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetTestMetrics__Execute(t *testing.T) {
	component := &GetTestMetrics{}

	t.Run("valid request -> emits test metrics", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"most_failed_tests": [
							{
								"test_name": "TestAuth",
								"classname": "auth_test.go",
								"failed_runs": 5,
								"total_runs": 42,
								"flaky": true,
								"source": "go",
								"file": "pkg/auth/auth_test.go"
							}
						],
						"slowest_tests": [
							{
								"test_name": "TestMigration",
								"classname": "migration_test.go",
								"failed_runs": 1,
								"total_runs": 42,
								"flaky": false,
								"p50_duration_secs": 12.5,
								"source": "go",
								"file": "pkg/db/migration_test.go"
							}
						],
						"total_test_runs": 42,
						"test_runs": []
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "build-test-deploy",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, GetTestMetricsPayloadType, execState.Type)

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/insights/gh/acme/my-app/workflows/build-test-deploy/test-metrics")
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "nonexistent",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.ErrorContains(t, err, "failed to get test metrics")
	})

	t.Run("with special workflow name -> escapes workflow name in path", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"most_failed_tests": [],
						"slowest_tests": [],
						"total_test_runs": 0,
						"test_runs": []
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "build/test #1",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/workflows/build%2Ftest%20%231/test-metrics")
	})
}
