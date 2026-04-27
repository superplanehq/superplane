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
	"github.com/superplanehq/superplane/pkg/configuration"
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

	t.Run("invalid secret type", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"secrets": []map[string]any{
					{
						"type": "invalid",
						"value": map[string]any{
							"secret": "credentials",
							"key":    "token",
						},
					},
				},
			},
		})

		require.ErrorContains(t, err, "invalid secret type")
	})

	t.Run("negative bootstrap timeout is rejected", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap": map[string]any{
					"from":    SandboxBootstrapFromInline,
					"script":  "npm ci",
					"timeout": -1,
				},
			},
		})

		require.ErrorContains(t, err, "bootstrap.timeout cannot be negative")
	})

	t.Run("bootstrap timeout above the ceiling is rejected", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "https://github.com/superplanehq/superplane.git",
				"bootstrap": map[string]any{
					"from":    SandboxBootstrapFromInline,
					"script":  "npm ci",
					"timeout": int(CreateRepositorySandboxMaxTimeout.Minutes()) + 1,
				},
			},
		})

		require.ErrorContains(t, err, "bootstrap.timeout cannot exceed")
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
		Logger:         newTestLogger(),
	})

	require.NoError(t, err)
	assert.False(t, execCtx.Finished)
	assert.Equal(t, "poll", requestCtx.Action)
	assert.Equal(t, CreateRepositorySandboxPollInterval, requestCtx.Duration)

	metadata, ok := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
	require.True(t, ok)
	assert.Equal(t, repositorySandboxStagePreparingSandbox, metadata.Stage)
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

func Test__CreateRepositorySandbox__Execute_WithConfiguredBootstrapTimeout(t *testing.T) {
	component := CreateRepositorySandbox{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"creating"}`)),
			},
		},
	}

	metadataCtx := &contexts.MetadataContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"repository": "https://github.com/superplanehq/superplane.git",
			"bootstrap": map[string]any{
				"from":    SandboxBootstrapFromInline,
				"script":  "npm ci",
				"timeout": 2,
			},
		},
		HTTP:           httpContext,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-api-key"}},
		ExecutionState: &contexts.ExecutionStateContext{},
		Metadata:       metadataCtx,
		Requests:       &contexts.RequestContext{},
		Logger:         newTestLogger(),
	})

	require.NoError(t, err)
	metadata := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
	assert.Equal(t, 2*60, metadata.Timeout, "bootstrap timeout in minutes should be converted to seconds")
}

func Test__CreateRepositorySandbox__HandleAction(t *testing.T) {
	component := CreateRepositorySandbox{}

	t.Run("waits while sandbox is creating", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStagePreparingSandbox,
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
		err := component.HandleHook(core.ActionHookContext{
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
				Stage:            repositorySandboxStagePreparingSandbox,
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
				// FetchConfig for CloneRepository
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// CloneRepository
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				// FetchConfig for bootstrap folder creation
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// CreateFolder /home/daytona/.superplane
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				// FetchConfig for inline bootstrap upload
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// Upload inline bootstrap script
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				// FetchConfig for CreateSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// CreateSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				// FetchConfig for ExecuteSessionCommand
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// ExecuteSessionCommand bootstrap
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cmdId":"cmd-bootstrap"}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleHook(core.ActionHookContext{
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
		assert.Equal(t, repositorySandboxStageBootstrapping, updated.Stage)
		assert.NotEmpty(t, updated.SessionID)
		require.NotNil(t, updated.Clone)
		assert.Nil(t, updated.Clone.Error)
		assert.NotEmpty(t, updated.Clone.StartedAt)
		assert.NotEmpty(t, updated.Clone.FinishedAt)
		require.NotNil(t, updated.Bootstrap)
		assert.Equal(t, "cmd-bootstrap", updated.Bootstrap.CmdID)
		assert.NotEmpty(t, updated.Bootstrap.StartedAt)

		require.Len(t, httpContext.Requests, 11)
		cloneBody, err := io.ReadAll(httpContext.Requests[2].Body)
		require.NoError(t, err)
		cloneReq := CloneRepositoryRequest{}
		require.NoError(t, json.Unmarshal(cloneBody, &cloneReq))
		assert.Equal(t, "https://github.com/superplanehq/superplane.git", cloneReq.URL)
		assert.Equal(t, "/home/daytona/superplane", cloneReq.Path)

		uploadedScriptBody, err := io.ReadAll(httpContext.Requests[6].Body)
		require.NoError(t, err)
		assert.Contains(t, string(uploadedScriptBody), "npm ci")

		body, err := io.ReadAll(httpContext.Requests[10].Body)
		require.NoError(t, err)
		req := SessionExecuteRequest{}
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Contains(t, req.Command, "cd '/home/daytona/superplane' && sh '/home/daytona/.superplane/bootstrap.sh'")
	})

	t.Run("inline script with CRLF line endings is normalized to LF before upload", func(t *testing.T) {
		// The configuration form sometimes produces CRLF line endings; dash
		// interprets the trailing \r literally, which breaks `sleep 2` and
		// keywords like `done`. The component must strip them before upload.
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStagePreparingSandbox,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				Repository:       "https://github.com/superplanehq/superplane.git",
				Directory:        "/home/daytona/superplane",
				Bootstrap: &BootstrapMetadata{
					From:   SandboxBootstrapFromInline,
					Script: ptr("echo line1\r\nsleep 2\r\necho line2\r\n"),
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"started"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cmdId":"cmd-bootstrap"}`))},
			},
		}

		err := component.HandleHook(core.ActionHookContext{
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

		require.NoError(t, err)

		uploadedScriptBody, err := io.ReadAll(httpContext.Requests[6].Body)
		require.NoError(t, err)
		// The multipart envelope itself uses CRLF; we only care that the
		// script content is LF-only. Asserting the exact LF substring is
		// sufficient because the original input had \r\n line endings.
		assert.Contains(t, string(uploadedScriptBody), "echo line1\nsleep 2\necho line2\n")
		assert.NotContains(t, string(uploadedScriptBody), "echo line1\r", "carriage returns inside the script must be stripped before upload")
	})

	t.Run("clone failure marks execution as failed", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStagePreparingSandbox,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				Repository:       "https://github.com/superplanehq/private-repo.git",
				Directory:        "/home/daytona/private-repo",
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetSandbox
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"started"}`))},
				// FetchConfig for CloneRepository
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// CloneRepository error
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"authentication failed"}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
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
		assert.False(t, execCtx.Passed)
		assert.Equal(t, "error", execCtx.FailureReason)
		assert.Contains(t, execCtx.FailureMessage, "repository clone failed")

		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.NotNil(t, updated.Clone)
		assert.NotNil(t, updated.Clone.Error)
		assert.Contains(t, *updated.Clone.Error, "authentication failed")
		assert.NotEmpty(t, updated.Clone.StartedAt)
		assert.NotEmpty(t, updated.Clone.FinishedAt)
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
					StartedAt:  time.Now().Format(time.RFC3339),
					FinishedAt: time.Now().Format(time.RFC3339),
				},
				Bootstrap: &BootstrapMetadata{
					CmdID:     "cmd-bootstrap",
					From:      SandboxBootstrapFromInline,
					StartedAt: time.Now().Format(time.RFC3339),
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// FetchConfig for GetSessionCommandLogs (logs fetched first so UI gets early output)
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`bootstrap logs partial`))},
				// FetchConfig for GetSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":0}]}`))},
				// FetchConfig for the final log re-fetch (after ExitCode is known)
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`bootstrap logs complete`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
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
		require.NotNil(t, payload.Clone)
		assert.Nil(t, payload.Clone.Error)
		assert.Equal(t, "bootstrap logs complete", payload.Bootstrap.Result, "final Result should reflect the post-exit log re-fetch, not the earlier partial snapshot")
		assert.Equal(t, "bootstrap logs complete", payload.Bootstrap.Log)
		assert.Equal(t, 0, payload.Bootstrap.ExitCode)
	})

	t.Run("bootstrap stage failure fails execution", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageBootstrapping,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Bootstrap: &BootstrapMetadata{
					CmdID:     "cmd-bootstrap",
					From:      SandboxBootstrapFromInline,
					StartedAt: time.Now().Format(time.RFC3339),
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// FetchConfig for GetSessionCommandLogs
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`npm ERR!`))},
				// FetchConfig for GetSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":2}]}`))},
				// FetchConfig for the final log re-fetch (after ExitCode is known)
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`npm ERR!`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
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
		assert.False(t, execCtx.Passed)
		assert.Equal(t, "error", execCtx.FailureReason)
		assert.Contains(t, execCtx.FailureMessage, "bootstrap script failed with exit code 2: npm ERR!")
	})

	t.Run("times out when sandbox startup exceeded the hardcoded startup timeout", func(t *testing.T) {
		// Sandbox creation + clone is bounded by a fixed startup window;
		// the user-configured bootstrap timeout does not apply yet.
		execCtx := &contexts.ExecutionStateContext{}

		err := component.HandleHook(core.ActionHookContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: CreateRepositorySandboxMetadata{
					Stage:            repositorySandboxStagePreparingSandbox,
					SandboxID:        "sandbox-123",
					SandboxStartedAt: time.Now().Add(-(CreateRepositorySandboxStartupTimeout + time.Minute)).Format(time.RFC3339),
					Timeout:          int(time.Hour.Seconds()), // user-configured bootstrap timeout — must be ignored here
				},
			},
			ExecutionState: execCtx,
			Requests:       &contexts.RequestContext{},
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.False(t, execCtx.Passed)
		assert.Equal(t, "error", execCtx.FailureReason)
		assert.Contains(t, execCtx.FailureMessage, "sandbox startup failed on stage preparingSandbox")
	})

	t.Run("does not apply bootstrap timeout during sandbox startup", func(t *testing.T) {
		// A short bootstrap timeout must not abort an in-progress sandbox
		// startup, since bootstrap has not begun yet.
		execCtx := &contexts.ExecutionStateContext{}
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"creating"}`))},
			},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name: "poll",
			HTTP: httpContext,
			Metadata: &contexts.MetadataContext{
				Metadata: CreateRepositorySandboxMetadata{
					Stage:            repositorySandboxStagePreparingSandbox,
					SandboxID:        "sandbox-123",
					SandboxStartedAt: time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
					Timeout:          int(time.Minute.Seconds()),
				},
			},
			ExecutionState: execCtx,
			Requests:       &contexts.RequestContext{},
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.False(t, execCtx.Finished, "user bootstrap timeout must not apply during sandbox startup phase")
	})

	t.Run("times out during bootstrap stage and marks execution as failed", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage: repositorySandboxStageBootstrapping,
				// SandboxStartedAt long ago; must not influence the bootstrap deadline.
				SandboxStartedAt: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
				Timeout:          int(time.Minute.Seconds()),
				SessionID:        "session-1",
				SandboxID:        "sandbox-123",
				Bootstrap: &BootstrapMetadata{
					CmdID:     "cmd-bootstrap",
					StartedAt: time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
				},
			},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
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
		assert.False(t, execCtx.Passed)
		assert.Equal(t, "error", execCtx.FailureReason)
		assert.Contains(t, execCtx.FailureMessage, "bootstrap failed after 1m0s")

		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.NotNil(t, updated.Bootstrap)
		assert.NotEmpty(t, updated.Bootstrap.FinishedAt, "bootstrap FinishedAt should be set on timeout so the UI has a terminal timestamp")
	})

	t.Run("captures logs while bootstrap is still running", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageBootstrapping,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Bootstrap: &BootstrapMetadata{
					CmdID:     "cmd-bootstrap",
					From:      SandboxBootstrapFromInline,
					StartedAt: time.Now().Format(time.RFC3339),
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// FetchConfig for GetSessionCommandLogs
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("installing deps...\nrunning tests..."))},
				// FetchConfig for GetSession (ExitCode still nil => still running)
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":null}]}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: execCtx,
			Requests:       requestCtx,
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.False(t, execCtx.Finished, "execution should still be running while bootstrap command is in flight")
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, CreateRepositorySandboxBootstrapPollInterval, requestCtx.Duration)

		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.NotNil(t, updated.Bootstrap)
		assert.Equal(t, "installing deps...\nrunning tests...", updated.Bootstrap.Log)
		assert.Empty(t, updated.Bootstrap.FinishedAt, "FinishedAt should not be set while the command is still running")
	})

	t.Run("bootstrap logs are tail-trimmed when exceeding the cap", func(t *testing.T) {
		big := strings.Repeat("x", CreateRepositorySandboxBootstrapLogMaxBytes+1024)

		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageBootstrapping,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Bootstrap: &BootstrapMetadata{
					CmdID:     "cmd-bootstrap",
					From:      SandboxBootstrapFromInline,
					StartedAt: time.Now().Format(time.RFC3339),
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(big))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":null}]}`))},
			},
		}

		err := component.HandleHook(core.ActionHookContext{
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

		require.NoError(t, err)
		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.NotNil(t, updated.Bootstrap)
		assert.Contains(t, updated.Bootstrap.Log, "[truncated")
		assert.LessOrEqual(t, len(updated.Bootstrap.Log), CreateRepositorySandboxBootstrapLogMaxBytes, "tail-trimmed log must respect the cap as a hard upper bound (marker included)")
	})

	t.Run("re-fetches logs after command exits so trailing output is captured", func(t *testing.T) {
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
					StartedAt:  time.Now().Format(time.RFC3339),
					FinishedAt: time.Now().Format(time.RFC3339),
				},
				Bootstrap: &BootstrapMetadata{
					CmdID:     "cmd-bootstrap",
					From:      SandboxBootstrapFromInline,
					StartedAt: time.Now().Format(time.RFC3339),
				},
			},
		}

		// First log fetch happens before the command has produced its final line;
		// between that fetch and the session-status check, the command exits and
		// emits "done". Without a post-exit re-fetch, "done" would be dropped.
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("step 10\n"))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":0}]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("step 10\ndone\n"))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
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
		assert.True(t, execCtx.Passed)
		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.NotNil(t, updated.Bootstrap)
		assert.Contains(t, updated.Bootstrap.Log, "done", "final log must include output produced between the in-flight fetch and the exit-code check")
	})

	t.Run("terminal poll preserves prior logs when both log fetches fail", func(t *testing.T) {
		// A previous poll captured "earlier output" into Bootstrap.Log /
		// Bootstrap.Result. The terminal poll (where ExitCode appears)
		// must not erase that snapshot if its own log fetches both fail.
		priorLog := "earlier output captured by a previous poll\n"
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
					StartedAt:  time.Now().Format(time.RFC3339),
					FinishedAt: time.Now().Format(time.RFC3339),
				},
				Bootstrap: &BootstrapMetadata{
					CmdID:     "cmd-bootstrap",
					From:      SandboxBootstrapFromInline,
					StartedAt: time.Now().Format(time.RFC3339),
					Log:       priorLog,
					Result:    priorLog,
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// First log fetch fails (5xx).
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"unavailable"}`))},
				// GetSession succeeds and reports the command as exited.
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":0}]}`))},
				// Final post-exit log re-fetch also fails.
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"unavailable"}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
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

		updated := metadataCtx.Metadata.(CreateRepositorySandboxMetadata)
		require.NotNil(t, updated.Bootstrap)
		assert.Equal(t, priorLog, updated.Bootstrap.Log, "prior log snapshot must be preserved when both terminal log fetches fail")
		assert.Equal(t, priorLog, updated.Bootstrap.Result, "prior Result must be preserved when both terminal log fetches fail")
		assert.Equal(t, 0, updated.Bootstrap.ExitCode)
	})

	t.Run("transient log fetch failure does not fail execution", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: CreateRepositorySandboxMetadata{
				Stage:            repositorySandboxStageBootstrapping,
				SandboxID:        "sandbox-123",
				SandboxStartedAt: time.Now().Format(time.RFC3339),
				Timeout:          int(5 * time.Minute.Seconds()),
				SessionID:        "session-1",
				Bootstrap: &BootstrapMetadata{
					CmdID:     "cmd-bootstrap",
					From:      SandboxBootstrapFromInline,
					StartedAt: time.Now().Format(time.RFC3339),
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// FetchConfig for GetSessionCommandLogs
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// Transient 5xx on log fetch
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"upstream unavailable"}`))},
				// FetchConfig for GetSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-1","commands":[{"id":"cmd-bootstrap","exitCode":null}]}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			HTTP:           httpContext,
			Metadata:       metadataCtx,
			ExecutionState: execCtx,
			Requests:       requestCtx,
			Logger:         newTestLogger(),
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
		})

		require.NoError(t, err)
		assert.False(t, execCtx.Finished)
		assert.Equal(t, "poll", requestCtx.Action, "a transient log-fetch failure must not abort the bootstrap poll")
	})

	t.Run("unknown hook returns error", func(t *testing.T) {
		err := component.HandleHook(core.ActionHookContext{Name: "unknown"})
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

func Test__CreateRepositorySandbox__CloneRepositoryRequest(t *testing.T) {
	component := CreateRepositorySandbox{}

	t.Run("includes github token from secrets", func(t *testing.T) {
		request, err := component.cloneRepositoryRequest(
			&contexts.SecretsContext{
				Values: map[string][]byte{
					"credentials/token": []byte("ghp_test_token"),
				},
			},
			&CreateRepositorySandboxMetadata{
				Repository: "https://github.com/superplanehq/superplane.git",
				Directory:  "/home/daytona/superplane",
				Secrets: []SandboxSecret{
					{
						Type: SandboxSecretTypeEnvVar,
						Name: "GITHUB_TOKEN",
						Value: configuration.SecretKeyRef{
							Secret: "credentials",
							Key:    "token",
						},
					},
				},
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "https://github.com/superplanehq/superplane.git", request.URL)
		assert.Equal(t, "/home/daytona/superplane", request.Path)
		assert.Equal(t, "x-access-token", request.Username)
		assert.Equal(t, "ghp_test_token", request.Password)
	})

	t.Run("no github token secret keeps clone request without credentials", func(t *testing.T) {
		request, err := component.cloneRepositoryRequest(
			&contexts.SecretsContext{},
			&CreateRepositorySandboxMetadata{
				Repository: "https://github.com/superplanehq/superplane.git",
				Directory:  "/home/daytona/superplane",
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "https://github.com/superplanehq/superplane.git", request.URL)
		assert.Equal(t, "/home/daytona/superplane", request.Path)
		assert.Empty(t, request.Username)
		assert.Empty(t, request.Password)
	})
}

func newTestLogger() *log.Entry {
	return log.NewEntry(log.New())
}

func ptr(value string) *string {
	return &value
}
