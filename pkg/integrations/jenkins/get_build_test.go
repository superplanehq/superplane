package jenkins

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

func buildJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func executeGetBuild(t *testing.T, config map[string]any, httpContext *contexts.HTTPContext) (*contexts.ExecutionStateContext, error) {
	t.Helper()

	integrationCtx := &contexts.IntegrationContext{Configuration: validConfig()}
	executionState := &contexts.ExecutionStateContext{}

	g := &GetBuild{}
	err := g.Execute(core.ExecutionContext{
		Configuration:  config,
		HTTP:           httpContext,
		Integration:    integrationCtx,
		ExecutionState: executionState,
	})

	return executionState, err
}

func Test__GetBuild__Execute(t *testing.T) {
	t.Run("running build -> building true, result null", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				buildJSONResponse(`{"building":true,"result":null,"number":7,"url":"https://jenkins.example.com/job/my-job/7/","duration":0}`),
			},
		}

		executionState, err := executeGetBuild(t, map[string]any{"jobName": "my-job", "buildNumber": 7}, httpContext)

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "/job/my-job/7/api/json", httpContext.Requests[0].URL.Path)

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, true, data["building"])
		assert.Nil(t, data["result"])
		assert.Equal(t, 7, data["number"])
		assert.Equal(t, "https://jenkins.example.com/job/my-job/7/", data["url"])
		assert.Equal(t, int64(0), data["durationMs"])
	})

	t.Run("finished build (SUCCESS) -> building false, result SUCCESS", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				buildJSONResponse(`{"building":false,"result":"SUCCESS","number":8,"url":"https://jenkins.example.com/job/my-job/8/","duration":12345}`),
			},
		}

		executionState, err := executeGetBuild(t, map[string]any{"jobName": "my-job", "buildNumber": 8}, httpContext)

		require.NoError(t, err)
		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, false, data["building"])
		result := data["result"].(*string)
		require.NotNil(t, result)
		assert.Equal(t, "SUCCESS", *result)
		assert.Equal(t, int64(12345), data["durationMs"])
	})

	t.Run("finished build (FAILURE) -> building false, result FAILURE", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				buildJSONResponse(`{"building":false,"result":"FAILURE","number":9,"url":"https://jenkins.example.com/job/my-job/9/","duration":543}`),
			},
		}

		executionState, err := executeGetBuild(t, map[string]any{"jobName": "my-job", "buildNumber": 9}, httpContext)

		require.NoError(t, err)
		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, false, data["building"])
		result := data["result"].(*string)
		require.NotNil(t, result)
		assert.Equal(t, "FAILURE", *result)
	})

	t.Run("404 -> build not found error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))},
			},
		}

		executionState, err := executeGetBuild(t, map[string]any{"jobName": "my-job", "buildNumber": 999}, httpContext)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "999")
		assert.Contains(t, err.Error(), "not found")
		assert.False(t, executionState.Finished)
	})

	t.Run("non-200 response -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("internal error"))},
			},
		}

		executionState, err := executeGetBuild(t, map[string]any{"jobName": "my-job", "buildNumber": 1}, httpContext)

		require.Error(t, err)
		assert.False(t, executionState.Finished)
	})

	t.Run("malformed JSON body -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				buildJSONResponse("not json"),
			},
		}

		_, err := executeGetBuild(t, map[string]any{"jobName": "my-job", "buildNumber": 1}, httpContext)

		require.Error(t, err)
	})

	t.Run("missing buildNumber -> clean error, no request made", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		_, err := executeGetBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "buildNumber")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("invalid (negative) buildNumber -> clean error, no request made", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		_, err := executeGetBuild(t, map[string]any{"jobName": "my-job", "buildNumber": -1}, httpContext)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "buildNumber")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("missing jobName -> clean error, no request made", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		_, err := executeGetBuild(t, map[string]any{"buildNumber": 1}, httpContext)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "jobName")
		assert.Empty(t, httpContext.Requests)
	})
}

func Test__GetBuild__Setup(t *testing.T) {
	t.Run("missing jobName -> error", func(t *testing.T) {
		g := &GetBuild{}
		err := g.Setup(core.SetupContext{Configuration: map[string]any{"buildNumber": 1}})
		require.Error(t, err)
	})

	t.Run("missing buildNumber -> error", func(t *testing.T) {
		g := &GetBuild{}
		err := g.Setup(core.SetupContext{Configuration: map[string]any{"jobName": "my-job"}})
		require.Error(t, err)
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		g := &GetBuild{}
		err := g.Setup(core.SetupContext{Configuration: map[string]any{"jobName": "my-job", "buildNumber": 1}})
		require.NoError(t, err)
	})
}
