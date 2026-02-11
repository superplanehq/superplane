package codebuild

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
)

func Test__RunBuild__Setup(t *testing.T) {
	component := &RunBuild{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  " ",
				"project": "backend-build",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("invalid project arn -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "arn:aws:codebuild:us-east-1:123456789012:project",
			},
		})

		require.ErrorContains(t, err, "invalid project ARN")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "backend-build",
			},
		})

		require.NoError(t, err)
	})
}

func Test__RunBuild__Execute(t *testing.T) {
	component := &RunBuild{}

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "backend-build",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("build in progress -> schedules poll action", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"build": {
								"id": "backend-build:build-123",
								"projectName": "backend-build",
								"buildStatus": "IN_PROGRESS",
								"currentPhase": "BUILD"
							}
						}
					`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "backend-build",
			},
			HTTP:           httpContext,
			Metadata:       metadata,
			Requests:       requests,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(RunBuildMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "backend-build", stored.ProjectName)
		assert.Equal(t, "backend-build:build-123", stored.BuildID)

		assert.Equal(t, "pollBuild", requests.Action)
		assert.Equal(t, BuildPollInterval, requests.Duration)
	})

	t.Run("build succeeded immediately -> emits payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"build": {
								"id": "backend-build:build-123",
								"projectName": "backend-build",
								"buildStatus": "SUCCEEDED",
								"currentPhase": "COMPLETED"
							}
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "backend-build",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		assert.Equal(t, BuildPayloadType, execState.Type)
		payload := execState.Payloads[0].(map[string]any)["data"]
		build, ok := payload.(*Build)
		require.True(t, ok)
		assert.Equal(t, "SUCCEEDED", build.BuildStatus)
	})

	t.Run("build fails immediately -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"build": {
								"id": "backend-build:build-123",
								"projectName": "backend-build",
								"buildStatus": "FAILED",
								"currentPhase": "COMPLETED"
							}
						}
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "backend-build",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.ErrorContains(t, err, "status FAILED")
	})
}

func Test__RunBuild__HandleAction(t *testing.T) {
	component := &RunBuild{}

	t.Run("unknown action -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "unknown",
		})

		require.ErrorContains(t, err, "unknown action")
	})

	t.Run("build still running -> schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"builds": [
								{
									"id": "backend-build:build-123",
									"projectName": "backend-build",
									"buildStatus": "IN_PROGRESS",
									"currentPhase": "BUILD"
								}
							]
						}
					`)),
				},
			},
		}

		requests := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:     "pollBuild",
			HTTP:     httpContext,
			Requests: requests,
			Metadata: &contexts.MetadataContext{
				Metadata: RunBuildMetadata{
					Region:      "us-east-1",
					ProjectName: "backend-build",
					BuildID:     "backend-build:build-123",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "pollBuild", requests.Action)
		assert.Equal(t, BuildPollInterval, requests.Duration)
	})

	t.Run("build succeeded -> emits payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"builds": [
								{
									"id": "backend-build:build-123",
									"projectName": "backend-build",
									"buildStatus": "SUCCEEDED",
									"currentPhase": "COMPLETED"
								}
							]
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name:           "pollBuild",
			HTTP:           httpContext,
			ExecutionState: execState,
			Metadata: &contexts.MetadataContext{
				Metadata: RunBuildMetadata{
					Region:      "us-east-1",
					ProjectName: "backend-build",
					BuildID:     "backend-build:build-123",
				},
			},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"]
		build, ok := payload.(*Build)
		require.True(t, ok)
		assert.Equal(t, "SUCCEEDED", build.BuildStatus)
	})

	t.Run("build failed -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"builds": [
								{
									"id": "backend-build:build-123",
									"projectName": "backend-build",
									"buildStatus": "FAILED",
									"currentPhase": "COMPLETED"
								}
							]
						}
					`)),
				},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name: "pollBuild",
			HTTP: httpContext,
			Metadata: &contexts.MetadataContext{
				Metadata: RunBuildMetadata{
					Region:      "us-east-1",
					ProjectName: "backend-build",
					BuildID:     "backend-build:build-123",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.ErrorContains(t, err, "status FAILED")
	})
}

func Test__IsBuildTerminalStatus(t *testing.T) {
	t.Run("succeeded", func(t *testing.T) {
		finished, succeeded := isBuildTerminalStatus("SUCCEEDED")
		assert.True(t, finished)
		assert.True(t, succeeded)
	})

	t.Run("failed", func(t *testing.T) {
		finished, succeeded := isBuildTerminalStatus("FAILED")
		assert.True(t, finished)
		assert.False(t, succeeded)
	})

	t.Run("in progress", func(t *testing.T) {
		finished, succeeded := isBuildTerminalStatus("IN_PROGRESS")
		assert.False(t, finished)
		assert.False(t, succeeded)
	})

	t.Run("unknown", func(t *testing.T) {
		finished, succeeded := isBuildTerminalStatus("PENDING")
		assert.False(t, finished)
		assert.False(t, succeeded)
	})
}

func Test__BuildFailureError(t *testing.T) {
	err := buildFailureError(&Build{
		ID:          "backend-build:build-123",
		BuildStatus: "FAILED",
		Logs: &BuildLogs{
			DeepLink: "https://console.aws.amazon.com/codesuite/codebuild/projects/backend-build",
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status FAILED")
	assert.Contains(t, err.Error(), "https://console.aws.amazon.com/codesuite/codebuild/projects/backend-build")
}

func Test__RunBuild__ExecuteSchedulesPollInterval(t *testing.T) {
	component := &RunBuild{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					{
						"build": {
							"id": "backend-build:build-123",
							"projectName": "backend-build",
							"buildStatus": "IN_PROGRESS",
							"currentPhase": "BUILD"
						}
					}
				`)),
			},
		},
	}

	requests := &contexts.RequestContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":  "us-east-1",
			"project": "backend-build",
		},
		HTTP:           httpContext,
		Metadata:       &contexts.MetadataContext{},
		Requests:       requests,
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Integration: &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "pollBuild", requests.Action)
	assert.Equal(t, 10*time.Second, requests.Duration)
}
