package dataforseo

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

func mockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func Test__Client__Verify(t *testing.T) {
	t.Run("valid credentials", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "1", "result": [{"login": "user@example.com"}]}]
				}`),
			},
		}
		client, err := NewClient(httpCtx, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		})
		require.NoError(t, err)

		err = client.Verify()
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "Basic dXNlcjpwYXNz", httpCtx.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, "https://api.dataforseo.com/v3/appendix/user_data", httpCtx.Requests[0].URL.String())
	})

	t.Run("invalid credentials", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{mockResponse(http.StatusUnauthorized, `{"status_code": 40100}`)},
		}
		client, err := NewClient(httpCtx, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "bad"},
		})
		require.NoError(t, err)

		err = client.Verify()
		require.Error(t, err)
	})
}

func Test__Client__PostAudit(t *testing.T) {
	t.Run("task created", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "task-1", "status_code": 20100, "status_message": "Task Created."}]
				}`),
			},
		}
		client, err := NewClient(httpCtx, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		})
		require.NoError(t, err)

		taskID, err := client.PostAudit("freehire.me", 100)
		require.NoError(t, err)
		assert.Equal(t, "task-1", taskID)
	})

	t.Run("rejected by DataForSEO - bad domain", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "task-1", "status_code": 40501, "status_message": "Invalid Field: 'target'."}]
				}`),
			},
		}
		client, err := NewClient(httpCtx, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		})
		require.NoError(t, err)

		taskID, err := client.PostAudit("not a domain", 100)
		require.Error(t, err)
		assert.Empty(t, taskID)
		assert.Contains(t, err.Error(), "Invalid Field")
	})

	t.Run("rejected by DataForSEO - top level status_code", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mockResponse(http.StatusOK, `{
					"status_code": 40200,
					"status_message": "Access denied.",
					"tasks": []
				}`),
			},
		}
		client, err := NewClient(httpCtx, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		})
		require.NoError(t, err)

		taskID, err := client.PostAudit("freehire.me", 100)
		require.Error(t, err)
		assert.Empty(t, taskID)
		assert.Contains(t, err.Error(), "Access denied")
	})
}

func Test__DataForSEO__Sync(t *testing.T) {
	d := &DataForSEO{}

	t.Run("missing apiKey", func(t *testing.T) {
		ctx := core.SyncContext{
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{},
		}
		err := d.Sync(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiKey is required")
	})

	t.Run("valid credentials", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		}
		ctx := core.SyncContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
			Integration:   integrationCtx,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					mockResponse(http.StatusOK, `{"status_code": 20000, "tasks": [{"id": "1"}]}`),
				},
			},
		}
		err := d.Sync(ctx)
		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
	})
}
