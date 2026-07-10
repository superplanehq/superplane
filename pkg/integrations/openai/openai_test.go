package openai

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OpenAI__Sync(t *testing.T) {
	o := &OpenAI{}

	t.Run("success with api key -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
		}

		err := o.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.openai.com/v1/models", httpContext.Requests[0].URL.String())
	})

	t.Run("success with api and admin keys -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(usagePageBody)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "test-admin-key",
			},
		}

		err := o.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/organization/usage/completions")
		assert.Equal(t, "Bearer test-admin-key", httpContext.Requests[1].Header.Get("Authorization"))
	})

	t.Run("admin key verification failure -> still ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"invalid admin key"}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "bad-admin-key",
			},
		}

		// The admin key is optional: a bad key must not block the integration.
		err := o.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
	})

	t.Run("custom base URL does not affect admin verification", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(usagePageBody)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "test-admin-key",
				"baseURL":  "https://ollama.internal/v1",
			},
		}

		err := o.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		// Model verification uses the custom base URL; org usage endpoints only
		// exist on the OpenAI platform API.
		assert.Contains(t, httpContext.Requests[0].URL.String(), "https://ollama.internal/v1/models")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "https://api.openai.com/v1/organization/usage/completions")
	})
}

func Test__OpenAI__ListResources__files(t *testing.T) {
	o := &OpenAI{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("lists files", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"object": "list",
						"data": [
							{"id": "file-1", "object": "file", "filename": "a.pdf", "purpose": "assistants", "bytes": 10, "created_at": 1707825600},
							{"id": "file-2", "object": "file", "filename": "b.csv", "purpose": "batch_output", "bytes": 20, "created_at": 1707825601}
						],
						"has_more": false
					}`)),
				},
			},
		}
		ctx := core.ListResourcesContext{
			Logger:      logger,
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
		}

		resources, err := o.ListResources("file", ctx)
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "file-1", resources[0].ID)
		assert.Equal(t, "a.pdf", resources[0].Name)
	})

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Logger:      logger,
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
		}
		resources, err := o.ListResources("container", ctx)
		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
