package codepipeline

import (
	"encoding/json"
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

func Test__RunPipeline__Setup(t *testing.T) {
	component := &RunPipeline{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipeline": "my-pipeline",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing pipeline -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})

		require.ErrorContains(t, err, "pipeline is required")
	})

	t.Run("already cached metadata matches -> no API call", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
			Metadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					SubscriptionID: "sub-123",
					Region:         "us-east-1",
					Pipeline: &PipelineMetadata{
						Name: "my-pipeline",
					},
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("pipeline not found -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"pipelines": []}`)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "nonexistent-pipeline",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
			HTTP:     httpCtx,
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
		})

		require.ErrorContains(t, err, "pipeline not found")
	})

	t.Run("pipeline found -> stores metadata and subscribes", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipelines": [
							{"name": "my-pipeline", "arn": "arn:aws:codepipeline:us-east-1:123456789:my-pipeline"}
						]
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
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
			Metadata:    metadataCtx,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)

		storedMetadata, ok := metadataCtx.Metadata.(RunPipelineNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "my-pipeline", storedMetadata.Pipeline.Name)

		require.Len(t, integrationCtx.Subscriptions, 1)
		assert.NotEmpty(t, storedMetadata.SubscriptionID)
	})
}

func Test__RunPipeline__Execute(t *testing.T) {
	component := &RunPipeline{}

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
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					Pipeline: &PipelineMetadata{
						Name: "my-pipeline",
					},
				},
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("successful execution -> starts pipeline and schedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"pipelineExecutionId": "exec-123"}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: RunPipelineNodeMetadata{
				Pipeline: &PipelineMetadata{
					Name: "my-pipeline",
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
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
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

		assert.Equal(t, "exec-123", execState.KVs["pipeline_execution_id"])
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "codepipeline.us-east-1.amazonaws.com")
	})
}

func Test__RunPipeline__HandleWebhook(t *testing.T) {
	component := &RunPipeline{}

	t.Run("invalid body -> error", func(t *testing.T) {
		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: []byte("invalid json"),
		})

		assert.Equal(t, http.StatusBadRequest, status)
		require.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("missing detail -> error", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{})
		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
		})

		assert.Equal(t, http.StatusBadRequest, status)
		require.ErrorContains(t, err, "missing detail")
	})

	t.Run("SUCCEEDED state -> emits to passed channel", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"detail": map[string]any{
				"pipeline":     "my-pipeline",
				"execution-id": "exec-123",
				"state":        "SUCCEEDED",
			},
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		execMetadata := &contexts.MetadataContext{
			Metadata: RunPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{
					Name: "my-pipeline",
				},
				Execution: &ExecutionMetadata{
					ID:     "exec-123",
					Status: PipelineStatusInProgress,
				},
			},
		}

		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       execMetadata,
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.True(t, execState.Finished)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
		assert.Equal(t, PayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)
		wrapped, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		pipeline, ok := data["pipeline"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "SUCCEEDED", pipeline["state"])
		detail, ok := data["detail"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-pipeline", detail["pipeline"])
		assert.Equal(t, "exec-123", detail["execution-id"])
		assert.Equal(t, "SUCCEEDED", detail["state"])
	})

	t.Run("FAILED state -> emits to failed channel", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"detail": map[string]any{
				"pipeline":     "my-pipeline",
				"execution-id": "exec-456",
				"state":        "FAILED",
			},
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		execMetadata := &contexts.MetadataContext{
			Metadata: RunPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{
					Name: "my-pipeline",
				},
				Execution: &ExecutionMetadata{
					ID:     "exec-456",
					Status: PipelineStatusInProgress,
				},
			},
		}

		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       execMetadata,
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.True(t, execState.Finished)
		assert.Equal(t, FailedOutputChannel, execState.Channel)
	})

	t.Run("already finished execution -> no-op", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"detail": map[string]any{
				"pipeline":     "my-pipeline",
				"execution-id": "exec-789",
				"state":        "SUCCEEDED",
			},
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		execMetadata := &contexts.MetadataContext{
			Metadata: RunPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{
					Name: "my-pipeline",
				},
				Execution: &ExecutionMetadata{
					ID:     "exec-789",
					Status: PipelineStatusSucceeded,
				},
			},
		}

		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       execMetadata,
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.False(t, execState.Finished)
	})

	t.Run("unknown execution -> ignored", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"detail": map[string]any{
				"pipeline":     "my-pipeline",
				"execution-id": "unknown-exec",
				"state":        "SUCCEEDED",
			},
		})

		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, assert.AnError
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("non-terminal state -> no emit", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"detail": map[string]any{
				"pipeline":     "my-pipeline",
				"execution-id": "exec-123",
				"state":        "STARTED",
			},
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		execMetadata := &contexts.MetadataContext{
			Metadata: RunPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{
					Name: "my-pipeline",
				},
				Execution: &ExecutionMetadata{
					ID:     "exec-123",
					Status: PipelineStatusInProgress,
				},
			},
		}

		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       execMetadata,
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.False(t, execState.Finished)
	})
}

func Test__RunPipeline__Poll(t *testing.T) {
	component := &RunPipeline{}

	t.Run("already finished -> no-op", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
			ExecutionState: &contexts.ExecutionStateContext{
				Finished: true,
				KVs:      map[string]string{},
			},
		})

		require.NoError(t, err)
	})

	t.Run("pipeline still running -> schedules next poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipelineExecution": {
							"pipelineExecutionId": "exec-123",
							"status": "InProgress",
							"pipelineName": "my-pipeline"
						}
					}`)),
				},
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
			Metadata: &contexts.MetadataContext{
				Metadata: RunPipelineExecutionMetadata{
					Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
					Execution: &ExecutionMetadata{ID: "exec-123", Status: PipelineStatusInProgress},
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

	t.Run("pipeline succeeded -> emits to passed channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipelineExecution": {
							"pipelineExecutionId": "exec-123",
							"status": "Succeeded",
							"pipelineName": "my-pipeline"
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
			Metadata: &contexts.MetadataContext{
				Metadata: RunPipelineExecutionMetadata{
					Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
					Execution: &ExecutionMetadata{ID: "exec-123", Status: PipelineStatusInProgress},
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
		assert.Equal(t, PayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)
		wrapped, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		pipeline, ok := data["pipeline"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "SUCCEEDED", pipeline["state"])
		detail, ok := data["detail"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-pipeline", detail["pipeline"])
		assert.Equal(t, "exec-123", detail["execution-id"])
		assert.Equal(t, "SUCCEEDED", detail["state"])
	})

	t.Run("pipeline failed -> emits to failed channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipelineExecution": {
							"pipelineExecutionId": "exec-123",
							"status": "Failed",
							"pipelineName": "my-pipeline"
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
			Metadata: &contexts.MetadataContext{
				Metadata: RunPipelineExecutionMetadata{
					Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
					Execution: &ExecutionMetadata{ID: "exec-123", Status: PipelineStatusInProgress},
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

func Test__RunPipeline__Finish(t *testing.T) {
	component := &RunPipeline{}

	t.Run("already finished execution -> no-op", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "finish",
			Metadata: &contexts.MetadataContext{
				Metadata: RunPipelineExecutionMetadata{
					Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
					Execution: &ExecutionMetadata{ID: "exec-123", Status: PipelineStatusInProgress},
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
				Metadata: RunPipelineExecutionMetadata{
					Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
					Execution: &ExecutionMetadata{ID: "exec-123", Status: PipelineStatusInProgress},
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
		assert.Equal(t, PayloadType, execState.Type)
	})

	t.Run("without data -> emits to passed channel", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: "finish",
			Metadata: &contexts.MetadataContext{
				Metadata: RunPipelineExecutionMetadata{
					Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
					Execution: &ExecutionMetadata{ID: "exec-123", Status: PipelineStatusInProgress},
				},
			},
			ExecutionState: execState,
			Parameters:     map[string]any{},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
	})

	t.Run("pipeline already completed -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "finish",
			Metadata: &contexts.MetadataContext{
				Metadata: RunPipelineExecutionMetadata{
					Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
					Execution: &ExecutionMetadata{ID: "exec-123", Status: PipelineStatusSucceeded},
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Parameters:     map[string]any{},
		})

		require.ErrorContains(t, err, "pipeline execution already finished")
	})
}

func Test__RunPipeline__OnIntegrationMessage(t *testing.T) {
	component := &RunPipeline{}

	t.Run("pipeline mismatch -> ignored", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					Pipeline: &PipelineMetadata{Name: "my-pipeline"},
				},
			},
			Message: common.EventBridgeEvent{
				Source:     "aws.codepipeline",
				DetailType: "CodePipeline Pipeline Execution State Change",
				Detail:     map[string]any{"pipeline": "other-pipeline", "state": "SUCCEEDED", "execution-id": "exec-1"},
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				t.Fatal("FindExecutionByKV should not be called for mismatched pipeline")
				return nil, nil
			},
		})

		require.NoError(t, err)
		assert.False(t, executionState.Finished)
	})

	t.Run("nil pipeline metadata -> ignored", func(t *testing.T) {
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{Pipeline: nil},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"pipeline": "my-pipeline", "state": "SUCCEEDED", "execution-id": "exec-1"},
			},
		})

		require.NoError(t, err)
	})

	t.Run("non-terminal state (STARTED) -> ignored", func(t *testing.T) {
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					Pipeline: &PipelineMetadata{Name: "my-pipeline"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"pipeline": "my-pipeline", "state": "STARTED", "execution-id": "exec-1"},
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
			Metadata: RunPipelineExecutionMetadata{
				Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
				Execution: &ExecutionMetadata{ID: "exec-1", Status: PipelineStatusInProgress},
			},
		}

		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					Pipeline: &PipelineMetadata{Name: "my-pipeline"},
				},
			},
			Message: common.EventBridgeEvent{
				Source:     "aws.codepipeline",
				DetailType: "CodePipeline Pipeline Execution State Change",
				Detail:     map[string]any{"pipeline": "my-pipeline", "state": "SUCCEEDED", "execution-id": "exec-1"},
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				assert.Equal(t, "pipeline_execution_id", key)
				assert.Equal(t, "exec-1", value)
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
		assert.Equal(t, PayloadType, executionState.Type)
	})

	t.Run("FAILED -> resolves execution on failed channel", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		executionMetadata := &contexts.MetadataContext{
			Metadata: RunPipelineExecutionMetadata{
				Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
				Execution: &ExecutionMetadata{ID: "exec-1", Status: PipelineStatusInProgress},
			},
		}

		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					Pipeline: &PipelineMetadata{Name: "my-pipeline"},
				},
			},
			Message: common.EventBridgeEvent{
				Source:     "aws.codepipeline",
				DetailType: "CodePipeline Pipeline Execution State Change",
				Detail:     map[string]any{"pipeline": "my-pipeline", "state": "FAILED", "execution-id": "exec-1"},
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

	t.Run("execution not found -> ignored", func(t *testing.T) {
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					Pipeline: &PipelineMetadata{Name: "my-pipeline"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"pipeline": "my-pipeline", "state": "SUCCEEDED", "execution-id": "unknown-exec"},
			},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, nil
			},
		})

		require.NoError(t, err)
	})

	t.Run("already finished execution -> ignored", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{Finished: true, KVs: map[string]string{}}
		executionMetadata := &contexts.MetadataContext{
			Metadata: RunPipelineExecutionMetadata{
				Pipeline:  &PipelineMetadata{Name: "my-pipeline"},
				Execution: &ExecutionMetadata{ID: "exec-1", Status: PipelineStatusSucceeded},
			},
		}

		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: &contexts.EventContext{},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					Pipeline: &PipelineMetadata{Name: "my-pipeline"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"pipeline": "my-pipeline", "state": "SUCCEEDED", "execution-id": "exec-1"},
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
		assert.Empty(t, executionState.Channel, "should not re-emit on an already-resolved execution")
	})

	t.Run("FindExecutionByKV nil -> falls back to event emission", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Logger: logrus.NewEntry(logrus.New()),
			Events: eventContext,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: RunPipelineNodeMetadata{
					Pipeline: &PipelineMetadata{Name: "my-pipeline"},
				},
			},
			Message: common.EventBridgeEvent{
				Detail: map[string]any{"pipeline": "my-pipeline", "state": "SUCCEEDED", "execution-id": "exec-1"},
			},
			FindExecutionByKV: nil,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, PayloadType, eventContext.Payloads[0].Type)
	})
}

func Test__RunPipeline__ListPipelines(t *testing.T) {
	t.Run("missing region -> error", func(t *testing.T) {
		_, err := ListPipelines(core.ListResourcesContext{
			Parameters: map[string]string{},
		}, "codepipeline.pipeline")

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		_, err := ListPipelines(core.ListResourcesContext{
			Parameters:  map[string]string{"region": "us-east-1"},
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
		}, "codepipeline.pipeline")

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> returns pipelines", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipelines": [
							{"name": "pipeline-a", "arn": "arn:aws:codepipeline:us-east-1:123:pipeline-a"},
							{"name": "pipeline-b", "arn": "arn:aws:codepipeline:us-east-1:123:pipeline-b"}
						]
					}`)),
				},
			},
		}

		resources, err := ListPipelines(core.ListResourcesContext{
			Parameters: map[string]string{"region": "us-east-1"},
			HTTP:       httpCtx,
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
		}, "codepipeline.pipeline")

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "pipeline-a", resources[0].Name)
		assert.Equal(t, "pipeline-a", resources[0].ID)
		assert.Equal(t, "codepipeline.pipeline", resources[0].Type)
		assert.Equal(t, "pipeline-b", resources[1].Name)
	})
}
