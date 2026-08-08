package waitforendpoint

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type sequenceHTTPContext struct {
	responses []*http.Response
	errors    []error
	requests  []*http.Request
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func (c *sequenceHTTPContext) Do(request *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, request)
	if len(c.errors) > 0 {
		err := c.errors[0]
		c.errors = c.errors[1:]
		if err != nil {
			return nil, err
		}
	}
	if len(c.responses) == 0 {
		return nil, errors.New("no response mocked")
	}

	response := c.responses[0]
	c.responses = c.responses[1:]
	response.Request = request
	return response, nil
}

func endpointResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func endpointConfig(url string) map[string]any {
	return map[string]any{
		"url":                   url,
		"method":                http.MethodGet,
		"expectedStatus":        "2xx",
		"intervalSeconds":       10,
		"timeoutSeconds":        60,
		"requestTimeoutSeconds": 5,
		"maxResponseBytes":      4096,
	}
}

func executionContext(
	config map[string]any,
	httpCtx core.HTTPContext,
) (core.ExecutionContext, *contexts.ExecutionStateContext, *contexts.MetadataContext, *contexts.RequestContext) {
	state := &contexts.ExecutionStateContext{}
	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}
	return core.ExecutionContext{
		Logger:         log.NewEntry(log.StandardLogger()),
		Configuration:  config,
		HTTP:           httpCtx,
		Metadata:       metadata,
		ExecutionState: state,
		Requests:       requests,
	}, state, metadata, requests
}

func hookContext(
	execution core.ExecutionContext,
	requests *contexts.RequestContext,
) core.ActionHookContext {
	return core.ActionHookContext{
		Name:           checkEndpointHook,
		Logger:         execution.Logger,
		Configuration:  execution.Configuration,
		HTTP:           execution.HTTP,
		Metadata:       execution.Metadata,
		ExecutionState: execution.ExecutionState,
		Requests:       requests,
		Secrets:        execution.Secrets,
	}
}

func metadataFrom(t *testing.T, metadataCtx *contexts.MetadataContext) Metadata {
	t.Helper()

	var metadata Metadata
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
	return metadata
}

func outputData(t *testing.T, state *contexts.ExecutionStateContext) map[string]any {
	t.Helper()

	require.Len(t, state.Payloads, 1)
	payload, ok := state.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	return data
}

func TestWaitForEndpointSetup(t *testing.T) {
	component := &WaitForEndpoint{}
	require.NoError(t, component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"url": "https://service.example.com/ready",
		},
	}))

	tests := []struct {
		name    string
		update  func(map[string]any)
		message string
	}{
		{
			name:    "missing URL",
			update:  func(config map[string]any) { delete(config, "url") },
			message: "url is required",
		},
		{
			name:    "unsupported method",
			update:  func(config map[string]any) { config["method"] = http.MethodPost },
			message: "method must be GET or HEAD",
		},
		{
			name:    "invalid expected status",
			update:  func(config map[string]any) { config["expectedStatus"] = "20x" },
			message: "invalid HTTP status matcher",
		},
		{
			name:    "invalid condition",
			update:  func(config map[string]any) { config["condition"] = "body.status ==" },
			message: "invalid readiness condition",
		},
		{
			name:    "response limit too large",
			update:  func(config map[string]any) { config["maxResponseBytes"] = maxMaxResponseBytes + 1 },
			message: "maximum response bytes must be between",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := endpointConfig("https://service.example.com/ready")
			test.update(config)
			err := component.Setup(core.SetupContext{Configuration: config})
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.message)
		})
	}
}

func TestWaitForEndpointExecuteReadyImmediately(t *testing.T) {
	now := time.Now()
	component := &WaitForEndpoint{now: func() time.Time { return now }}
	httpCtx := &sequenceHTTPContext{
		responses: []*http.Response{endpointResponse(http.StatusOK, `{"status":"ready"}`)},
	}
	config := endpointConfig("https://service.example.com/ready")
	config["condition"] = `body.status == "ready"`
	execution, state, metadataCtx, requests := executionContext(config, httpCtx)

	require.NoError(t, component.Execute(execution))
	assert.True(t, state.Finished)
	assert.Equal(t, ReadyOutputChannel, state.Channel)
	assert.Equal(t, ReadyPayloadType, state.Type)
	assert.Empty(t, requests.Action)

	metadata := metadataFrom(t, metadataCtx)
	assert.Equal(t, 1, metadata.Attempts)
	assert.Equal(t, http.StatusOK, *metadata.LastStatus)

	data := outputData(t, state)
	assert.Equal(t, 1, data["attempts"])
	assert.Equal(t, http.StatusOK, *data["status"].(*int))
}

func TestWaitForEndpointExecuteSchedulesWhenNotReady(t *testing.T) {
	now := time.Now()
	component := &WaitForEndpoint{now: func() time.Time { return now }}
	httpCtx := &sequenceHTTPContext{
		responses: []*http.Response{endpointResponse(http.StatusServiceUnavailable, `{"status":"starting"}`)},
	}
	execution, state, metadataCtx, requests := executionContext(
		endpointConfig("https://service.example.com/ready"),
		httpCtx,
	)

	require.NoError(t, component.Execute(execution))
	assert.False(t, state.Finished)
	assert.Equal(t, checkEndpointHook, requests.Action)
	assert.Equal(t, 10*time.Second, requests.Duration)

	metadata := metadataFrom(t, metadataCtx)
	assert.Equal(t, 1, metadata.Attempts)
	assert.Equal(t, http.StatusServiceUnavailable, *metadata.LastStatus)
	assert.NotEmpty(t, metadata.NextAttemptAt)
}

func TestWaitForEndpointExecuteUsesSchedulerMinimumNearDeadline(t *testing.T) {
	startedAt := time.Now()
	times := []time.Time{startedAt, startedAt, startedAt.Add(9500 * time.Millisecond)}
	component := &WaitForEndpoint{now: func() time.Time {
		now := times[0]
		times = times[1:]
		return now
	}}
	httpCtx := &sequenceHTTPContext{
		responses: []*http.Response{endpointResponse(http.StatusServiceUnavailable, "starting")},
	}
	config := endpointConfig("https://service.example.com/ready")
	config["timeoutSeconds"] = 10
	execution, state, _, requests := executionContext(config, httpCtx)

	require.NoError(t, component.Execute(execution))
	assert.False(t, state.Finished)
	assert.Equal(t, time.Second, requests.Duration)
}

func TestWaitForEndpointHandleHookSucceedsAfterNetworkFailure(t *testing.T) {
	now := time.Now()
	component := &WaitForEndpoint{now: func() time.Time { return now }}
	httpCtx := &sequenceHTTPContext{
		errors: []error{errors.New("connection refused"), nil},
		responses: []*http.Response{
			endpointResponse(http.StatusOK, `{"status":"ready"}`),
		},
	}
	execution, state, metadataCtx, _ := executionContext(
		endpointConfig("https://service.example.com/ready"),
		httpCtx,
	)

	require.NoError(t, component.Execute(execution))
	metadata := metadataFrom(t, metadataCtx)
	assert.Equal(t, 1, metadata.Attempts)
	assert.Contains(t, metadata.LastError, "connection refused")

	now = now.Add(10 * time.Second)
	require.NoError(t, component.HandleHook(hookContext(execution, &contexts.RequestContext{})))
	assert.True(t, state.Finished)
	assert.Equal(t, ReadyOutputChannel, state.Channel)

	metadata = metadataFrom(t, metadataCtx)
	assert.Equal(t, 2, metadata.Attempts)
	assert.Empty(t, metadata.LastError)
}

func TestWaitForEndpointExecuteRetriesResponseReadFailure(t *testing.T) {
	readErr := errors.New("connection reset while reading response")
	httpCtx := &sequenceHTTPContext{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(errorReader{err: readErr}),
		}},
	}
	execution, state, metadataCtx, requests := executionContext(
		endpointConfig("https://service.example.com/ready"),
		httpCtx,
	)

	require.NoError(t, (&WaitForEndpoint{}).Execute(execution))
	assert.False(t, state.Finished)
	assert.Equal(t, checkEndpointHook, requests.Action)
	assert.Contains(t, metadataFrom(t, metadataCtx).LastError, readErr.Error())
}

func TestWaitForEndpointHandleHookEmitsTimeout(t *testing.T) {
	now := time.Now()
	component := &WaitForEndpoint{now: func() time.Time { return now }}
	httpCtx := &sequenceHTTPContext{
		responses: []*http.Response{endpointResponse(http.StatusServiceUnavailable, "starting")},
	}
	config := endpointConfig("https://service.example.com/ready")
	config["timeoutSeconds"] = 10
	execution, state, _, _ := executionContext(config, httpCtx)

	require.NoError(t, component.Execute(execution))
	now = now.Add(10 * time.Second)

	require.NoError(t, component.HandleHook(hookContext(execution, &contexts.RequestContext{})))
	assert.True(t, state.Finished)
	assert.Equal(t, TimeoutOutputChannel, state.Channel)
	assert.Equal(t, TimeoutPayloadType, state.Type)

	data := outputData(t, state)
	assert.Equal(t, "deadline_exceeded", data["reason"])
	assert.Equal(t, 1, data["attempts"])
}

func TestWaitForEndpointExecuteRejectsPolicyViolation(t *testing.T) {
	httpCtx, err := registry.NewHTTPContext(registry.HTTPOptions{
		BlockedHosts: []string{"example.com"},
	})
	require.NoError(t, err)

	execution, state, _, requests := executionContext(endpointConfig("https://example.com/ready"), httpCtx)
	err = (&WaitForEndpoint{}).Execute(execution)
	require.Error(t, err)

	var policyErr *registry.HTTPPolicyError
	assert.True(t, errors.As(err, &policyErr))
	assert.False(t, state.Finished)
	assert.Empty(t, requests.Action)
}

func TestWaitForEndpointExecuteRejectsOversizedResponse(t *testing.T) {
	config := endpointConfig("https://service.example.com/ready")
	config["maxResponseBytes"] = minMaxResponseBytes
	httpCtx := &sequenceHTTPContext{
		responses: []*http.Response{endpointResponse(http.StatusOK, strings.Repeat("a", minMaxResponseBytes+1))},
	}
	execution, state, _, requests := executionContext(config, httpCtx)

	err := (&WaitForEndpoint{}).Execute(execution)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response exceeds configured maximum size")
	assert.False(t, state.Finished)
	assert.Empty(t, requests.Action)
}

func TestWaitForEndpointExecuteRejectsRegistryResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Length", "2048")
		_, _ = response.Write([]byte(strings.Repeat("a", 2048)))
	}))
	t.Cleanup(server.Close)

	httpCtx, err := registry.NewHTTPContext(registry.HTTPOptions{MaxResponseBytes: 1024})
	require.NoError(t, err)

	config := endpointConfig(server.URL)
	config["maxResponseBytes"] = 4096
	execution, state, _, requests := executionContext(config, httpCtx)

	err = (&WaitForEndpoint{}).Execute(execution)
	require.Error(t, err)
	var responseTooLargeErr *registry.HTTPResponseTooLargeError
	assert.True(t, errors.As(err, &responseTooLargeErr))
	assert.False(t, state.Finished)
	assert.Empty(t, requests.Action)
}

func TestWaitForEndpointTruncatesBodyToEventPayloadLimit(t *testing.T) {
	t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "4096")
	config := endpointConfig("https://service.example.com/ready")
	config["maxResponseBytes"] = 16 * 1024
	httpCtx := &sequenceHTTPContext{
		responses: []*http.Response{endpointResponse(http.StatusOK, strings.Repeat("a", 8*1024))},
	}
	execution, state, _, _ := executionContext(config, httpCtx)

	require.NoError(t, (&WaitForEndpoint{}).Execute(execution))
	data := outputData(t, state)
	body, ok := data["body"].(string)
	require.True(t, ok)
	assert.True(t, strings.Contains(body, "[truncated]") || strings.Contains(body, "omitted"))

	event, err := json.Marshal(state.Payloads[0])
	require.NoError(t, err)
	assert.LessOrEqual(t, len(event), 4096)
}

func TestWaitForEndpointFinishedHookIsNoOp(t *testing.T) {
	component := &WaitForEndpoint{}
	state := &contexts.ExecutionStateContext{Finished: true}
	requests := &contexts.RequestContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name:           checkEndpointHook,
		ExecutionState: state,
		Requests:       requests,
	})
	require.NoError(t, err)
	assert.Empty(t, requests.Action)
}

func TestWaitForEndpointUnknownHook(t *testing.T) {
	err := (&WaitForEndpoint{}).HandleHook(core.ActionHookContext{Name: "unknown"})
	assert.EqualError(t, err, "unknown hook: unknown")
}
