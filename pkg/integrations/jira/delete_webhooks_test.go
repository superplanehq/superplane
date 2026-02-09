package jira

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

func Test__DeleteWebhooks__Execute(t *testing.T) {
	component := DeleteWebhooks{}

	t.Run("delete by ID -> success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		webhookID := int64(123)
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"webhookId": &webhookID,
				"deleteAll": false,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "jira.webhookDeleted", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("delete all -> success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// First call: ListWebhooks
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"startAt": 0,
						"maxResults": 50,
						"total": 2,
						"values": [{"id": 1}, {"id": 2}]
					}`)),
				},
				// Second call: DeleteWebhook
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"deleteAll": true,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "jira.webhookDeleted", execCtx.Type)
		// List + Delete
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
	})

	t.Run("neither webhookId nor deleteAll -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"deleteAll": false,
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.ErrorContains(t, err, "either webhookId or deleteAll must be specified")
		assert.False(t, execCtx.Finished)
	})

	t.Run("delete by ID -> API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Webhook not found"]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		webhookID := int64(999)
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"webhookId": &webhookID,
				"deleteAll": false,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.False(t, execCtx.Finished)
	})
}

func Test__DeleteWebhooks__ComponentInfo(t *testing.T) {
	component := DeleteWebhooks{}

	assert.Equal(t, "jira.deleteWebhooks", component.Name())
	assert.Equal(t, "Delete Webhooks", component.Label())
	assert.Equal(t, "jira", component.Icon())
	assert.Equal(t, "blue", component.Color())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
}

func Test__DeleteWebhooks__Configuration(t *testing.T) {
	component := DeleteWebhooks{}
	config := component.Configuration()
	require.Len(t, config, 2)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "webhookId")
	assert.Contains(t, fieldNames, "deleteAll")

	for _, f := range config {
		assert.False(t, f.Required, "%s should not be required", f.Name)
	}
}

func Test__DeleteWebhooks__OutputChannels(t *testing.T) {
	component := DeleteWebhooks{}
	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}
