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

func Test__UpdateService__Setup(t *testing.T) {
	component := &UpdateService{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
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
}

func Test__UpdateService__Execute(t *testing.T) {
	component := &UpdateService{}

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
								"desiredCount": 3,
								"taskDefinition": "arn:aws:ecs:us-east-1:123456789012:task-definition/api:3"
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
				"service":              "api",
				"desiredCount":         3,
				"taskDefinition":       "api:3",
				"forceNewDeployment":   true,
				"enableExecuteCommand": false,
				"additionalCreateOrUpdateArguments": map[string]any{
					"availabilityZoneRebalancing": "ENABLED",
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
		assert.Equal(t, 3, service.DesiredCount)

		require.Len(t, httpContext.Requests, 1)
		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.Equal(t, "demo", payloadSent["cluster"])
		assert.Equal(t, "api", payloadSent["service"])
		assert.Equal(t, float64(3), payloadSent["desiredCount"])
		assert.Equal(t, "api:3", payloadSent["taskDefinition"])
		assert.Equal(t, true, payloadSent["forceNewDeployment"])
		assert.Equal(t, false, payloadSent["enableExecuteCommand"])
		assert.Equal(t, "ENABLED", payloadSent["availabilityZoneRebalancing"])
	})
}
