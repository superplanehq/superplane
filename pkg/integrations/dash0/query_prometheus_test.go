package dash0

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

func Test__QueryPrometheus__Setup(t *testing.T) {
	component := QueryPrometheus{}

	t.Run("query is required", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"query": ""},
		})

		require.ErrorContains(t, err, "query is required")
	})

	t.Run("query cannot be empty", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"query": "   "},
		})

		require.ErrorContains(t, err, "query cannot be empty")
	})

	t.Run("dataset is required", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"query": "up", "dataset": ""},
		})

		require.ErrorContains(t, err, "dataset is required")
	})

	t.Run("range query requires start", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"query":   "up",
				"dataset": "default",
				"type":    "range",
			},
		})

		require.ErrorContains(t, err, "start is required for range queries")
	})

	t.Run("range query requires end", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"query":   "up",
				"dataset": "default",
				"type":    "range",
				"start":   "now-5m",
			},
		})

		require.ErrorContains(t, err, "end is required for range queries")
	})

	t.Run("range query requires step", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"query":   "up",
				"dataset": "default",
				"type":    "range",
				"start":   "now-5m",
				"end":     "now",
			},
		})

		require.ErrorContains(t, err, "step is required for range queries")
	})

	t.Run("valid instant query setup", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"query":   "up",
				"dataset": "default",
				"type":    "instant",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid range query setup", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"query":   "up",
				"dataset": "default",
				"type":    "range",
				"start":   "now-5m",
				"end":     "now",
				"step":    "15s",
			},
		})

		require.NoError(t, err)
	})
}

func Test__QueryPrometheus__Execute(t *testing.T) {
	component := QueryPrometheus{}

	t.Run("successful instant query", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {"service_name": "test"},
										"value": [1234567890, "1"]
									}
								]
							}
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query":   "up",
				"dataset": "default",
				"type":    "instant",
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "dash0.prometheus.response", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("successful range query", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "matrix",
								"result": [
									{
										"metric": {"service_name": "test"},
										"values": [[1234567890, "1"], [1234567900, "2"]]
									}
								]
							}
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query":   "up",
				"dataset": "default",
				"type":    "range",
				"start":   "now-5m",
				"end":     "now",
				"step":    "15s",
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "dash0.prometheus.response", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("query execution failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"status":"error","errorType":"bad_data","error":"parse error"}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query":   "invalid query",
				"dataset": "default",
				"type":    "instant",
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute Prometheus query")
	})
}
