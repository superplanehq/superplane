package http

import (
	"context"
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
	"github.com/superplanehq/superplane/test/support/contexts"
)

func createExecutionContext(config map[string]any) (core.ExecutionContext, *contexts.ExecutionStateContext, *contexts.MetadataContext) {
	if _, ok := config["timeoutSeconds"]; !ok {
		config["timeoutSeconds"] = 1
	}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	return core.ExecutionContext{
		Logger:         log.NewEntry(log.StandardLogger()),
		Configuration:  config,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		HTTP:           &http.Client{},
	}, stateCtx, metadataCtx
}

func decodeMetadata(t *testing.T, metadataCtx *contexts.MetadataContext) Metadata {
	t.Helper()

	var metadata Metadata
	err := mapstructure.Decode(metadataCtx.Get(), &metadata)
	require.NoError(t, err)
	return metadata
}

func responsePayload(t *testing.T, stateCtx *contexts.ExecutionStateContext) map[string]any {
	t.Helper()

	require.Len(t, stateCtx.Payloads, 1)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	require.True(t, ok)

	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)

	return data
}

func retryConfig(url string) map[string]any {
	return map[string]any{
		"method":         "GET",
		"url":            url,
		"timeoutSeconds": 1,
		"retry": map[string]any{
			"enabled":         true,
			"strategy":        RetryStrategyFixed,
			"maxAttempts":     3,
			"intervalSeconds": 5,
		},
	}
}

type contextBoundReadCloser struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextBoundReadCloser) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}

	return r.reader.Read(p)
}

func (r *contextBoundReadCloser) Close() error {
	return nil
}

type contextBoundHTTPClient struct{}

func (c *contextBoundHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body: &contextBoundReadCloser{
			ctx:    req.Context(),
			reader: strings.NewReader(`{"hello":"world"}`),
		},
	}, nil
}

type sequenceHTTPClient struct {
	errors    []error
	responses []*http.Response
}

func (c *sequenceHTTPClient) Do(_ *http.Request) (*http.Response, error) {
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

	resp := c.responses[0]
	c.responses = c.responses[1:]
	return resp, nil
}

func newResponse(statusCode int, body string, headers http.Header) *http.Response {
	if headers == nil {
		headers = http.Header{}
	}

	return &http.Response{
		StatusCode: statusCode,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestHTTP__Setup__ValidConfigurations(t *testing.T) {
	h := &HTTP{}

	tests := []struct {
		name   string
		config map[string]any
	}{
		{
			name: "minimal GET request",
			config: map[string]any{
				"method": "GET",
				"url":    "https://api.example.com",
			},
		},
		{
			name: "POST with JSON and retry",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/json",
				"json":        map[string]any{"key": "value"},
				"retry": map[string]any{
					"enabled":         true,
					"strategy":        RetryStrategyFixed,
					"maxAttempts":     3,
					"intervalSeconds": 5,
				},
			},
		},
		{
			name: "GET with success code override",
			config: map[string]any{
				"method":       "GET",
				"url":          "https://api.example.com",
				"successCodes": "200,404",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.Setup(core.SetupContext{Configuration: tt.config})
			assert.NoError(t, err)
		})
	}
}

func TestHTTP__Setup__ValidationErrors(t *testing.T) {
	h := &HTTP{}

	tests := []struct {
		name      string
		config    map[string]any
		expectErr string
	}{
		{
			name:      "missing URL",
			config:    map[string]any{"method": "GET"},
			expectErr: "url is required",
		},
		{
			name:      "missing method",
			config:    map[string]any{"url": "https://api.example.com"},
			expectErr: "method is required",
		},
		{
			name: "JSON contentType without json field",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/json",
			},
			expectErr: "json is required",
		},
		{
			name: "invalid retry strategy",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/json",
				"json":        map[string]any{"key": "value"},
				"retry": map[string]any{
					"enabled":         true,
					"strategy":        "linear",
					"maxAttempts":     3,
					"intervalSeconds": 5,
				},
			},
			expectErr: "invalid retry strategy",
		},
		{
			name: "max attempts above limit",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/json",
				"json":        map[string]any{"key": "value"},
				"retry": map[string]any{
					"enabled":         true,
					"strategy":        RetryStrategyFixed,
					"maxAttempts":     RetryMaxAttempts + 1,
					"intervalSeconds": 5,
				},
			},
			expectErr: "max attempts must be less than or equal to 30",
		},
		{
			name: "interval below minimum",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/json",
				"json":        map[string]any{"key": "value"},
				"retry": map[string]any{
					"enabled":         true,
					"strategy":        RetryStrategyFixed,
					"maxAttempts":     3,
					"intervalSeconds": 4,
				},
			},
			expectErr: "interval seconds must be greater than or equal to 5",
		},
		{
			name: "interval above maximum",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/json",
				"json":        map[string]any{"key": "value"},
				"retry": map[string]any{
					"enabled":         true,
					"strategy":        RetryStrategyFixed,
					"maxAttempts":     3,
					"intervalSeconds": int(RetryMaxInterval.Seconds()) + 1,
				},
			},
			expectErr: "interval seconds must be less than or equal to 300",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.Setup(core.SetupContext{Configuration: tt.config})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}
}

func TestHTTP__Execute__GET(t *testing.T) {
	h := &HTTP{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/test", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]string{"hello": "world"})
		require.NoError(t, err)
	}))
	defer server.Close()

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "GET",
		"url":    server.URL + "/test",
	})

	err := h.Execute(ctx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
	assert.Equal(t, "http.request.finished", stateCtx.Type)

	response := responsePayload(t, stateCtx)
	assert.Equal(t, http.StatusOK, response["status"])
	assert.NotNil(t, response["headers"])

	body := response["body"].(map[string]any)
	assert.Equal(t, "world", body["hello"])
}

func TestHTTP__Execute__ReadsResponseBodyBeforeContextCancellation(t *testing.T) {
	h := &HTTP{}
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Logger: log.NewEntry(log.StandardLogger()),
		Configuration: map[string]any{
			"method":         "GET",
			"url":            "https://api.example.com/test",
			"timeoutSeconds": 1,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		HTTP:           &contextBoundHTTPClient{},
	}

	err := h.Execute(ctx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
	assert.Equal(t, "http.request.finished", stateCtx.Type)

	response := responsePayload(t, stateCtx)
	assert.Equal(t, http.StatusOK, response["status"])

	body := response["body"].(map[string]any)
	assert.Equal(t, "world", body["hello"])
}

func TestHTTP__Execute__GET_WithQueryParams(t *testing.T) {
	h := &HTTP{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "bar", r.URL.Query().Get("foo"))
		assert.Equal(t, "2", r.URL.Query().Get("existing"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "GET",
		"url":    server.URL + "/test?existing=1",
		"queryParams": []map[string]any{
			{"key": "foo", "value": "bar"},
			{"key": "existing", "value": "2"},
		},
	})

	err := h.Execute(ctx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
	assert.Equal(t, "http.request.finished", stateCtx.Type)
}

func TestHTTP__Execute__SerializesPayloads(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]any
		assertRequest func(*testing.T, *http.Request)
	}{
		{
			name: "json",
			config: map[string]any{
				"method":      "POST",
				"contentType": "application/json",
				"json":        map[string]any{"foo": "bar"},
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()

				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var requestData map[string]any
				err = json.Unmarshal(body, &requestData)
				require.NoError(t, err)
				assert.Equal(t, "bar", requestData["foo"])
			},
		},
		{
			name: "xml",
			config: map[string]any{
				"method":      "POST",
				"contentType": "application/xml",
				"xml":         "<root><value>ok</value></root>",
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()

				assert.Equal(t, "application/xml", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, "<root><value>ok</value></root>", string(body))
			},
		},
		{
			name: "text",
			config: map[string]any{
				"method":      "POST",
				"contentType": "text/plain",
				"text":        "hello world",
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()

				assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, "hello world", string(body))
			},
		},
		{
			name: "form data",
			config: map[string]any{
				"method":      "POST",
				"contentType": "application/x-www-form-urlencoded",
				"formData": []map[string]any{
					{"key": "username", "value": "john"},
					{"key": "password", "value": "secret123"},
				},
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()

				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

				err := r.ParseForm()
				require.NoError(t, err)
				assert.Equal(t, "john", r.FormValue("username"))
				assert.Equal(t, "secret123", r.FormValue("password"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HTTP{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.assertRequest(t, r)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			}))
			defer server.Close()

			config := tt.config
			config["url"] = server.URL

			ctx, stateCtx, _ := createExecutionContext(config)
			err := h.Execute(ctx)
			require.NoError(t, err)

			assert.True(t, stateCtx.Passed)
			assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
		})
	}
}

func TestHTTP__Execute__HeadersOverrideContentType(t *testing.T) {
	h := &HTTP{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/custom", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method":      "POST",
		"url":         server.URL,
		"contentType": "application/json",
		"json":        map[string]any{"test": "data"},
		"headers": []map[string]any{
			{"name": "Content-Type", "value": "application/custom"},
		},
	})

	err := h.Execute(ctx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
}

func TestHTTP__Execute__NonJSONResponse(t *testing.T) {
	h := &HTTP{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Plain text response"))
	}))
	defer server.Close()

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "GET",
		"url":    server.URL,
	})

	err := h.Execute(ctx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
	assert.Equal(t, "Plain text response", responsePayload(t, stateCtx)["body"])
}

func TestHTTP__Execute__EmptyResponse(t *testing.T) {
	h := &HTTP{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "DELETE",
		"url":    server.URL,
	})

	err := h.Execute(ctx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	response := responsePayload(t, stateCtx)
	assert.Equal(t, http.StatusNoContent, response["status"])
	assert.Nil(t, response["body"])
}

func TestHTTP__Execute__SuccessCodesTreat404AsSuccess(t *testing.T) {
	h := &HTTP{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		err := json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		require.NoError(t, err)
	}))
	defer server.Close()

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method":       "GET",
		"url":          server.URL,
		"successCodes": "404",
	})

	err := h.Execute(ctx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
	assert.Equal(t, "http.request.finished", stateCtx.Type)

	response := responsePayload(t, stateCtx)
	assert.Equal(t, http.StatusNotFound, response["status"])
	assert.Equal(t, "not found", response["body"].(map[string]any)["error"])
}

func TestHTTP__Execute__NetworkErrorWithRetrySchedulesAction(t *testing.T) {
	h := &HTTP{}
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	requestCtx := &contexts.RequestContext{}

	ctx := core.ExecutionContext{
		Logger:         log.NewEntry(log.StandardLogger()),
		Configuration:  retryConfig("https://api.example.com"),
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx,
		HTTP: &sequenceHTTPClient{
			errors: []error{errors.New("dial tcp: connection refused")},
		},
	}

	err := h.Execute(ctx)
	require.NoError(t, err)

	assert.False(t, stateCtx.Finished)
	assert.Equal(t, "retryRequest", requestCtx.Action)
	assert.Equal(t, 5*time.Second, requestCtx.Duration)

	metadata := decodeMetadata(t, metadataCtx)
	assert.Equal(t, 1, metadata.TimeoutSeconds)
	require.NotNil(t, metadata.Retry)
	assert.Equal(t, 1, metadata.Retry.Attempts)
	assert.Equal(t, 3, metadata.Retry.MaxAttempts)
	assert.Equal(t, RetryStrategyFixed, metadata.Retry.Strategy)
	assert.Equal(t, 5, metadata.Retry.Interval)
	assert.Contains(t, metadata.Retry.LastError, "connection refused")
}

func TestHTTP__HandleAction__RetryRequest_SchedulesNextAttempt(t *testing.T) {
	h := &HTTP{}
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	initialRequestCtx := &contexts.RequestContext{}

	httpCtx := &sequenceHTTPClient{
		errors: []error{
			errors.New("temporary outage"),
			errors.New("temporary outage"),
		},
	}

	execCtx := core.ExecutionContext{
		Logger:         log.NewEntry(log.StandardLogger()),
		Configuration:  retryConfig("https://api.example.com"),
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       initialRequestCtx,
		HTTP:           httpCtx,
	}

	err := h.Execute(execCtx)
	require.NoError(t, err)

	actionRequestCtx := &contexts.RequestContext{}
	actionCtx := core.ActionContext{
		Logger:         log.NewEntry(log.StandardLogger()),
		Name:           "retryRequest",
		Configuration:  execCtx.Configuration,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       actionRequestCtx,
		HTTP:           httpCtx,
	}

	err = h.HandleAction(actionCtx)
	require.NoError(t, err)

	assert.False(t, stateCtx.Finished)
	assert.Equal(t, "retryRequest", actionRequestCtx.Action)
	assert.Equal(t, 5*time.Second, actionRequestCtx.Duration)

	metadata := decodeMetadata(t, metadataCtx)
	require.NotNil(t, metadata.Retry)
	assert.Equal(t, 2, metadata.Retry.Attempts)
	assert.Contains(t, metadata.Retry.LastError, "temporary outage")
}

func TestHTTP__HandleAction__RetryRequest_SuccessAfterNetworkError(t *testing.T) {
	h := &HTTP{}
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	requestCtx := &contexts.RequestContext{}

	httpCtx := &sequenceHTTPClient{
		errors: []error{
			errors.New("temporary outage"),
			nil,
		},
		responses: []*http.Response{
			newResponse(http.StatusOK, `{"result":"success on retry"}`, http.Header{
				"Content-Type": []string{"application/json"},
			}),
		},
	}

	execCtx := core.ExecutionContext{
		Logger:         log.NewEntry(log.StandardLogger()),
		Configuration:  retryConfig("https://api.example.com"),
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx,
		HTTP:           httpCtx,
	}

	err := h.Execute(execCtx)
	require.NoError(t, err)
	assert.Equal(t, "retryRequest", requestCtx.Action)

	actionCtx := core.ActionContext{
		Logger:         log.NewEntry(log.StandardLogger()),
		Name:           "retryRequest",
		Configuration:  execCtx.Configuration,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       &contexts.RequestContext{},
		HTTP:           httpCtx,
	}

	err = h.HandleAction(actionCtx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
	assert.Equal(t, "http.request.finished", stateCtx.Type)

	response := responsePayload(t, stateCtx)
	assert.Equal(t, http.StatusOK, response["status"])
	assert.Equal(t, "success on retry", response["body"].(map[string]any)["result"])
}

func TestHTTP__CalculateNextRetryDelay(t *testing.T) {
	h := &HTTP{}

	assert.Equal(t, 5*time.Second, h.calculateNextRetryDelay(RetryStrategyFixed, 2, 5))
	assert.Equal(t, 20*time.Second, h.calculateNextRetryDelay(RetryStrategyExponential, 2, 5))
}
