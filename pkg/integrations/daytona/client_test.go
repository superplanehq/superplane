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

func Test__Client__CreateSession(t *testing.T) {
	t.Run("successful session creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"sessionId":"session-1"}`)),
				},
			},
		}

		client := newTestClient(t, httpContext)
		err := client.CreateSession("sandbox-123", "session-1")

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/toolbox/sandbox-123/process/session")
	})
}

func Test__Client__ExecuteSessionCommand(t *testing.T) {
	t.Run("successful async execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"cmdId":"cmd-abc"}`)),
				},
			},
		}

		client := newTestClient(t, httpContext)
		resp, err := client.ExecuteSessionCommand("sandbox-123", "session-1", "echo hello")

		require.NoError(t, err)
		assert.Equal(t, "cmd-abc", resp.CmdID)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/process/session/session-1/exec")
	})
}

func Test__Client__GetSession(t *testing.T) {
	t.Run("command still running", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"cmdId":"cmd-abc","command":"echo hello","exitCode":null}]}`)),
				},
			},
		}

		client := newTestClient(t, httpContext)
		session, err := client.GetSession("sandbox-123", "session-1")

		require.NoError(t, err)
		assert.Equal(t, "session-1", session.SessionID)
		require.Len(t, session.Commands, 1)
		assert.Nil(t, session.Commands[0].ExitCode)
	})

	t.Run("command completed", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"cmdId":"cmd-abc","command":"echo hello","exitCode":0}]}`)),
				},
			},
		}

		client := newTestClient(t, httpContext)
		session, err := client.GetSession("sandbox-123", "session-1")

		require.NoError(t, err)
		cmd := session.FindCommand("cmd-abc")
		require.NotNil(t, cmd)
		require.NotNil(t, cmd.ExitCode)
		assert.Equal(t, 0, *cmd.ExitCode)
	})
}

func Test__Client__GetSessionCommandLogs(t *testing.T) {
	t.Run("successful log retrieval", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`"hello world"`)),
				},
			},
		}

		client := newTestClient(t, httpContext)
		logs, err := client.GetSessionCommandLogs("sandbox-123", "session-1", "cmd-abc")

		require.NoError(t, err)
		assert.Contains(t, logs, "hello world")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/process/session/session-1/command/cmd-abc/logs")
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

		client := newTestClient(t, httpContext)
		err := client.DeleteSandbox("sandbox-123", false)

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

		client := newTestClient(t, httpContext)
		err := client.DeleteSandbox("sandbox-123", true)

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

		client := newTestClient(t, httpContext)
		err := client.DeleteSandbox("invalid-id", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

func Test__Session__FindCommand(t *testing.T) {
	session := &Session{
		Commands: []SessionCommand{
			{CmdID: "cmd-1", Command: "echo a"},
			{CmdID: "cmd-2", Command: "echo b"},
		},
	}

	t.Run("existing command", func(t *testing.T) {
		cmd := session.FindCommand("cmd-2")
		require.NotNil(t, cmd)
		assert.Equal(t, "echo b", cmd.Command)
	})

	t.Run("missing command", func(t *testing.T) {
		cmd := session.FindCommand("cmd-999")
		assert.Nil(t, cmd)
	})
}

func configResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`)),
	}
}

func newTestClient(t *testing.T, httpContext *contexts.HTTPContext) *Client {
	t.Helper()
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "test-api-key",
		},
	}

	client, err := NewClient(httpContext, appCtx)
	require.NoError(t, err)
	return client
}
