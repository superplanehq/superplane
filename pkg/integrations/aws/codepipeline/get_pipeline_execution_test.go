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

func Test__GetPipelineExecution__Setup(t *testing.T) {
	component := &GetPipelineExecution{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":      " ",
				"pipeline":    "my-pipeline",
				"executionId": "abc-123",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing pipeline -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"executionId": "abc-123",
			},
		})
		require.ErrorContains(t, err, "pipeline is required")
	})

	t.Run("missing executionId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"pipeline": "my-pipeline",
			},
		})
		require.ErrorContains(t, err, "execution ID is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"pipeline":    "my-pipeline",
				"executionId": "a1b2c3d4-5678-90ab-cdef-111122223333",
			},
		})
		require.NoError(t, err)
	})
}

func Test__GetPipelineExecution__Execute(t *testing.T) {
	component := &GetPipelineExecution{}

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
				"region":      "us-east-1",
				"pipeline":    "my-pipeline",
				"executionId": "a1b2c3d4-5678-90ab-cdef-111122223333",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits pipeline execution details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipelineExecution": {
							"pipelineExecutionId": "a1b2c3d4-5678-90ab-cdef-111122223333",
							"pipelineName": "my-pipeline",
							"pipelineVersion": 3,
							"status": "Succeeded",
							"statusSummary": "Pipeline completed successfully",
							"artifactRevisions": [
								{
									"name": "SourceArtifact",
									"revisionId": "abc123def456",
									"revisionSummary": "Merge pull request #42"
								}
							],
							"trigger": {
								"triggerType": "StartPipelineExecution",
								"triggerDetail": "arn:aws:iam::123456789012:user/developer"
							},
							"executionMode": "SUPERSEDED"
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"pipeline":    "my-pipeline",
				"executionId": "a1b2c3d4-5678-90ab-cdef-111122223333",
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

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t,
			"https://codepipeline.us-east-1.amazonaws.com/",
			httpContext.Requests[0].URL.String(),
		)
	})
}
