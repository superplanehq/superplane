package harness

import (
	"encoding/json"
	"fmt"
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
		Configuration: RunPipelineSpec{
			OrgID:              "default",
			ProjectID:          "default_project",
			PipelineIdentifier: "deploy-prod",
			Ref:                "refs/heads/main",
			InputYAML:          "foo: bar",
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
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

func Test__RunPipeline__Execute_NestedPlanExecutionUUID(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"status":"SUCCESS","data":{"planExecution":{"uuid":"exec-nested-1","metadata":{"executionUuid":"exec-nested-meta"}}}}`,
			)),
		}},
	}

	metadataCtx := &contexts.MetadataContext{}
	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestCtx := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: RunPipelineSpec{
			OrgID:              "default",
			ProjectID:          "default_project",
			PipelineIdentifier: "deploy-prod",
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
		}},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "poll", requestCtx.Action)
	assert.Equal(t, "exec-nested-1", executionStateCtx.KVs["execution"])

	metadata := RunPipelineExecutionMetadata{}
	require.NoError(t, mapToStruct(metadataCtx.Get(), &metadata))
	assert.Equal(t, "exec-nested-1", metadata.ExecutionID)
}

func Test__RunPipeline__Setup(t *testing.T) {
	component := &RunPipeline{}

	t.Run("valid configuration", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
		}}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"organization":{"identifier":"default","name":"Default"}}]}}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"projectResponse":{"project":{"identifier":"default_project","name":"Default Project"}}}]}}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"yamlPipeline":"pipeline:\n  identifier: deploy-prod\n"}}`,
					)),
				},
			},
		}
		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineSpec{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy-prod",
				Ref:                "refs/heads/main",
			},
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		requestConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "deploy-prod", requestConfig.PipelineIdentifier)
		assert.Equal(t, "default", requestConfig.OrgID)
		assert.Equal(t, "default_project", requestConfig.ProjectID)
	})

	t.Run("invalid project selection fails setup", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
		}}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"organization":{"identifier":"default","name":"Default"}}]}}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"data":{"content":[{"projectResponse":{"project":{"identifier":"other_project","name":"Other"}}}]}}`,
					)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineSpec{
				OrgID:              "default",
				ProjectID:          "default_project",
				PipelineIdentifier: "deploy-prod",
				Ref:                "refs/heads/main",
			},
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.ErrorContains(t, err, `project "default_project" not found or inaccessible in organization "default"`)
		require.Empty(t, integrationCtx.WebhookRequests)
	})

	t.Run("missing pipeline identifier", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineSpec{
				OrgID:     "default",
				ProjectID: "default_project",
			},
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
		Configuration: RunPipelineSpec{
			OrgID:              "default",
			ProjectID:          "default_project",
			PipelineIdentifier: "deploy-prod",
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
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
		Configuration: RunPipelineSpec{
			OrgID:              "default",
			ProjectID:          "default_project",
			PipelineIdentifier: "deploy-prod",
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
		}},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
	})

	require.ErrorContains(t, err, "missing execution id")
	assert.Empty(t, requestCtx.Action)
	assert.Empty(t, executionStateCtx.KVs["execution"])
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
		Configuration:  RunPipelineSpec{OrgID: "default", ProjectID: "default_project", PipelineIdentifier: "deploy-prod"},
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
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
		Configuration:  RunPipelineSpec{OrgID: "default", ProjectID: "default_project", PipelineIdentifier: "deploy-prod"},
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
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
		Configuration:  RunPipelineSpec{OrgID: "default", ProjectID: "default_project", PipelineIdentifier: "deploy-prod"},
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
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
		Configuration:  RunPipelineSpec{OrgID: "default", ProjectID: "default_project", PipelineIdentifier: "deploy-prod"},
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
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

func Test__RunPipeline__Poll_EmitsFailedForErrored(t *testing.T) {
	component := &RunPipeline{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`
				{"data":{"content":[{"planExecutionId":"exec-1","pipelineIdentifier":"deploy-prod","status":"ERRORED","planExecutionUrl":"https://app.harness.io/executions/exec-1"}]}}
			`)),
		}},
	}

	executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{ExecutionID: "exec-1", PipelineIdentifier: "deploy-prod", Status: "running"}}

	err := component.HandleAction(core.ActionContext{
		Name:           RunPipelinePollAction,
		Configuration:  RunPipelineSpec{OrgID: "default", ProjectID: "default_project", PipelineIdentifier: "deploy-prod"},
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
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
		Configuration:  RunPipelineSpec{OrgID: "default", ProjectID: "default_project", PipelineIdentifier: "deploy-prod"},
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
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
		Configuration:  RunPipelineSpec{OrgID: "default", ProjectID: "default_project", PipelineIdentifier: "deploy-prod"},
		HTTP:           httpCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       &contexts.RequestContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.acc.test",
		}},
	})

	require.NoError(t, err)
	metadata := RunPipelineExecutionMetadata{}
	require.NoError(t, mapToStruct(metadataCtx.Get(), &metadata))
	assert.Equal(t, "1771083861471", metadata.StartedAt)
	assert.Equal(t, "1771083876397", metadata.EndedAt)
}

func Test__RunPipeline__HandleWebhook(t *testing.T) {
	component := &RunPipeline{}

	t.Run("invalid webhook secret -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer wrong")

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-1","pipelineIdentifier":"deploy","nodeStatus":"FAILED"}}`),
			Webhook: &contexts.NodeWebhookContext{Secret: "expected"},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid webhook authorization")
	})

	t.Run("ignores non-pipeline-completed webhook events", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{ExecutionID: "exec-1"}}

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"StageEnd","eventData":{"planExecutionId":"exec-1","nodeStatus":"FAILED"}}`),
			Webhook: &contexts.NodeWebhookContext{Secret: "expected"},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       metadata,
					ExecutionState: executionState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.False(t, executionState.IsFinished())
	})

	t.Run("ignores execution IDs not started by SuperPlane", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-unknown","nodeStatus":"FAILED"}}`),
			Webhook: &contexts.NodeWebhookContext{Secret: "expected"},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, fmt.Errorf("not found")
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("emits success for completed status", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{
			ExecutionID:        "exec-1",
			PipelineIdentifier: "deploy-prod",
			Status:             "running",
		}}

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body: []byte(
				`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-1","pipelineIdentifier":"deploy-prod","nodeStatus":"COMPLETED","executionUrl":"https://app.harness.io/executions/exec-1","startTs":"1771083861471","endTs":"1771083876397"}}`,
			),
			Webhook: &contexts.NodeWebhookContext{Secret: "expected"},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       metadata,
					ExecutionState: executionState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, RunPipelineSuccess, executionState.Channel)
		assert.Equal(t, RunPipelinePayloadType, executionState.Type)
		require.NotEmpty(t, executionState.Payloads)

		payloadWrapper, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := payloadWrapper["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "succeeded", payload["status"])
		assert.Equal(t, "https://app.harness.io/executions/exec-1", payload["planExecutionUrl"])

		stored := RunPipelineExecutionMetadata{}
		require.NoError(t, mapToStruct(metadata.Get(), &stored))
		assert.Equal(t, "succeeded", canonicalStatus(stored.Status))
		assert.Equal(t, "1771083861471", stored.StartedAt)
		assert.Equal(t, "1771083876397", stored.EndedAt)
	})

	t.Run("emits failed for aborted status", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{Metadata: RunPipelineExecutionMetadata{
			ExecutionID:        "exec-2",
			PipelineIdentifier: "deploy-prod",
			Status:             "running",
		}}

		code, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body: []byte(
				`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-2","pipelineIdentifier":"deploy-prod","nodeStatus":"ABORTED"}}`,
			),
			Webhook: &contexts.NodeWebhookContext{Secret: "expected"},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       metadata,
					ExecutionState: executionState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, RunPipelineFailed, executionState.Channel)
		assert.Equal(t, RunPipelinePayloadType, executionState.Type)
		require.NotEmpty(t, executionState.Payloads)

		payloadWrapper, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := payloadWrapper["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "aborted", payload["status"])
	})
}

func mapToStruct(input any, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}
