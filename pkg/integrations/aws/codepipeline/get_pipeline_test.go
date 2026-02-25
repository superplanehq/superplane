package codepipeline

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

func Test__GetPipeline__Setup(t *testing.T) {
	component := &GetPipeline{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   " ",
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

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
		})
		require.NoError(t, err)
	})
}

func Test__GetPipeline__Execute(t *testing.T) {
	component := &GetPipeline{}

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
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits pipeline definition", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
                        "pipeline": {
                            "name": "my-pipeline",
                            "roleArn": "arn:aws:iam::123456789012:role/pipeline-role",
                            "version": 3,
                            "stages": [
                                {
                                    "name": "Source",
                                    "actions": [
                                        {"name": "SourceAction", "actionTypeId": {"category": "Source"}}
                                    ]
                                },
                                {
                                    "name": "Deploy",
                                    "actions": [
                                        {"name": "DeployAction", "actionTypeId": {"category": "Deploy"}}
                                    ]
                                }
                            ]
                        },
                        "metadata": {
                            "pipelineArn": "arn:aws:codepipeline:us-east-1:123456789012:my-pipeline",
                            "created": "2025-01-15T10:30:00Z",
                            "updated": "2026-02-20T14:00:00Z"
                        }
                    }`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
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

		// Verify the HTTP request went to the right endpoint
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t,
			"https://codepipeline.us-east-1.amazonaws.com/",
			httpContext.Requests[0].URL.String(),
		)
	})
}
