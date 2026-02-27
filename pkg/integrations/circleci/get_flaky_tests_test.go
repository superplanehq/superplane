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

func Test__GetFlakyTests__Setup(t *testing.T) {
	component := &GetFlakyTests{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing project slug -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug": "",
			},
		})

		require.ErrorContains(t, err, "project slug is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug": "gh/acme/my-app",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetFlakyTests__Execute(t *testing.T) {
	component := &GetFlakyTests{}

	t.Run("valid request -> emits flaky tests data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"flaky-tests": [
							{
								"test-name": "TestAuth",
								"classname": "auth_test.go",
								"pipeline-name": "build-pipeline",
								"workflow-name": "build-test-deploy",
								"job-name": "test",
								"times-flaked": 8,
								"source": "go",
								"file": "pkg/auth/auth_test.go"
							}
						],
						"total-flaky-tests": 1
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
				"projectSlug": "gh/acme/my-app",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, GetFlakyTestsPayloadType, execState.Type)

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/insights/gh/acme/my-app/flaky-tests")
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
				"projectSlug": "gh/acme/nonexistent",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.ErrorContains(t, err, "failed to get flaky tests")
	})
}
