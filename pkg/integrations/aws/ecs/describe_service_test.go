package ecs

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DescribeService__Setup(t *testing.T) {
	component := &DescribeService{}

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
				"service": "web",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing cluster -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"service": "web",
			},
		})

		require.ErrorContains(t, err, "cluster is required")
	})

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
			},
		})

		require.ErrorContains(t, err, "service is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
				"service": "web",
			},
		})

		require.NoError(t, err)
	})
}

func Test__DescribeService__Execute(t *testing.T) {
	component := &DescribeService{}

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
				"service": "web",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits service", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"services": [
								{
									"serviceArn": "arn:aws:ecs:us-east-1:123456789012:service/demo/web",
									"serviceName": "web",
									"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/demo",
									"status": "ACTIVE",
									"desiredCount": 2,
									"runningCount": 2,
									"pendingCount": 0,
									"taskSets": [
										{
											"id": "ecs-svc/1234567890123456789",
											"taskSetArn": "arn:aws:ecs:us-east-1:123456789012:task-set/demo/web/1234567890123456789",
											"status": "PRIMARY",
											"taskDefinition": "arn:aws:ecs:us-east-1:123456789012:task-definition/web:7",
											"runningCount": 2
										}
									]
								}
							]
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
				"service": "web",
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
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)

		service, ok := payload["service"].(Service)
		require.True(t, ok)
		assert.Equal(t, "web", service.ServiceName)
		assert.Equal(t, "ACTIVE", service.Status)
		require.Len(t, service.TaskSets, 1)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task-set/demo/web/1234567890123456789", service.TaskSets[0].TaskSetArn)
		assert.Equal(t, "PRIMARY", service.TaskSets[0].Status)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ecs.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})
}
