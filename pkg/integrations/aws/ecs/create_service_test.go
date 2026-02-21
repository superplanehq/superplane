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
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateService__Setup(t *testing.T) {
	component := &CreateService{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing service name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
			},
		})

		require.ErrorContains(t, err, "service name is required")
	})

	t.Run("daemon scheduling with desired count -> error", func(t *testing.T) {
		desiredCount := 2
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"cluster":            "demo",
				"serviceName":        "api",
				"taskDefinition":     "worker:1",
				"schedulingStrategy": "DAEMON",
				"desiredCount":       desiredCount,
			},
		})

		require.ErrorContains(t, err, "desired count cannot be set when scheduling strategy is DAEMON")
	})
}

func Test__CreateService__Execute(t *testing.T) {
	component := &CreateService{}

	t.Run("valid request -> emits service", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"service": {
								"serviceArn": "arn:aws:ecs:us-east-1:123456789012:service/demo/api",
								"serviceName": "api",
								"status": "ACTIVE",
								"taskDefinition": "arn:aws:ecs:us-east-1:123456789012:task-definition/api:2",
								"desiredCount": 2
							}
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":               "us-east-1",
				"cluster":              "demo",
				"serviceName":          "api",
				"taskDefinition":       "api:2",
				"desiredCount":         2,
				"launchType":           "FARGATE",
				"enableExecuteCommand": true,
				"tags": []any{
					map[string]any{"key": "env", "value": "prod"},
				},
				"additionalCreateOrUpdateArguments": map[string]any{
					"deploymentController": map[string]any{
						"type": "ECS",
					},
				},
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)

		service, ok := payload["service"].(Service)
		require.True(t, ok)
		assert.Equal(t, "api", service.ServiceName)
		assert.Equal(t, "ACTIVE", service.Status)

		require.Len(t, httpContext.Requests, 1)
		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.Equal(t, "demo", payloadSent["cluster"])
		assert.Equal(t, "api", payloadSent["serviceName"])
		assert.Equal(t, "api:2", payloadSent["taskDefinition"])
		assert.Equal(t, float64(2), payloadSent["desiredCount"])
		assert.Equal(t, "FARGATE", payloadSent["launchType"])
		assert.Equal(t, true, payloadSent["enableExecuteCommand"])
		assert.Contains(t, payloadSent, "deploymentController")
	})
}
