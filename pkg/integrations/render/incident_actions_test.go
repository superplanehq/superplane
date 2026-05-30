package render

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

func Test__Render_ListDeploys__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`[{"cursor":"a","deploy":{"id":"dep-1","status":"live","createdAt":"2026-05-30T12:00:00Z"}}]`,
			)),
		}},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := (&ListDeploys{}).Execute(core.ExecutionContext{
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
		ExecutionState: executionState,
		Configuration:  map[string]any{"service": "srv-123", "statuses": []string{"live"}, "limit": 5},
	})

	require.NoError(t, err)
	assert.Equal(t, ListDeploysPayloadType, executionState.Type)
	data := readMap(readMap(executionState.Payloads[0])["data"])
	assert.Equal(t, "srv-123", data["serviceId"])
	assert.Equal(t, 1, data["count"])
	assert.NotNil(t, data["latestSuccessful"])

	require.Len(t, httpCtx.Requests, 1)
	request := httpCtx.Requests[0]
	assert.Equal(t, http.MethodGet, request.Method)
	assert.Equal(t, "/v1/services/srv-123/deploys", request.URL.Path)
	assert.Equal(t, "5", request.URL.Query().Get("limit"))
	assert.Equal(t, "live", request.URL.Query().Get("status"))
}

func Test__Render_GetMetrics__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"labels":[{"field":"resource","value":"srv-123"}],"unit":"%","values":[{"timestamp":"2026-05-30T12:00:00Z","value":40},{"timestamp":"2026-05-30T12:01:00Z","value":85}]}]`,
				)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"labels":[{"field":"resource","value":"srv-123"}],"unit":"%","values":[{"timestamp":"2026-05-30T12:00:00Z","value":70}]}]`,
				)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := (&GetMetrics{}).Execute(core.ExecutionContext{
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
		ExecutionState: executionState,
		Configuration: map[string]any{
			"resources":         []string{"srv-123"},
			"metricTypes":       []string{"cpu", "memory"},
			"resolutionSeconds": 60,
			"aggregationMethod": "AVG",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, GetMetricsPayloadType, executionState.Type)
	data := readMap(readMap(executionState.Payloads[0])["data"])
	summaries := readMap(data["summaries"])
	cpu := readMap(summaries["cpu"])
	assert.Equal(t, 85.0, cpu["latest"])
	assert.Equal(t, 62.5, cpu["avg"])
	assert.Equal(t, 85.0, cpu["max"])

	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "/v1/metrics/cpu", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, "/v1/metrics/memory", httpCtx.Requests[1].URL.Path)
	assert.Equal(t, "srv-123", httpCtx.Requests[0].URL.Query().Get("resource"))
}

func Test__Render_ListLogs__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"hasMore":false,"logs":[{"timestamp":"2026-05-30T12:00:00Z","level":"error","message":"timeout"},{"timestamp":"2026-05-30T12:01:00Z","level":"info","message":"ok"}]}`,
			)),
		}},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := (&ListLogs{}).Execute(core.ExecutionContext{
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "rnd_test"},
			Metadata:      Metadata{Workspace: &WorkspaceMetadata{ID: "usr-123"}},
		},
		ExecutionState: executionState,
		Configuration:  map[string]any{"resources": []string{"srv-123"}, "levels": []string{"error"}, "limit": 20},
	})

	require.NoError(t, err)
	assert.Equal(t, ListLogsPayloadType, executionState.Type)
	data := readMap(readMap(executionState.Payloads[0])["data"])
	assert.Equal(t, 2, data["count"])
	assert.Equal(t, 1, data["errorCount"])

	require.Len(t, httpCtx.Requests, 1)
	request := httpCtx.Requests[0]
	assert.Equal(t, "/v1/logs", request.URL.Path)
	assert.Equal(t, "usr-123", request.URL.Query().Get("ownerId"))
	assert.Equal(t, "srv-123", request.URL.Query().Get("resource"))
	assert.Equal(t, "error", request.URL.Query().Get("level"))
}

func Test__Render_UpdateService__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"srv-123","name":"api","autoDeploy":"no"}`)),
		}},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := (&UpdateService{}).Execute(core.ExecutionContext{
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
		ExecutionState: executionState,
		Configuration:  map[string]any{"service": "srv-123", "autoDeploy": "no"},
	})

	require.NoError(t, err)
	assert.Equal(t, UpdateServicePayloadType, executionState.Type)
	data := readMap(readMap(executionState.Payloads[0])["data"])
	assert.Equal(t, "no", data["autoDeploy"])

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	var requestBody map[string]any
	require.NoError(t, json.Unmarshal(body, &requestBody))
	assert.Equal(t, "no", requestBody["autoDeploy"])
	assert.Equal(t, http.MethodPatch, httpCtx.Requests[0].Method)
}

func Test__Render_UpdateAutoscaling__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"enabled":true,"min":1,"max":3}`)),
		}},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := (&UpdateAutoscaling{}).Execute(core.ExecutionContext{
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
		ExecutionState: executionState,
		Configuration: map[string]any{
			"service": "srv-123", "enabled": true, "minInstances": 1, "maxInstances": 3, "cpuPercent": 70, "memoryPercent": 75,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, UpdateAutoscalingPayloadType, executionState.Type)
	assert.Equal(t, http.MethodPut, httpCtx.Requests[0].Method)
	assert.Equal(t, "/v1/services/srv-123/autoscaling", httpCtx.Requests[0].URL.Path)
}

func Test__Render_CreateAndGetJob__Execute(t *testing.T) {
	t.Run("create job", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":"job-123","serviceId":"srv-123","startCommand":"python manage.py check","status":"pending"}`)),
		}}}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&CreateJob{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"service": "srv-123", "startCommand": "python manage.py check"},
		})

		require.NoError(t, err)
		assert.Equal(t, CreateJobPayloadType, executionState.Type)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "/v1/services/srv-123/jobs", httpCtx.Requests[0].URL.Path)
	})

	t.Run("get job", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"job-123","serviceId":"srv-123","status":"succeeded"}`)),
		}}}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&GetJob{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"service": "srv-123", "jobId": "job-123"},
		})

		require.NoError(t, err)
		assert.Equal(t, GetJobPayloadType, executionState.Type)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/v1/services/srv-123/jobs/job-123", httpCtx.Requests[0].URL.Path)
	})
}
