package ecs

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
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunTask__Setup(t *testing.T) {
	component := &RunTask{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
			Metadata:      &contexts.MetadataContext{},
			Requests:      &contexts.RequestContext{},
			Integration:   setupIntegrationContext(nil),
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("negative timeout -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"timeoutSeconds": -1,
			},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
			Integration: setupIntegrationContext(nil),
		})

		require.ErrorContains(t, err, "timeout seconds cannot be negative")
	})

	t.Run("count is zero -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"count":          0,
			},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
			Integration: setupIntegrationContext(nil),
		})

		require.ErrorContains(t, err, "count must be at least 1")
	})

	t.Run("count is above api limit -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"count":          11,
			},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
			Integration: setupIntegrationContext(nil),
		})

		require.ErrorContains(t, err, "count cannot exceed 10")
	})

	t.Run("launch type with capacity provider strategy -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"launchType":     "FARGATE",
				"capacityProviderStrategy": []any{
					map[string]any{
						"capacityProvider": "FARGATE_SPOT",
						"weight":           1,
					},
				},
			},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
			Integration: setupIntegrationContext(nil),
		})

		require.ErrorContains(t, err, "launch type cannot be combined with capacity provider strategy")
	})

	t.Run("capacity provider strategy item missing provider -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"capacityProviderStrategy": []any{
					map[string]any{
						"capacityProvider": "  ",
						"weight":           1,
					},
				},
			},
			Metadata:    &contexts.MetadataContext{},
			Requests:    &contexts.RequestContext{},
			Integration: setupIntegrationContext(nil),
		})

		require.ErrorContains(t, err, "capacity provider is required for each strategy item")
	})

	t.Run("rule missing -> schedules rule provisioning", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		integration := setupIntegrationContext(&common.EventBridgeMetadata{
			Rules: map[string]common.EventBridgeRuleMetadata{},
		})

		err := component.Setup(core.SetupContext{
			Logger: log.NewEntry(log.New()),
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
			},
			Metadata:    metadata,
			Requests:    requests,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.ActionRequests, 1)
		assert.Equal(t, "provisionRule", integration.ActionRequests[0].ActionName)
		assert.Equal(t, runTaskCheckRuleAvailabilityAction, requests.Action)
		assert.Equal(t, runTaskInitialRuleAvailabilityCheck, requests.Duration)

		nodeMetadata, ok := metadata.Metadata.(RunTaskNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", nodeMetadata.Region)
		assert.Empty(t, nodeMetadata.SubscriptionID)
	})

	t.Run("rule exists -> subscribes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := setupIntegrationContext(&common.EventBridgeMetadata{
			Rules: map[string]common.EventBridgeRuleMetadata{
				"aws.ecs:us-east-1": {
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
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
			},
			Metadata:    metadata,
			Requests:    &contexts.RequestContext{},
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)

		nodeMetadata, ok := metadata.Metadata.(RunTaskNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", nodeMetadata.Region)
		assert.NotEmpty(t, nodeMetadata.SubscriptionID)
	})
}

func Test__RunTask__Execute(t *testing.T) {
	component := &RunTask{}

	t.Run("run task failure -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [],
							"failures": [
								{
									"reason": "MISSING",
									"detail": "Task definition not found"
								}
							]
						}
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    validIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to run ECS task: MISSING (Task definition not found)")
	})

	t.Run("started tasks and no timeout -> emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
									"lastStatus": "RUNNING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"count":          2,
				"launchType":     "FARGATE",
				"startedBy":      "superplane-test",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, false, payload["timedOut"])

		require.Len(t, httpContext.Requests, 1)
		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.Equal(t, "demo", payloadSent["cluster"])
		assert.Equal(t, "worker:1", payloadSent["taskDefinition"])
		assert.Equal(t, float64(2), payloadSent["count"])
	})

	t.Run("default network configuration template -> omits networkConfiguration from API payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
									"lastStatus": "RUNNING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"networkConfiguration": map[string]any{
					"awsvpcConfiguration": map[string]any{
						"subnets":        []any{},
						"securityGroups": []any{},
						"assignPublicIp": "DISABLED",
					},
				},
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.NotContains(t, payloadSent, "networkConfiguration")
	})

	t.Run("capacity provider strategy -> sends runTask with capacityProviderStrategy", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
									"lastStatus": "RUNNING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"launchType":     "AUTO",
				"capacityProviderStrategy": []any{
					map[string]any{
						"capacityProvider": "FARGATE_SPOT",
						"weight":           1,
					},
				},
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.NotContains(t, payloadSent, "launchType")
		capacityProviderStrategy, ok := payloadSent["capacityProviderStrategy"].([]any)
		require.True(t, ok)
		require.Len(t, capacityProviderStrategy, 1)
		firstItem, ok := capacityProviderStrategy[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "FARGATE_SPOT", firstItem["capacityProvider"])
		assert.Equal(t, float64(1), firstItem["weight"])
	})

	t.Run("pending task and no timeout -> waits for integration message", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
									"lastStatus": "PENDING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
			},
			HTTP:           httpContext,
			Metadata:       metadata,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", execState.KVs[ecsTaskExecutionKVTaskARN])

		executionMetadata, ok := metadata.Metadata.(RunTaskExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", executionMetadata.Region)
		assert.Equal(t, "demo", executionMetadata.Cluster)
		assert.Equal(t, []string{"arn:aws:ecs:us-east-1:123456789012:task/demo/abc"}, executionMetadata.TaskARNs)
		assert.Equal(t, 0, executionMetadata.TimeoutSeconds)
		assert.NotEmpty(t, executionMetadata.StartedAt)
	})

	t.Run("running task with long timeout -> waits for stopped and schedules timeout checks every 5 minutes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
									"lastStatus": "RUNNING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		requests := &contexts.RequestContext{}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"timeoutSeconds": 3600,
			},
			HTTP:           httpContext,
			Metadata:       metadata,
			Requests:       requests,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
		assert.Equal(t, runTaskTimeoutAction, requests.Action)
		assert.Equal(t, 5*time.Minute, requests.Duration)

		executionMetadata, ok := metadata.Metadata.(RunTaskExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, 3600, executionMetadata.TimeoutSeconds)
		assert.NotEmpty(t, executionMetadata.DeadlineAt)
	})
}

func Test__RunTask__HandleAction(t *testing.T) {
	component := &RunTask{}

	t.Run("timeout action before deadline -> reschedules timeout check", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name: runTaskTimeoutAction,
			Metadata: &contexts.MetadataContext{
				Metadata: RunTaskExecutionMetadata{
					Region:         "us-east-1",
					Cluster:        "demo",
					TaskARNs:       []string{"arn:aws:ecs:us-east-1:123:task/demo/abc"},
					TimeoutSeconds: 3600,
					StartedAt:      time.Now().UTC().Format(time.RFC3339Nano),
				},
			},
			Requests:       requests,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.NoError(t, err)
		assert.Equal(t, runTaskTimeoutAction, requests.Action)
		assert.Equal(t, 5*time.Minute, requests.Duration)
	})

	t.Run("timeout action after deadline -> emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123:task/demo/abc",
									"lastStatus": "RUNNING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: runTaskTimeoutAction,
			Metadata: &contexts.MetadataContext{
				Metadata: RunTaskExecutionMetadata{
					Region:         "us-east-1",
					Cluster:        "demo",
					TaskARNs:       []string{"arn:aws:ecs:us-east-1:123:task/demo/abc"},
					TimeoutSeconds: 1,
					StartedAt:      time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339Nano),
				},
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, true, payload["timedOut"])
	})
}

func Test__RunTask__OnIntegrationMessage(t *testing.T) {
	component := &RunTask{}

	t.Run("unknown task -> ignore", func(t *testing.T) {
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: ecsTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/missing"),
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return nil, nil
			},
		})

		require.NoError(t, err)
	})

	t.Run("started task and no timeout -> emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123:task/demo/abc",
									"lastStatus": "RUNNING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: ecsTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/abc"),
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata: &contexts.MetadataContext{Metadata: RunTaskExecutionMetadata{
						Region:    "us-east-1",
						Cluster:   "demo",
						TaskARNs:  []string{"arn:aws:ecs:us-east-1:123:task/demo/abc"},
						StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
					}},
					ExecutionState: execState,
					HTTP:           httpContext,
					Integration:    validIntegrationContext(),
				}, nil
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, false, payload["timedOut"])
	})

	t.Run("timeout configured and task still running -> keep waiting", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123:task/demo/abc",
									"lastStatus": "RUNNING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: ecsTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/abc"),
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata: &contexts.MetadataContext{Metadata: RunTaskExecutionMetadata{
						Region:         "us-east-1",
						Cluster:        "demo",
						TaskARNs:       []string{"arn:aws:ecs:us-east-1:123:task/demo/abc"},
						TimeoutSeconds: 300,
						StartedAt:      time.Now().UTC().Format(time.RFC3339Nano),
					}},
					ExecutionState: execState,
					HTTP:           httpContext,
					Integration:    validIntegrationContext(),
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Empty(t, execState.Payloads)
	})

	t.Run("timeout configured and all tasks stopped -> emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123:task/demo/abc",
									"lastStatus": "STOPPED"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: ecsTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/abc"),
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata: &contexts.MetadataContext{Metadata: RunTaskExecutionMetadata{
						Region:         "us-east-1",
						Cluster:        "demo",
						TaskARNs:       []string{"arn:aws:ecs:us-east-1:123:task/demo/abc"},
						TimeoutSeconds: 300,
						StartedAt:      time.Now().UTC().Format(time.RFC3339Nano),
					}},
					ExecutionState: execState,
					HTTP:           httpContext,
					Integration:    validIntegrationContext(),
				}, nil
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, false, payload["timedOut"])
	})

	t.Run("timeout reached and task still running -> emits timed out output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123:task/demo/abc",
									"lastStatus": "RUNNING"
								}
							],
							"failures": []
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: ecsTaskStateChangeEvent("arn:aws:ecs:us-east-1:123:task/demo/abc"),
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata: &contexts.MetadataContext{Metadata: RunTaskExecutionMetadata{
						Region:         "us-east-1",
						Cluster:        "demo",
						TaskARNs:       []string{"arn:aws:ecs:us-east-1:123:task/demo/abc"},
						TimeoutSeconds: 1,
						StartedAt:      time.Now().UTC().Add(-5 * time.Second).Format(time.RFC3339Nano),
					}},
					ExecutionState: execState,
					HTTP:           httpContext,
					Integration:    validIntegrationContext(),
				}, nil
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, true, payload["timedOut"])
	})
}

func setupIntegrationContext(eventBridge *common.EventBridgeMetadata) *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Metadata: common.IntegrationMetadata{EventBridge: eventBridge},
		Secrets:  map[string]core.IntegrationSecret{},
	}
}

func validIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Secrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}

func ecsTaskStateChangeEvent(taskARN string) common.EventBridgeEvent {
	return common.EventBridgeEvent{
		Region:     "us-east-1",
		Source:     ecsEventBridgeSource,
		DetailType: ecsTaskStateChangeEventDetailType,
		Detail: map[string]any{
			"taskArn": taskARN,
		},
	}
}
