package ecs

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__StopTask__Setup(t *testing.T) {
	component := &StopTask{}

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
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing cluster -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"task":   "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
			},
		})

		require.ErrorContains(t, err, "cluster is required")
	})

	t.Run("missing task -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
			},
		})

		require.ErrorContains(t, err, "task is required")
	})

	t.Run("rule missing -> schedules rule provisioning", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		integration := setupStopTaskIntegrationContext(&common.EventBridgeMetadata{
			Rules: map[string]common.EventBridgeRuleMetadata{},
		})

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
			},
			Metadata:    metadata,
			Requests:    requests,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integration.ActionRequests[0].ActionName)
		assert.Equal(t, stopTaskCheckRuleAvailabilityAction, requests.Action)
		assert.Equal(t, stopTaskInitialRuleAvailabilityCheck, requests.Duration)

		nodeMetadata, ok := metadata.Metadata.(StopTaskNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", nodeMetadata.Region)
		assert.Empty(t, nodeMetadata.SubscriptionID)
	})

	t.Run("rule exists -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := setupStopTaskIntegrationContext(&common.EventBridgeMetadata{
			Rules: map[string]common.EventBridgeRuleMetadata{
				ecsEventBridgeSource: {
					Source:      ecsEventBridgeSource,
					Region:      "us-east-1",
					Name:        "ecs-task-events",
					RuleArn:     "arn:aws:events:us-east-1:123:rule/ecs-task-events",
					DetailTypes: []string{ecsTaskStateChangeEventDetailType},
				},
			},
		})

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
			},
			Metadata:    metadata,
			Requests:    &contexts.RequestContext{},
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)

		nodeMetadata, ok := metadata.Metadata.(StopTaskNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", nodeMetadata.Region)
		assert.NotEmpty(t, nodeMetadata.SubscriptionID)
	})
}

func Test__StopTask__Execute(t *testing.T) {
	component := &StopTask{}

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
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("missing task in response -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"task": {}
						}
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    validStopTaskIntegrationContext(),
		})

		require.ErrorContains(t, err, "response did not include a task")
	})

	t.Run("task already stopped -> emits immediately", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"task": {
								"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
								"lastStatus": "STOPPED",
								"desiredStatus": "STOPPED"
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
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
				"reason":  "manual test stop",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validStopTaskIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)

		task, ok := payload["task"].(Task)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", task.TaskArn)
		assert.Equal(t, "STOPPED", task.LastStatus)

		require.Len(t, httpContext.Requests, 1)
		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.Equal(t, "demo", payloadSent["cluster"])
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", payloadSent["task"])
		assert.Equal(t, "manual test stop", payloadSent["reason"])
	})

	t.Run("task not stopped yet -> waits for integration message", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"task": {
								"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
								"lastStatus": "DEACTIVATING",
								"desiredStatus": "STOPPED"
							}
						}
					`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
			},
			HTTP:           httpContext,
			Metadata:       metadata,
			ExecutionState: execState,
			Integration:    validStopTaskIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Empty(t, execState.Payloads)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", execState.KVs[ecsTaskExecutionKVTaskARN])

		executionMetadata, ok := metadata.Metadata.(StopTaskExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", executionMetadata.Region)
		assert.Equal(t, "demo", executionMetadata.Cluster)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", executionMetadata.TaskARN)
	})
}

func Test__StopTask__OnIntegrationMessage(t *testing.T) {
	component := &StopTask{}

	t.Run("unknown task -> ignore", func(t *testing.T) {
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: stopTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/missing", "STOPPED"),
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return nil, nil
			},
		})

		require.NoError(t, err)
	})

	t.Run("non-stopped event -> ignore", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: stopTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/abc", "RUNNING"),
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Empty(t, execState.Payloads)
	})

	t.Run("stopped event -> emits output", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: stopTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/abc", "STOPPED"),
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		task, ok := payload["task"].(Task)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123:task/demo/abc", task.TaskArn)
		assert.Equal(t, "STOPPED", task.LastStatus)
	})

	t.Run("stopped event with RFC3339 timestamps -> emits output", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		event := stopTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/with-time", "STOPPED")
		event.Detail["createdAt"] = "2026-02-16T15:38:23.288Z"

		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: event,
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		task, ok := payload["task"].(Task)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123:task/demo/with-time", task.TaskArn)
		assert.Equal(t, "STOPPED", task.LastStatus)
		assert.False(t, task.CreatedAt.IsZero())
	})
}

func setupStopTaskIntegrationContext(eventBridge *common.EventBridgeMetadata) *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Metadata: common.IntegrationMetadata{EventBridge: eventBridge},
		Secrets:  map[string]core.IntegrationSecret{},
	}
}

func validStopTaskIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Secrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}

func stopTaskStateChangeEvent(taskARN string, status string) common.EventBridgeEvent {
	return common.EventBridgeEvent{
		Region:     "us-east-1",
		Source:     ecsEventBridgeSource,
		DetailType: ecsTaskStateChangeEventDetailType,
		Detail: map[string]any{
			"taskArn":       taskARN,
			"lastStatus":    status,
			"desiredStatus": "STOPPED",
		},
	}
}
