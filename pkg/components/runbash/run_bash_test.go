package runbash

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gopkg.in/yaml.v3"
)

func TestBuildspecValidYAML(t *testing.T) {
	spec := Spec{
		Commands: "echo hello\necho world",
		Source: &SourceSpec{
			Repository: "https://github.com/example/app.git",
			Ref:        "main",
			Depth:      1,
		},
		WorkingDirectory: "subdir",
	}
	out := buildspec(spec)
	var parsed any
	err := yaml.Unmarshal([]byte(out), &parsed)
	require.NoError(t, err, "buildspec must be valid YAML (CodeBuild parses it before running commands)")
}

func TestRunBashSetup(t *testing.T) {
	component := &RunBash{}

	t.Run("commands are required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commands is required")
	})

	t.Run("invalid environment variable returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"commands": "echo hello",
			"environment": []map[string]any{
				{"name": "1INVALID", "value": "x"},
			},
		}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid environment variable name")
	})

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: validConfig()})
		require.NoError(t, err)
	})
}

func TestRunBashExecuteStartsCodeBuild(t *testing.T) {
	withBackendEnv(t)
	component := &RunBash{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"build":{"id":"build-1","arn":"arn:build-1","buildStatus":"IN_PROGRESS","startTime":1760000000,"logs":{"groupName":"group","streamName":"stream","deepLink":"https://logs"}}}`),
	}}
	metadata := &contexts.MetadataContext{}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requests := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration:  validConfig(),
		HTTP:           httpCtx,
		Metadata:       metadata,
		ExecutionState: state,
		Requests:       requests,
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{
			"git/token": []byte("secret-token"),
		}},
	})
	require.NoError(t, err)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "CodeBuild_20161006.StartBuild", httpCtx.Requests[0].Header.Get("X-Amz-Target"))
	assert.Equal(t, "build-1", state.KVs["codebuild_build_id"])
	assert.Equal(t, hookPoll, requests.Action)
	assert.Equal(t, defaultPollInterval, requests.Duration)

	stored, err := decodeMetadata(metadata.Get())
	require.NoError(t, err)
	assert.Equal(t, "build-1", stored.BuildID)
	assert.Equal(t, "IN_PROGRESS", stored.Status)
	assert.Equal(t, "github.com/example/app.git", stored.Source.Repository)
}

func TestRunBashPollCompletesSuccessfulBuild(t *testing.T) {
	withBackendEnv(t)
	component := &RunBash{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"builds":[{"id":"build-1","arn":"arn:build-1","buildStatus":"SUCCEEDED","startTime":1760000000,"endTime":1760000042,"logs":{"groupName":"group","streamName":"stream","deepLink":"https://logs"}}]}`),
		jsonResponse(`{"events":[{"timestamp":1760000001,"message":"Successfully built image"},{"timestamp":1760000002,"message":"SUPERPLANE_EXIT_CODE=0"}]}`),
	}}
	metadata := &contexts.MetadataContext{Metadata: ExecutionMetadata{BuildID: "build-1", Status: "IN_PROGRESS"}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.HandleHook(core.ActionHookContext{
		Name:           hookPoll,
		Configuration:  validConfig(),
		HTTP:           httpCtx,
		Metadata:       metadata,
		ExecutionState: state,
		Requests:       &contexts.RequestContext{},
		Secrets:        &contexts.SecretsContext{},
	})
	require.NoError(t, err)

	assert.True(t, state.Finished)
	assert.True(t, state.Passed)
	assert.Equal(t, channelSuccess, state.Channel)
	assert.Equal(t, payloadType, state.Type)

	stored, err := decodeMetadata(metadata.Get())
	require.NoError(t, err)
	require.NotNil(t, stored.ExitCode)
	assert.Equal(t, 0, *stored.ExitCode)
	assert.Contains(t, stored.Output.Stdout, "Successfully built image")
}

func TestRunBashPollCompletesFailedBuild(t *testing.T) {
	withBackendEnv(t)
	component := &RunBash{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"builds":[{"id":"build-1","arn":"arn:build-1","buildStatus":"FAILED","startTime":1760000000,"endTime":1760000042,"logs":{"groupName":"group","streamName":"stream"}}]}`),
		jsonResponse(`{"events":[{"timestamp":1760000001,"message":"terraform failed"},{"timestamp":1760000002,"message":"SUPERPLANE_EXIT_CODE=1"}]}`),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.HandleHook(core.ActionHookContext{
		Name:           hookPoll,
		Configuration:  validConfig(),
		HTTP:           httpCtx,
		Metadata:       &contexts.MetadataContext{Metadata: ExecutionMetadata{BuildID: "build-1", Status: "IN_PROGRESS"}},
		ExecutionState: state,
		Requests:       &contexts.RequestContext{},
		Secrets:        &contexts.SecretsContext{},
	})
	require.NoError(t, err)

	assert.True(t, state.Finished)
	assert.True(t, state.Passed)
	assert.Equal(t, channelFailed, state.Channel)
}

func validConfig() map[string]any {
	return map[string]any{
		"commands":         "echo hello",
		"workingDirectory": ".",
		"timeout":          600,
		"source": map[string]any{
			"repository": "https://github.com/example/app.git",
			"ref":        "main",
			"depth":      1,
			"token": map[string]any{
				"secret": "git",
				"key":    "token",
			},
		},
		"docker": map[string]any{"enabled": true},
	}
}

func withBackendEnv(t *testing.T) {
	t.Setenv("RUN_BASH_CODEBUILD_REGION", "us-east-1")
	t.Setenv("RUN_BASH_CODEBUILD_PROJECT", "superplane-run-bash")
	t.Setenv("RUN_BASH_CODEBUILD_DOCKER_PROJECT", "superplane-run-bash-docker")
	t.Setenv("RUN_BASH_AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("RUN_BASH_AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("RUN_BASH_AWS_SESSION_TOKEN", "token")
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
		Request:    &http.Request{},
	}
}

func TestParseExitCode(t *testing.T) {
	code, ok := parseExitCode("line\nSUPERPLANE_EXIT_CODE=123\n")
	require.True(t, ok)
	assert.Equal(t, 123, code)

	_, ok = parseExitCode(time.Now().String())
	assert.False(t, ok)
}
