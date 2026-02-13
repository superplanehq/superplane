package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetPipeline__Execute(t *testing.T) {
	component := &GetPipeline{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project":    "123",
			"pipelineId": "1001",
		},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":    AuthTypePersonalAccessToken,
				"groupId":     "123",
				"accessToken": "pat",
				"baseUrl":     "https://gitlab.com",
			},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 1001,
					"iid": 73,
					"project_id": 123,
					"status": "running",
					"ref": "main"
				}`),
			},
		},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
	assert.Equal(t, "gitlab.pipeline", executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	payload := executionState.Payloads[0].(map[string]any)
	dataBytes, err := json.Marshal(payload["data"])
	require.NoError(t, err)

	var pipeline Pipeline
	require.NoError(t, json.Unmarshal(dataBytes, &pipeline))
	assert.Equal(t, 1001, pipeline.ID)
	assert.Equal(t, "running", pipeline.Status)
}

func Test__GetLatestPipeline__Execute(t *testing.T) {
	component := &GetLatestPipeline{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project": "123",
			"ref":     "refs/heads/main",
		},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":    AuthTypePersonalAccessToken,
				"groupId":     "123",
				"accessToken": "pat",
				"baseUrl":     "https://gitlab.com",
			},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 1002,
					"iid": 74,
					"project_id": 123,
					"status": "success",
					"ref": "main"
				}`),
			},
		},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
	assert.Equal(t, "gitlab.pipeline", executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	payload := executionState.Payloads[0].(map[string]any)
	dataBytes, err := json.Marshal(payload["data"])
	require.NoError(t, err)

	var pipeline Pipeline
	require.NoError(t, json.Unmarshal(dataBytes, &pipeline))
	assert.Equal(t, 1002, pipeline.ID)
	assert.Equal(t, "success", pipeline.Status)
}

func Test__GetTestReportSummary__Execute(t *testing.T) {
	component := &GetTestReportSummary{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project":    "123",
			"pipelineId": "1002",
		},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":    AuthTypePersonalAccessToken,
				"groupId":     "123",
				"accessToken": "pat",
				"baseUrl":     "https://gitlab.com",
			},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"total": {"count": 40, "success": 39, "failed": 1},
					"test_suites": [{"name": "rspec", "total_count": 40}]
				}`),
			},
		},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
	assert.Equal(t, "gitlab.testReportSummary", executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	payload := executionState.Payloads[0].(map[string]any)
	dataBytes, err := json.Marshal(payload["data"])
	require.NoError(t, err)

	var summary PipelineTestReportSummary
	require.NoError(t, json.Unmarshal(dataBytes, &summary))
	assert.Equal(t, 40.0, summary.Total["count"])
}
