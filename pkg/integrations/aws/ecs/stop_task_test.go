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

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
				"task":    "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
				"reason":  "manual stop",
			},
		})

		require.NoError(t, err)
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
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.ErrorContains(t, err, "response did not include a task")
	})

	t.Run("valid request -> emits task", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"task": {
								"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
								"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/demo",
								"taskDefinitionArn": "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:1",
								"lastStatus": "DEACTIVATING",
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

		task, ok := payload["task"].(Task)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", task.TaskArn)
		assert.Equal(t, "STOPPED", task.DesiredStatus)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ecs.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())

		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.Equal(t, "demo", payloadSent["cluster"])
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", payloadSent["task"])
		assert.Equal(t, "manual test stop", payloadSent["reason"])
	})
}
