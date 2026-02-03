package dockerhub

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

func Test__ListTags__Setup(t *testing.T) {
	component := &ListTags{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("missing repository -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "library/nginx",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with optional fields -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "myorg/myapp",
				"pageSize":   25,
				"nameFilter": "v1.*",
			},
		})

		require.NoError(t, err)
	})
}

func Test__ListTags__Execute(t *testing.T) {
	component := &ListTags{}

	t.Run("successful tag listing", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Login response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"token": "test-jwt-token"}`)),
				},
				// List tags response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"count": 2,
						"results": [
							{
								"name": "latest",
								"last_updated": "2026-02-01T10:30:00.000000Z",
								"full_size": 142857600,
								"digest": "sha256:abc123..."
							},
							{
								"name": "v1.2.3",
								"last_updated": "2026-01-28T15:45:00.000000Z",
								"full_size": 140000000,
								"digest": "sha256:def456..."
							}
						]
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"username":    "testuser",
				"accessToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "library/nginx",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "dockerhub.tags", executionState.Type)

		require.Len(t, httpContext.Requests, 2)

		// First request should be login
		loginReq := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, loginReq.Method)
		assert.Contains(t, loginReq.URL.String(), "users/login")

		// Second request should be list tags
		listReq := httpContext.Requests[1]
		assert.Equal(t, http.MethodGet, listReq.Method)
		assert.Contains(t, listReq.URL.String(), "/repositories/library/nginx/tags")
		assert.Equal(t, "Bearer test-jwt-token", listReq.Header.Get("Authorization"))
	})

	t.Run("successful tag listing with filters", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Login response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"token": "test-jwt-token"}`)),
				},
				// List tags response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"count": 1,
						"results": [
							{
								"name": "v1.2.3",
								"last_updated": "2026-01-28T15:45:00.000000Z",
								"full_size": 140000000
							}
						]
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"username":    "testuser",
				"accessToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "myorg/myapp",
				"pageSize":   10,
				"nameFilter": "v1",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		// Check query parameters
		listReq := httpContext.Requests[1]
		assert.Contains(t, listReq.URL.String(), "page_size=10")
		assert.Contains(t, listReq.URL.String(), "name=v1")
	})

	t.Run("API error -> execution fails", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Login response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"token": "test-jwt-token"}`)),
				},
				// List tags error response
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "repository not found"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"username":    "testuser",
				"accessToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "nonexistent/repo",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err) // Component handles error gracefully
		assert.False(t, executionState.Passed)
	})
}
