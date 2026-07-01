package cloudflare

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

func Test__DeployWorker__Setup(t *testing.T) {
	component := &DeployWorker{}

	t.Run("missing accountId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":          "",
				"scriptName":         "w",
				"provisionIfMissing": false,
				"source":             deployWorkerScriptSourceInline,
				"inlineCode":         "export default { fetch() { return new Response('ok'); } };",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "accountId is required")
	})

	t.Run("missing scriptName returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":          "acc",
				"scriptName":         "",
				"provisionIfMissing": false,
				"source":             deployWorkerScriptSourceInline,
				"inlineCode":         "export default { fetch() { return new Response('ok'); } };",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "scriptName is required")
	})

	t.Run("inline without code returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":          "acc",
				"scriptName":         "w",
				"provisionIfMissing": false,
				"source":             deployWorkerScriptSourceInline,
				"inlineCode":         "",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "inlineCode is required")
	})

	t.Run("url without scriptUrl returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":          "acc",
				"scriptName":         "w",
				"provisionIfMissing": false,
				"source":             deployWorkerScriptSourceURL,
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "scriptUrl is required")
	})

	t.Run("invalid source returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":          "acc",
				"scriptName":         "w",
				"provisionIfMissing": false,
				"source":             "invalid",
				"inlineCode":         "x",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "source must be")
	})

	t.Run("valid inline passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":          "acc",
				"scriptName":         "w",
				"provisionIfMissing": false,
				"source":             deployWorkerScriptSourceInline,
				"inlineCode":         "export default { fetch() { return new Response('ok'); } };",
			},
			Integration: &contexts.IntegrationContext{},
		}
		require.NoError(t, component.Setup(ctx))
	})

	t.Run("invalid observability sampling rate returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":          "acc",
				"scriptName":         "w",
				"provisionIfMissing": true,
				"source":             deployWorkerScriptSourceInline,
				"inlineCode":         "export default { fetch() { return new Response('ok'); } };",
				"provision": map[string]any{
					"observabilityHeadSamplingRate": "x",
				},
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "observabilityHeadSamplingRate")
	})
}

func Test__DeployWorker__Execute(t *testing.T) {
	component := &DeployWorker{}
	inline := "export default { fetch() { return new Response('ok'); } };"

	t.Run("successful inline deploy emits and calls APIs", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "ver-1" }
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {
							"id": "dep-1",
							"strategy": "percentage",
							"versions": [{"percentage": 100, "version_id": "ver-1"}]
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token"},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId":          "acc123",
				"scriptName":         "my-worker",
				"provisionIfMissing": false,
				"source":             deployWorkerScriptSourceInline,
				"inlineCode":         inline,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		assert.True(t, execState.Passed)
		assert.Equal(t, "cloudflare.worker.deployed", execState.Type)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/accounts/acc123/workers/scripts/my-worker/versions")
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/accounts/acc123/workers/scripts/my-worker/deployments")
	})

	t.Run("url source downloads then uploads", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(inline)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "ver-2" }
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "dep-2", "strategy": "percentage" }
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token"},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId":          "acc123",
				"scriptName":         "w2",
				"provisionIfMissing": false,
				"source":             deployWorkerScriptSourceURL,
				"scriptUrl":          "https://cdn.example.com/worker.js",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, "https://cdn.example.com/worker.js", httpContext.Requests[0].URL.String())
	})

	t.Run("url source with empty body returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token"},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId":          "acc123",
				"scriptName":         "w-empty",
				"provisionIfMissing": false,
				"source":             deployWorkerScriptSourceURL,
				"scriptUrl":          "https://cdn.example.com/empty.js",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("provision then upload and deploy calls three endpoints", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "wid", "name": "fresh" }
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "ver-fresh" }
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "dep-fresh", "strategy": "percentage" }
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token"},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId":  "acc123",
				"scriptName": "fresh",
				"source":     deployWorkerScriptSourceInline,
				"inlineCode": inline,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		assert.Equal(t, "cloudflare.worker.deployed", execState.Type)
		require.Len(t, httpContext.Requests, 3)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/workers/workers")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/workers/scripts/fresh/versions")
		assert.Contains(t, httpContext.Requests[2].URL.String(), "/workers/scripts/fresh/deployments")
	})

	t.Run("provision by default calls workers then upload and deploy", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "wid-1", "name": "new-worker" }
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "ver-new" }
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "dep-new", "strategy": "percentage" }
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token"},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId":  "acc123",
				"scriptName": "new-worker",
				"source":     deployWorkerScriptSourceInline,
				"inlineCode": inline,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		require.Len(t, httpContext.Requests, 3)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/accounts/acc123/workers/workers")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/workers/scripts/new-worker/versions")
		assert.Contains(t, httpContext.Requests[2].URL.String(), "/workers/scripts/new-worker/deployments")
	})

	t.Run("provision conflict is ignored then upload succeeds", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusConflict,
					Body: io.NopCloser(strings.NewReader(`{
						"success": false,
						"errors": [{"code": 10009, "message": "A worker with this name already exists"}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "ver-dup" }
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": { "id": "dep-dup", "strategy": "percentage" }
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token"},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId":  "acc123",
				"scriptName": "existing",
				"source":     deployWorkerScriptSourceInline,
				"inlineCode": inline,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		require.Len(t, httpContext.Requests, 3)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/versions")
	})
}
