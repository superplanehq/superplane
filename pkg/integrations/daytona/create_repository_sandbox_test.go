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
		require.NotNil(t, payload.Clone)
		assert.Nil(t, payload.Clone.Error)
		assert.Equal(t, "bootstrap logs", payload.Bootstrap.Result)
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
		assert.False(t, execCtx.Passed)
		assert.Equal(t, "error", execCtx.FailureReason)
		assert.Contains(t, execCtx.FailureMessage, "bootstrap script failed with exit code 2: npm ERR!")
	})

	t.Run("times out when sandbox startup exceeded timeout", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
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
		assert.True(t, execCtx.Finished)
		assert.False(t, execCtx.Passed)
		assert.Equal(t, "error", execCtx.FailureReason)
		assert.Contains(t, execCtx.FailureMessage, "sandbox creation failed on stage preparingSandbox after 1m0s")
	})

	t.Run("times out during bootstrap stage and marks execution as failed", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: CreateRepositorySandboxMetadata{
					Stage:            repositorySandboxStageBootstrapping,
					SandboxID:        "sandbox-123",
					SandboxStartedAt: time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
					Timeout:          int(time.Minute.Seconds()),
					SessionID:        "session-1",
					Bootstrap: &BootstrapMetadata{
						CmdID: "cmd-bootstrap",
					},
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
		assert.Contains(
			t,
			execCtx.FailureMessage,
			"sandbox creation failed on stage "+repositorySandboxStageBootstrapping+" after 1m0s",
		)
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
