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

func Test__RunTask__Setup(t *testing.T) {
	component := &RunTask{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         " ",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing cluster -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"taskDefinition": "worker:1",
			},
		})

		require.ErrorContains(t, err, "cluster is required")
	})

	t.Run("missing task definition -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"cluster": "demo",
			},
		})

		require.ErrorContains(t, err, "task definition is required")
	})

	t.Run("invalid launch type -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"launchType":     "INVALID",
			},
		})

		require.ErrorContains(t, err, "invalid launch type")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"count":          1,
				"launchType":     "FARGATE",
			},
		})

		require.NoError(t, err)
	})

	t.Run("auto launch type -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
				"launchType":     "AUTO",
			},
		})

		require.NoError(t, err)
	})
}

func Test__RunTask__Execute(t *testing.T) {
	component := &RunTask{}

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
				"region":         "us-east-1",
				"cluster":        "demo",
				"taskDefinition": "worker:1",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

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
									"arn": "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:1",
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
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.ErrorContains(t, err, "failed to run ECS task: MISSING (Task definition not found)")
	})

	t.Run("valid request -> emits tasks", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
									"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/demo",
									"taskDefinitionArn": "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:1",
									"lastStatus": "PENDING",
									"desiredStatus": "RUNNING"
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
				"networkConfiguration": map[string]any{
					"awsvpcConfiguration": map[string]any{
						"subnets":        []any{"subnet-123"},
						"securityGroups": []any{"sg-123"},
						"assignPublicIp": "ENABLED",
					},
				},
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

		tasks, ok := payload["tasks"].([]Task)
		require.True(t, ok)
		require.Len(t, tasks, 1)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task/demo/abc", tasks[0].TaskArn)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ecs.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())

		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)
		assert.Equal(t, "demo", payloadSent["cluster"])
		assert.Equal(t, "worker:1", payloadSent["taskDefinition"])
		assert.Equal(t, float64(2), payloadSent["count"])
		assert.Equal(t, "FARGATE", payloadSent["launchType"])
		assert.Equal(t, "superplane-test", payloadSent["startedBy"])
	})

	t.Run("empty optional objects -> not sent to ECS API", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
									"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/demo",
									"taskDefinitionArn": "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:1",
									"lastStatus": "PENDING",
									"desiredStatus": "RUNNING"
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
				"region":               "us-east-1",
				"cluster":              "demo",
				"taskDefinition":       "worker:1",
				"networkConfiguration": map[string]any{},
				"overrides":            map[string]any{},
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

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)

		_, hasNetworkConfiguration := payloadSent["networkConfiguration"]
		assert.False(t, hasNetworkConfiguration)
		_, hasOverrides := payloadSent["overrides"]
		assert.False(t, hasOverrides)
	})

	t.Run("default templates -> not sent to ECS API", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"tasks": [
								{
									"taskArn": "arn:aws:ecs:us-east-1:123456789012:task/demo/abc",
									"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/demo",
									"taskDefinitionArn": "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:1",
									"lastStatus": "PENDING",
									"desiredStatus": "RUNNING"
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
				"overrides": map[string]any{
					"containerOverrides": []any{},
				},
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

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		requestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		payloadSent := map[string]any{}
		err = json.Unmarshal(requestBody, &payloadSent)
		require.NoError(t, err)

		_, hasNetworkConfiguration := payloadSent["networkConfiguration"]
		assert.False(t, hasNetworkConfiguration)
		_, hasOverrides := payloadSent["overrides"]
		assert.False(t, hasOverrides)
	})
}
