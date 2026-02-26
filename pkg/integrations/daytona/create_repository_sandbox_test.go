package daytona

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateRepositorySandbox__Setup(t *testing.T) {
	component := CreateRepositorySandbox{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"bootstrap": map[string]any{
					"from":   SandboxBootstrapFromInline,
					"script": "npm ci",
				},
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("bootstrap is optional", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
			},
		})

		require.NoError(t, err)
	})

	t.Run("bootstrap from is required when bootstrap is provided", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap":  map[string]any{},
			},
		})

		require.ErrorContains(t, err, "bootstrap.from is required")
	})

	t.Run("inline bootstrap requires script", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap": map[string]any{
					"from": SandboxBootstrapFromInline,
				},
			},
		})

		require.ErrorContains(t, err, "bootstrap.script is required when bootstrap.from is inline")
	})

	t.Run("file bootstrap requires path", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap": map[string]any{
					"from": SandboxBootstrapFromFile,
				},
			},
		})

		require.ErrorContains(t, err, "bootstrap.path is required when bootstrap.from is file")
	})

	t.Run("invalid bootstrap from", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap": map[string]any{
					"from": "url",
				},
			},
		})

		require.ErrorContains(t, err, "invalid bootstrap.from")
	})

	t.Run("invalid env name", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap": map[string]any{
					"from":   SandboxBootstrapFromInline,
					"script": "npm ci",
				},
				"env": []map[string]any{
					{"name": "INVALID-NAME", "value": "1"},
				},
			},
		})

		require.ErrorContains(t, err, "invalid env variable name")
	})

	t.Run("valid inline bootstrap setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap": map[string]any{
					"from":   SandboxBootstrapFromInline,
					"script": "npm ci && npm test",
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid file bootstrap setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap": map[string]any{
					"from": SandboxBootstrapFromFile,
					"path": "scripts/bootstrap.sh",
				},
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateRepositorySandbox__Execute(t *testing.T) {
	component := CreateRepositorySandbox{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"creating"}`)),
			},
		},
	}

	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "test-api-key",
		},
	}

	metadataCtx := &contexts.MetadataContext{}
	requestCtx := &contexts.RequestContext{}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"repository": "https://github.com/superplanehq/superplane.git",
			"bootstrap": map[string]any{
				"from":   SandboxBootstrapFromInline,
				"script": "npm ci",
			},
		},
		HTTP:           httpContext,
		Integration:    appCtx,
		ExecutionState: execCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx,
	})

	require.NoError(t, err)
	assert.False(t, execCtx.Finished)
	assert.Equal(t, "poll", requestCtx.Action)
	assert.Equal(t, CreateRepositorySandboxPollInterval, requestCtx.Duration)

	metadata, ok := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
	require.True(t, ok)
	assert.Equal(t, repositorySandboxStageWaitingSandbox, metadata.Stage)
	assert.Equal(t, "sandbox-123", metadata.SandboxID)
	assert.Equal(t, "https://github.com/superplanehq/superplane.git", metadata.Repository)
	assert.Equal(t, "/home/daytona/superplane", metadata.Directory)
	require.NotNil(t, metadata.SandboxStartedAt)
	assert.Equal(t, int(CreateRepositorySandboxDefaultTimeout.Seconds()), metadata.Timeout)
	require.NotNil(t, metadata.Bootstrap)
	assert.Equal(t, SandboxBootstrapFromInline, metadata.Bootstrap.From)
	require.NotNil(t, metadata.Bootstrap.Script)
	assert.Equal(t, "npm ci", *metadata.Bootstrap.Script)
}

func Test__CreateRepositorySandbox__HandleAction(t *testing.T) {
	component := CreateRepositorySandbox{}

	t.Run("waits while sandbox is creating", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageWaitingSandbox,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"creating"}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       requestCtx,
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("starts clone when sandbox is ready", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageWaitingSandbox,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				Repository:       "https://github.com/superplanehq/superplane.git",
				Directory:        "/home/daytona/superplane",
				Bootstrap: &BootstrapMetadata{
					From:   SandboxBootstrapFromInline,
					Script: ptr("npm ci"),
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetSandbox
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"started"}`))},
				// FetchConfig for CreateSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// CreateSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				// FetchConfig for ExecuteSessionCommand
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// ExecuteSessionCommand clone
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cmdId":"cmd-clone"}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       requestCtx,
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)

		updated, ok := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.True(t, ok)
		assert.Equal(t, repositorySandboxStageCloningRepo, updated.Stage)
		assert.NotEmpty(t, updated.SessionID)
		require.NotNil(t, updated.Clone)
		assert.Equal(t, "cmd-clone", updated.Clone.CmdID)
		assert.NotEmpty(t, updated.Clone.StartedAt)

		require.Len(t, httpContext.Requests, 5)
		body, err := io.ReadAll(httpContext.Requests[4].Body)
		require.NoError(t, err)

		req := SessionExecuteRequest{}
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Contains(t, req.Command, "git clone --depth 1")
		assert.Contains(t, req.Command, "'https://github.com/superplanehq/superplane.git'")
		assert.Contains(t, req.Command, "'/home/daytona/superplane'")
	})

	t.Run("clone stage with command not finished reschedules", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageCloningRepo,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Clone: &CloneMetadata{
					CmdID: "cmd-clone",
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-clone"}]}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       requestCtx,
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("clone stage failure returns error", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageCloningRepo,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Clone: &CloneMetadata{
					CmdID: "cmd-clone",
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-clone","exitCode":1}]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`fatal: repository not found`))},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.ErrorContains(t, err, "repository clone failed")

		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.NotNil(t, updated.Clone)
		assert.Equal(t, 1, updated.Clone.ExitCode)
		assert.Equal(t, "fatal: repository not found", updated.Clone.Result)
		assert.NotEmpty(t, updated.Clone.FinishedAt)
	})

	t.Run("clone stage success starts bootstrap", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageCloningRepo,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Repository:       "https://github.com/superplanehq/superplane.git",
				Directory:        "/home/daytona/superplane",
				Clone: &CloneMetadata{
					CmdID: "cmd-clone",
				},
				Bootstrap: &BootstrapMetadata{
					From:   SandboxBootstrapFromInline,
					Script: ptr("npm ci && npm test"),
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-clone","exitCode":0}]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`clone logs`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cmdId":"cmd-bootstrap"}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       requestCtx,
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)

		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		assert.Equal(t, repositorySandboxStageBootstrapping, updated.Stage)
		require.NotNil(t, updated.Clone)
		assert.Equal(t, "clone logs", updated.Clone.Result)
		require.NotNil(t, updated.Bootstrap)
		assert.Equal(t, "cmd-bootstrap", updated.Bootstrap.CmdID)
		assert.NotEmpty(t, updated.Bootstrap.StartedAt)

		require.Len(t, httpContext.Requests, 6)
		body, err := io.ReadAll(httpContext.Requests[5].Body)
		require.NoError(t, err)
		req := SessionExecuteRequest{}
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Contains(t, req.Command, "cd '/home/daytona/superplane' && npm ci && npm test")
	})

	t.Run("bootstrap stage success emits payload", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageBootstrapping,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Repository:       "https://github.com/superplanehq/superplane.git",
				Directory:        "/home/daytona/superplane",
				Clone: &CloneMetadata{
					CmdID:     "cmd-clone",
					ExitCode:  0,
					Result:    "clone logs",
					StartedAt: time.Now().Format(time.RFC3339),
				},
				Bootstrap: &BootstrapMetadata{
					CmdID: "cmd-bootstrap",
					From:  SandboxBootstrapFromInline,
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":0}]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`bootstrap logs`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: execCtx,
			Requests:       &contexts.RequestContext{},
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, CreateRepositorySandboxPayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)

		wrapped, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(CreateRepositorySandboxMetadata)
		require.True(t, ok)
		assert.Equal(t, "sandbox-123", payload.SandboxID)
		assert.Equal(t, "/home/daytona/superplane", payload.Directory)
		assert.Equal(t, "clone logs", payload.Clone.Result)
		assert.Equal(t, "bootstrap logs", payload.Bootstrap.Result)
		assert.Equal(t, 0, payload.Bootstrap.ExitCode)
	})

	t.Run("bootstrap stage failure returns error", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageBootstrapping,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Bootstrap: &BootstrapMetadata{
					CmdID: "cmd-bootstrap",
					From:  SandboxBootstrapFromInline,
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":2}]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`npm ERR!`))},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.ErrorContains(t, err, "bootstrap script failed")

		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.NotNil(t, updated.Bootstrap)
		assert.Equal(t, 2, updated.Bootstrap.ExitCode)
		assert.Equal(t, "npm ERR!", updated.Bootstrap.Result)
	})

	t.Run("times out when sandbox startup exceeded timeout", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: CreateRepositorySandboxMetadata{
					Stage:            repositorySandboxStageWaitingSandbox,
					SandboxID:        "sandbox-123",
					SandboxStartedAt: time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
					Timeout:          int(time.Minute.Seconds()),
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.ErrorContains(t, err, "timed out")
	})

	t.Run("unknown action returns error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{Name: "unknown"})
		require.ErrorContains(t, err, "unknown action")
	})
}

func Test__CreateRepositorySandbox__GetDirectoryName(t *testing.T) {
	component := CreateRepositorySandbox{}

	t.Run("https repository", func(t *testing.T) {
		directory, err := component.getDirectoryName("https://github.com/superplanehq/superplane.git")
		require.NoError(t, err)
		assert.Equal(t, "superplane", directory)
	})

	t.Run("ssh repository", func(t *testing.T) {
		directory, err := component.getDirectoryName("git@github.com:superplanehq/superplane.git")
		require.NoError(t, err)
		assert.Equal(t, "superplane", directory)
	})

	t.Run("invalid repository", func(t *testing.T) {
		_, err := component.getDirectoryName("https://github.com")
		require.Error(t, err)
	})
}

func Test__CreateRepositorySandbox__BootstrapCommand(t *testing.T) {
	component := CreateRepositorySandbox{}

	t.Run("inline mode", func(t *testing.T) {
		command, err := component.bootstrapCommand(&CreateRepositorySandboxMetadata{
			Directory: "/home/daytona/repository",
			Bootstrap: &BootstrapMetadata{
				From:   SandboxBootstrapFromInline,
				Script: ptr("npm ci && npm test"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "cd '/home/daytona/repository' && npm ci && npm test", command)
	})

	t.Run("file mode", func(t *testing.T) {
		command, err := component.bootstrapCommand(&CreateRepositorySandboxMetadata{
			Directory: "/home/daytona/repository",
			Bootstrap: &BootstrapMetadata{
				From: SandboxBootstrapFromFile,
				Path: ptr("scripts/bootstrap.sh"),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "cd '/home/daytona/repository' && sh 'scripts/bootstrap.sh'", command)
	})

	t.Run("invalid from", func(t *testing.T) {
		_, err := component.bootstrapCommand(&CreateRepositorySandboxMetadata{
			Directory: "/home/daytona/repository",
			Bootstrap: &BootstrapMetadata{
				From: "url",
			},
		})

		require.ErrorContains(t, err, "invalid bootstrap.from")
	})
}

func newTestLogger() *log.Entry {
	return log.NewEntry(log.New())
}

func ptr(value string) *string {
	return &value
}
