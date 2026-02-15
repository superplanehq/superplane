package harness

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

func Test__RunPipeline__Execute(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":{"planExecutionId":"exec-1"}}`)),
		}},
	}

	metadataCtx := &contexts.MetadataContext{}
	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestCtx := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: RunPipelineSpec{PipelineIdentifier: "deploy-prod", Ref: "refs/heads/main", InputYAML: "foo: bar"},
		HTTP:          httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "poll", requestCtx.Action)
	assert.Equal(t, "exec-1", executionStateCtx.KVs["execution"])

	require.Len(t, httpCtx.Requests, 1)
	bodyBytes, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)
	assert.Contains(t, string(bodyBytes), "lastYamlToMerge")

	metadata := RunPipelineExecutionMetadata{}
	require.NoError(t, mapToStruct(metadataCtx.Get(), &metadata))
	assert.Equal(t, "exec-1", metadata.ExecutionID)
	assert.Equal(t, "deploy-prod", metadata.PipelineIdentifier)
}

func Test__RunPipeline__Setup(t *testing.T) {
	component := &RunPipeline{}

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineSpec{
				PipelineIdentifier: "deploy-prod",
				Ref:                "refs/heads/main",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing pipeline identifier", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineSpec{},
		})

		require.ErrorContains(t, err, "pipelineIdentifier is required")
	})
}

func Test__RunPipeline__Execute_APIError(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"message":"invalid request"}`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestCtx := &contexts.RequestContext{}
	metadataCtx := &contexts.MetadataContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: RunPipelineSpec{PipelineIdentifier: "deploy-prod"},
		HTTP:          httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
	})

	require.Error(t, err)
	assert.Empty(t, requestCtx.Action)
	assert.Empty(t, executionStateCtx.KVs["execution"])
}

func Test__RunPipeline__Execute_MissingExecutionID(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestCtx := &contexts.RequestContext{}
	metadataCtx := &contexts.MetadataContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: RunPipelineSpec{PipelineIdentifier: "deploy-prod"},
		HTTP:          httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
	})

	require.ErrorContains(t, err, "missing execution id")
	assert.Empty(t, requestCtx.Action)
	assert.Empty(t, executionStateCtx.KVs["execution"])
}

func Test__RunPipeline__Poll_EmitsSuccess(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`
				{"data":{"content":[{"planExecutionId":"exec-1","pipelineIdentifier":"deploy-prod","status":"SUCCEEDED","planExecutionUrl":"https://app.harness.io/executions/exec-1"}]}}
			`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{ExecutionID: "exec-1", PipelineIdentifier: "deploy-prod", Status: "running"}}

	err := component.HandleAction(core.ActionContext{
		Name:           "poll",
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, RunPipelineSuccess, executionStateCtx.Channel)
	assert.Equal(t, RunPipelinePayloadType, executionStateCtx.Type)
	require.NotEmpty(t, executionStateCtx.Payloads)
}

func Test__RunPipeline__Poll_EmitsSuccessForCompletedStatus(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`
				{"data":{"content":[{"planExecutionId":"exec-1","pipelineIdentifier":"deploy-prod","status":"COMPLETED"}]}}
			`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{ExecutionID: "exec-1", PipelineIdentifier: "deploy-prod", Status: "running"}}

	err := component.HandleAction(core.ActionContext{
		Name:           RunPipelinePollAction,
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, RunPipelineSuccess, executionStateCtx.Channel)
	assert.Equal(t, RunPipelinePayloadType, executionStateCtx.Type)
	require.NotEmpty(t, executionStateCtx.Payloads)
}

func Test__RunPipeline__Poll_ReschedulesOnAPIFailure(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader(`{"message":"temporary outage"}`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{ExecutionID: "exec-1", PipelineIdentifier: "deploy-prod", Status: "running"}}
	requestCtx := &contexts.RequestContext{}

	err := component.HandleAction(core.ActionContext{
		Name:           RunPipelinePollAction,
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, RunPipelinePollAction, requestCtx.Action)
	assert.Equal(t, RunPipelinePollInterval, requestCtx.Duration)
	assert.False(t, executionStateCtx.IsFinished())

	metadata := RunPipelineExecutionMetadata{}
	require.NoError(t, mapToStruct(metadataCtx.Get(), &metadata))
	assert.Equal(t, 1, metadata.PollErrorCount)
}

func Test__RunPipeline__Poll_ReschedulesWhenNotTerminal(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`
				{"data":{"content":[{"planExecutionId":"exec-1","pipelineIdentifier":"deploy-prod","status":"RUNNING"}]}}
			`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{ExecutionID: "exec-1", PipelineIdentifier: "deploy-prod", Status: "running"}}
	requestCtx := &contexts.RequestContext{}

	err := component.HandleAction(core.ActionContext{
		Name:           RunPipelinePollAction,
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, RunPipelinePollAction, requestCtx.Action)
	assert.Equal(t, RunPipelinePollInterval, requestCtx.Duration)
	assert.False(t, executionStateCtx.IsFinished())
}

func Test__RunPipeline__Poll_EmitsFailed(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`
				{"data":{"content":[{"planExecutionId":"exec-1","pipelineIdentifier":"deploy-prod","status":"FAILED","planExecutionUrl":"https://app.harness.io/executions/exec-1"}]}}
			`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{ExecutionID: "exec-1", PipelineIdentifier: "deploy-prod", Status: "running"}}

	err := component.HandleAction(core.ActionContext{
		Name:           RunPipelinePollAction,
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, RunPipelineFailed, executionStateCtx.Channel)
	assert.Equal(t, RunPipelinePayloadType, executionStateCtx.Type)
	require.NotEmpty(t, executionStateCtx.Payloads)
	payloadWrapper, ok := executionStateCtx.Payloads[0].(map[string]any)
	require.True(t, ok)
	payload, ok := payloadWrapper["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "failed", payload["status"])
}

func Test__RunPipeline__Poll_EmitsFailedAfterMaxPollErrors(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader(`{"message":"temporary outage"}`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunPipelineExecutionMetadata{
			ExecutionID:        "exec-1",
			PipelineIdentifier: "deploy-prod",
			Status:             "running",
			PollErrorCount:     RunPipelineMaxPollErrors - 1,
		},
	}

	err := component.HandleAction(core.ActionContext{
		Name:           RunPipelinePollAction,
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, RunPipelineFailed, executionStateCtx.Channel)
	assert.Equal(t, RunPipelinePayloadType, executionStateCtx.Type)
	require.NotEmpty(t, executionStateCtx.Payloads)
}

func Test__RunPipeline__Poll_ParsesNumericTimestamps(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`
				{"data":{"content":[{"planExecutionId":"exec-1","pipelineIdentifier":"deploy-prod","status":"SUCCEEDED","startTs":1771083861471,"endTs":1771083876397}]}}
			`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{ExecutionID: "exec-1", PipelineIdentifier: "deploy-prod", Status: "running"}}

	err := component.HandleAction(core.ActionContext{
		Name:           RunPipelinePollAction,
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token",
			"accountId": "acc",
		}},
	})

	require.NoError(t, err)
	metadata := RunPipelineExecutionMetadata{}
	require.NoError(t, mapToStruct(metadataCtx.Get(), &metadata))
	assert.Equal(t, "1771083861471", metadata.StartedAt)
	assert.Equal(t, "1771083876397", metadata.EndedAt)
}

func mapToStruct(input any, output any) error {
	data, err := jsonMarshal(input)
	if err != nil {
		return err
	}
	return jsonUnmarshal(data, output)
}

func jsonMarshal(input any) ([]byte, error) {
	return json.Marshal(input)
}

func jsonUnmarshal(data []byte, output any) error {
	return json.Unmarshal(data, output)
}
