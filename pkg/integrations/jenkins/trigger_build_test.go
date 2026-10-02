package jenkins

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func crumbDisabledResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func crumbEnabledResponse(field, crumb string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"crumbRequestField":"` + field + `","crumb":"` + crumb + `"}`)),
	}
}

func createdResponse(location string) *http.Response {
	res := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     http.Header{},
	}
	res.Header.Set("Location", location)
	return res
}

func executeTriggerBuild(t *testing.T, config map[string]any, httpContext *contexts.HTTPContext) (*contexts.ExecutionStateContext, error) {
	t.Helper()

	integrationCtx := &contexts.IntegrationContext{Configuration: validConfig()}
	executionState := &contexts.ExecutionStateContext{}

	tb := &TriggerBuild{}
	err := tb.Execute(core.ExecutionContext{
		Configuration:  config,
		HTTP:           httpContext,
		Integration:    integrationCtx,
		ExecutionState: executionState,
	})

	return executionState, err
}

func Test__TriggerBuild__Execute(t *testing.T) {
	t.Run("no parameters -> POST to /job/{job}/build", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				crumbDisabledResponse(),
				createdResponse("https://jenkins.example.com/queue/item/1/"),
			},
		}

		_, err := executeTriggerBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		buildReq := httpContext.Requests[1]
		assert.Equal(t, http.MethodPost, buildReq.Method)
		assert.Equal(t, "/job/my-job/build", buildReq.URL.Path)
	})

	t.Run("with parameters -> POST to /job/{job}/buildWithParameters", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				crumbDisabledResponse(),
				createdResponse("https://jenkins.example.com/queue/item/2/"),
			},
		}

		config := map[string]any{
			"jobName": "my-job",
			"parameters": []any{
				map[string]any{"name": "ENV", "value": "staging"},
			},
		}

		_, err := executeTriggerBuild(t, config, httpContext)

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		buildReq := httpContext.Requests[1]
		assert.Equal(t, http.MethodPost, buildReq.Method)
		assert.Equal(t, "/job/my-job/buildWithParameters", buildReq.URL.Path)

		body, err := io.ReadAll(buildReq.Body)
		require.NoError(t, err)
		values, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		assert.Equal(t, "staging", values.Get("ENV"))
	})

	t.Run("201 + Location parsed into emitted payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				crumbDisabledResponse(),
				createdResponse("https://jenkins.example.com/queue/item/42/"),
			},
		}

		executionState, err := executeTriggerBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "my-job", data["jobName"])
		assert.Equal(t, "https://jenkins.example.com/queue/item/42/", data["queueUrl"])
	})

	t.Run("crumb sent when present", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				crumbEnabledResponse("Jenkins-Crumb", "abc123"),
				createdResponse("https://jenkins.example.com/queue/item/3/"),
			},
		}

		_, err := executeTriggerBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "abc123", httpContext.Requests[1].Header.Get("Jenkins-Crumb"))
	})

	t.Run("crumb skipped when crumb issuer is disabled (404)", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				crumbDisabledResponse(),
				createdResponse("https://jenkins.example.com/queue/item/4/"),
			},
		}

		_, err := executeTriggerBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		assert.Empty(t, httpContext.Requests[1].Header.Get("Jenkins-Crumb"))
	})

	t.Run("non-201 response -> error, nothing emitted", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				crumbDisabledResponse(),
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("internal error")),
				},
			},
		}

		executionState, err := executeTriggerBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.Error(t, err)
		assert.False(t, executionState.Finished)
	})

	t.Run("404 on build -> job not found error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				crumbDisabledResponse(),
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		_, err := executeTriggerBuild(t, map[string]any{"jobName": "missing-job"}, httpContext)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `"missing-job" not found`)
	})

	t.Run("crumb request fails at transport level -> error, build never attempted", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{}}

		executionState, err := executeTriggerBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "CSRF crumb")
		assert.False(t, executionState.Finished)
		assert.Len(t, httpContext.Requests, 1)
	})

	t.Run("crumb issuer returns error status -> error, build never attempted", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("crumb issuer exploded")),
				},
			},
		}

		executionState, err := executeTriggerBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "CSRF crumb")
		assert.False(t, executionState.Finished)
		assert.Len(t, httpContext.Requests, 1)
	})

	t.Run("crumb issuer returns malformed JSON -> error, build never attempted", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("not json")),
				},
			},
		}

		executionState, err := executeTriggerBuild(t, map[string]any{"jobName": "my-job"}, httpContext)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "CSRF crumb")
		assert.False(t, executionState.Finished)
		assert.Len(t, httpContext.Requests, 1)
	})
}

func Test__TriggerBuild__Setup(t *testing.T) {
	t.Run("missing jobName -> error", func(t *testing.T) {
		tb := &TriggerBuild{}
		err := tb.Setup(core.SetupContext{Configuration: map[string]any{}})
		require.Error(t, err)
	})

	t.Run("valid jobName -> no error", func(t *testing.T) {
		tb := &TriggerBuild{}
		err := tb.Setup(core.SetupContext{Configuration: map[string]any{"jobName": "my-job"}})
		require.NoError(t, err)
	})
}

func Test__buildParameterMap(t *testing.T) {
	result := buildParameterMap([]BuildParameter{
		{Name: "env", Value: "production"},
		{Name: "version", Value: "1.0.0"},
	})

	assert.Equal(t, "production", result["env"])
	assert.Equal(t, "1.0.0", result["version"])
	assert.Len(t, result, 2)
}
