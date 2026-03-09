package codebuild

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func validSecrets() map[string]core.IntegrationSecret {
	return map[string]core.IntegrationSecret{
		"accessKeyId":     {Name: "accessKeyId", Value: []byte("AKIAIOSFODNN7EXAMPLE")},
		"secretAccessKey": {Name: "secretAccessKey", Value: []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")},
		"sessionToken":    {Name: "sessionToken", Value: []byte("FwoGZXIvYXdzEBYaDH7example")},
	}
}

func Test__StartBuild__Setup(t *testing.T) {
	component := &StartBuild{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "my-project",
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

	t.Run("already cached metadata matches -> no API call", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "my-project",
			},
			Metadata: &contexts.MetadataContext{
				Metadata: StartBuildNodeMetadata{
					SubscriptionID: "sub-123",
					Region:         "us-east-1",
					Project: &ProjectMetadata{
						Name: "my-project",
					},
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("project not found -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"projects": []}`)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "nonexistent-project",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
			HTTP:     httpCtx,
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
		})

		require.ErrorContains(t, err, "project not found")
	})

	t.Run("project found -> stores metadata and subscribes", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"projects": ["my-project", "other-project"]
					}`)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{Metadata: map[string]any{}}
		integrationCtx := &contexts.IntegrationContext{
			Secrets:  validSecrets(),
			Metadata: map[string]any{},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "my-project",
			},
			Metadata:    metadataCtx,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)

		storedMetadata, ok := metadataCtx.Metadata.(StartBuildNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "my-project", storedMetadata.Project.Name)

		require.Len(t, integrationCtx.Subscriptions, 1)
		assert.NotEmpty(t, storedMetadata.SubscriptionID)
	})
}

func Test__StartBuild__Execute(t *testing.T) {
	component := &StartBuild{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "my-project",
			},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: StartBuildNodeMetadata{
					Project: &ProjectMetadata{
						Name: "my-project",
					},
				},
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("successful execution -> starts build and schedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"build": {"id": "my-project:build-123", "buildStatus": "IN_PROGRESS"}}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: StartBuildNodeMetadata{
				Project: &ProjectMetadata{
					Name: "my-project",
				},
			},
		}
		requestCtx := &contexts.RequestContext{}
		integrationCtx := &contexts.IntegrationContext{
			Secrets:  validSecrets(),
			Metadata: map[string]any{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "my-project",
			},
			NodeMetadata:   metadataCtx,
			Metadata:       &contexts.MetadataContext{Metadata: map[string]any{}},
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			ExecutionState: execState,
			Requests:       requestCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)

		assert.Equal(t, "my-project:build-123", execState.KVs["build_id"])
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "codebuild.us-east-1.amazonaws.com")
	})
}

func Test__StartBuild__Poll(t *testing.T) {
	component := &StartBuild{}

	t.Run("already finished -> no-op", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "my-project",
			},
			ExecutionState: &contexts.ExecutionStateContext{
				Finished: true,
				KVs:      map[string]string{},
			},
		})

		require.NoError(t, err)
	})

	t.Run("build still running -> schedules next poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"builds": [{
							"id": "my-project:build-123",
							"buildStatus": "IN_PROGRESS",
							"projectName": "my-project"
						}],
						"buildsNotFound": []
					}`)),
				},
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "my-project",
			},
			Metadata: &contexts.MetadataContext{
				Metadata: StartBuildExecutionMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
					Build:   &BuildMetadata{ID: "my-project:build-123", Status: BuildStatusInProgress},
				},
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)
	})

	t.Run("build succeeded -> emits to passed channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"builds": [{
							"id": "my-project:build-123",
							"buildStatus": "SUCCEEDED",
							"projectName": "my-project",
							"currentPhase": "COMPLETED"
						}],
						"buildsNotFound": []
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "my-project",
			},
			Metadata: &contexts.MetadataContext{
				Metadata: StartBuildExecutionMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
					Build:   &BuildMetadata{ID: "my-project:build-123", Status: BuildStatusInProgress},
				},
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
			ExecutionState: execState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
		assert.Equal(t, BuildPayloadType, execState.Type)
	})

	t.Run("build failed -> emits to failed channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"builds": [{
							"id": "my-project:build-123",
							"buildStatus": "FAILED",
							"projectName": "my-project",
							"currentPhase": "COMPLETED"
						}],
						"buildsNotFound": []
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":  "us-east-1",
				"project": "my-project",
			},
			Metadata: &contexts.MetadataContext{
				Metadata: StartBuildExecutionMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
					Build:   &BuildMetadata{ID: "my-project:build-123", Status: BuildStatusInProgress},
				},
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
			ExecutionState: execState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, FailedOutputChannel, execState.Channel)
	})
}

func Test__StartBuild__Finish(t *testing.T) {
	component := &StartBuild{}

	t.Run("already finished execution -> no-op", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "finish",
			Metadata: &contexts.MetadataContext{
				Metadata: StartBuildExecutionMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
					Build:   &BuildMetadata{ID: "my-project:build-123", Status: BuildStatusInProgress},
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{
				Finished: true,
				KVs:      map[string]string{},
			},
			Parameters: map[string]any{},
		})

		require.NoError(t, err)
	})

	t.Run("with data -> emits to passed channel with data", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: "finish",
			Metadata: &contexts.MetadataContext{
				Metadata: StartBuildExecutionMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
					Build:   &BuildMetadata{ID: "my-project:build-123", Status: BuildStatusInProgress},
				},
			},
			ExecutionState: execState,
			Parameters: map[string]any{
				"data": map[string]any{"reason": "manual override"},
			},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
		assert.Equal(t, BuildPayloadType, execState.Type)
	})

	t.Run("build already completed -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "finish",
			Metadata: &contexts.MetadataContext{
				Metadata: StartBuildExecutionMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
					Build:   &BuildMetadata{ID: "my-project:build-123", Status: BuildStatusSucceeded},
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Parameters:     map[string]any{},
		})

		require.ErrorContains(t, err, "build already finished")
	})
}

func Test__StartBuild__OnIntegrationMessage(t *testing.T) {
	component := &StartBuild{}

	t.Run("project mismatch -> ignored", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: StartBuildNodeMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
				},
			},
			Message: common.EventBridgeEvent{
				Source:     "aws.codebuild",
				DetailType: "CodeBuild Build State Change",
				Detail:     map[string]any{"project-name": "other-project", "build-status": "SUCCEEDED", "build-id": "other-project:build-1"},
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				t.Fatal("FindExecutionByKV should not be called for mismatched project")
				return nil, nil
			},
		})

		require.NoError(t, err)
		assert.False(t, executionState.Finished)
	})

	t.Run("non-terminal state -> ignored", func(t *testing.T) {
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: StartBuildNodeMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"project-name": "my-project", "build-status": "IN_PROGRESS", "build-id": "my-project:build-1"},
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				t.Fatal("FindExecutionByKV should not be called for non-terminal state")
				return nil, nil
			},
		})

		require.NoError(t, err)
	})

	t.Run("SUCCEEDED -> resolves execution on passed channel", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		executionMetadata := &contexts.MetadataContext{
			Metadata: StartBuildExecutionMetadata{
				Project: &ProjectMetadata{Name: "my-project"},
				Build:   &BuildMetadata{ID: "my-project:build-1", Status: BuildStatusInProgress},
			},
		}

		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: StartBuildNodeMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
				},
			},
			Message: common.EventBridgeEvent{
				Source:     "aws.codebuild",
				DetailType: "CodeBuild Build State Change",
				Detail:     map[string]any{"project-name": "my-project", "build-status": "SUCCEEDED", "build-id": "my-project:build-1"},
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				assert.Equal(t, "build_id", key)
				assert.Equal(t, "my-project:build-1", value)
				return &core.ExecutionContext{
					Metadata:       executionMetadata,
					ExecutionState: executionState,
					Logger:         logrus.NewEntry(logrus.New()),
				}, nil
			},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.Equal(t, PassedOutputChannel, executionState.Channel)
		assert.Equal(t, BuildPayloadType, executionState.Type)
	})

	t.Run("FAILED -> resolves execution on failed channel", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		executionMetadata := &contexts.MetadataContext{
			Metadata: StartBuildExecutionMetadata{
				Project: &ProjectMetadata{Name: "my-project"},
				Build:   &BuildMetadata{ID: "my-project:build-1", Status: BuildStatusInProgress},
			},
		}

		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: StartBuildNodeMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
				},
			},
			Message: common.EventBridgeEvent{
				Source:     "aws.codebuild",
				DetailType: "CodeBuild Build State Change",
				Detail:     map[string]any{"project-name": "my-project", "build-status": "FAILED", "build-id": "my-project:build-1"},
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       executionMetadata,
					ExecutionState: executionState,
					Logger:         logrus.NewEntry(logrus.New()),
				}, nil
			},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.Equal(t, FailedOutputChannel, executionState.Channel)
	})

	t.Run("FindExecutionByKV nil -> falls back to event emission", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: StartBuildNodeMetadata{
					Project: &ProjectMetadata{Name: "my-project"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"project-name": "my-project", "build-status": "SUCCEEDED", "build-id": "my-project:build-1"},
			},
			FindExecutionByKV: nil,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, BuildPayloadType, eventContext.Payloads[0].Type)
	})
}

func Test__StartBuild__ListProjects(t *testing.T) {
	t.Run("missing region -> error", func(t *testing.T) {
		_, err := ListProjects(core.ListResourcesContext{
			Parameters: map[string]string{},
		}, "codebuild.project")

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		_, err := ListProjects(core.ListResourcesContext{
			Parameters:  map[string]string{"region": "us-east-1"},
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
		}, "codebuild.project")

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> returns projects", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"projects": ["project-a", "project-b"]
					}`)),
				},
			},
		}

		resources, err := ListProjects(core.ListResourcesContext{
			Parameters: map[string]string{"region": "us-east-1"},
			HTTP:       httpCtx,
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
		}, "codebuild.project")

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "project-a", resources[0].Name)
		assert.Equal(t, "project-a", resources[0].ID)
		assert.Equal(t, "codebuild.project", resources[0].Type)
		assert.Equal(t, "project-b", resources[1].Name)
	})
}
