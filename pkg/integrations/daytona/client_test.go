package daytona

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NewClient(t *testing.T) {
	t.Run("missing apiKey -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "https://app.daytona.io/api",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, appCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiKey")
	})

	t.Run("successful client creation with defaults", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, appCtx)

		require.NoError(t, err)
		assert.Equal(t, "test-api-key", client.APIKey)
		assert.Equal(t, defaultBaseURL, client.BaseURL)
	})

	t.Run("successful client creation with custom baseURL", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":  "test-api-key",
				"baseURL": "https://custom.daytona.io/api",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, appCtx)

		require.NoError(t, err)
		assert.Equal(t, "test-api-key", client.APIKey)
		assert.Equal(t, "https://custom.daytona.io/api", client.BaseURL)
	})

	t.Run("nil app installation context -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no app installation context")
	})
}

func Test__Client__Verify(t *testing.T) {
	t.Run("successful verification", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		err = client.Verify()
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/sandbox")
		assert.Equal(t, "Bearer test-api-key", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("verification failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "invalid-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		err = client.Verify()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

func Test__Client__CreateSandbox(t *testing.T) {
	t.Run("successful sandbox creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"started"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		sandbox, err := client.CreateSandbox(&CreateSandboxRequest{
			Target: "us",
		})

		require.NoError(t, err)
		assert.Equal(t, "sandbox-123", sandbox.ID)
		assert.Equal(t, "started", sandbox.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/sandbox")
	})

	t.Run("sandbox creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"message":"invalid request"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		_, err = client.CreateSandbox(&CreateSandboxRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

func Test__Client__ExecuteCommand(t *testing.T) {
	t.Run("successful command execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"exitCode":0,"result":"hello world"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		response, err := client.ExecuteCommand("sandbox-123", &ExecuteCommandRequest{
			Command: "echo hello world",
		})

		require.NoError(t, err)
		assert.Equal(t, 0, response.ExitCode)
		assert.Equal(t, "hello world", response.Result)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/toolbox/sandbox-123/process/execute")
	})

	t.Run("command execution failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"message":"execution failed"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		_, err = client.ExecuteCommand("sandbox-123", &ExecuteCommandRequest{
			Command: "invalid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func Test__Client__ExecuteCode(t *testing.T) {
	t.Run("successful python code execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"exitCode":0,"result":"42"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		response, err := client.ExecuteCode("sandbox-123", &ExecuteCodeRequest{
			Code:     "print(42)",
			Language: "python",
		})

		require.NoError(t, err)
		assert.Equal(t, 0, response.ExitCode)
		assert.Equal(t, "42", response.Result)
	})

	t.Run("successful javascript code execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"exitCode":0,"result":"hello"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		response, err := client.ExecuteCode("sandbox-123", &ExecuteCodeRequest{
			Code:     "console.log('hello')",
			Language: "javascript",
		})

		require.NoError(t, err)
		assert.Equal(t, 0, response.ExitCode)
	})
}

func Test__Client__DeleteSandbox(t *testing.T) {
	t.Run("successful sandbox deletion", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		err = client.DeleteSandbox("sandbox-123", false)

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/sandbox/sandbox-123")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "force=false")
	})

	t.Run("force delete sandbox", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		err = client.DeleteSandbox("sandbox-123", true)

		require.NoError(t, err)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "force=true")
	})

	t.Run("sandbox deletion failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"sandbox not found"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		err = client.DeleteSandbox("invalid-id", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}
