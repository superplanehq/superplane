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

func Test__DescribeImageTag__Setup(t *testing.T) {
	component := &DescribeImageTag{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("missing namespace -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":  "",
				"repository": "myapp",
				"tag":        "latest",
			},
		})

		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("missing repository -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "",
				"tag":        "latest",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing tag -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "myapp",
				"tag":        "",
			},
		})

		require.ErrorContains(t, err, "tag is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":  "library",
				"repository": "nginx",
				"tag":        "latest",
			},
		})

		require.NoError(t, err)
	})
}

func Test__DescribeImageTag__Execute(t *testing.T) {
	component := &DescribeImageTag{}

	t.Run("successful tag description", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Login response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"token": "test-jwt-token"}`)),
				},
				// Get tag response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"name": "latest",
						"last_updated": "2026-02-01T10:30:00.000000Z",
						"full_size": 142857600,
						"digest": "sha256:abc123...",
						"media_type": "application/vnd.docker.distribution.manifest.v2+json",
						"images": [
							{
								"architecture": "amd64",
								"os": "linux",
								"size": 70000000
							},
							{
								"architecture": "arm64",
								"os": "linux",
								"size": 72000000
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
				"namespace":  "library",
				"repository": "nginx",
				"tag":        "latest",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "dockerhub.tag", executionState.Type)

		require.Len(t, httpContext.Requests, 2)

		// First request should be login
		loginReq := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, loginReq.Method)
		assert.Contains(t, loginReq.URL.String(), "users/login")

		// Second request should be get tag
		getTagReq := httpContext.Requests[1]
		assert.Equal(t, http.MethodGet, getTagReq.Method)
		assert.Contains(t, getTagReq.URL.String(), "/repositories/library/nginx/tags/latest")
		assert.Equal(t, "Bearer test-jwt-token", getTagReq.Header.Get("Authorization"))
	})

	t.Run("tag not found -> execution fails", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Login response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"token": "test-jwt-token"}`)),
				},
				// Get tag error response
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "tag not found"}`)),
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
				"namespace":  "myorg",
				"repository": "myapp",
				"tag":        "nonexistent",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err) // Component handles error gracefully
		assert.False(t, executionState.Passed)
		assert.Equal(t, "not_found", executionState.FailureReason)
	})
}
