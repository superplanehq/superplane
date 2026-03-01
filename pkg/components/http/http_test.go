package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func createExecutionContext(config map[string]any) (core.ExecutionContext, *contexts.ExecutionStateContext, *contexts.MetadataContext) {
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	return core.ExecutionContext{
		Configuration:  config,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		HTTP:           &http.Client{},
	}, stateCtx, metadataCtx
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
			name: "POST with JSON",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/json",
				"json":        map[string]any{"key": "value"},
			},
		},
		{
			name: "POST with XML",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/xml",
				"xml":         "<root><element>value</element></root>",
			},
		},
		{
			name: "POST with plain text",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "text/plain",
				"text":        "plain text content",
			},
		},
		{
			name: "POST with form data",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/x-www-form-urlencoded",
				"formData":    []map[string]any{{"key": "username", "value": "john"}},
			},
		},
		{
			name: "with headers",
			config: map[string]any{
				"method":  "POST",
				"url":     "https://api.example.com",
				"headers": []map[string]any{{"name": "Authorization", "value": "Bearer token"}},
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
			name: "contentType with missing payload",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/json",
			},
			expectErr: "json is required",
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
			name: "XML contentType without xml field",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/xml",
			},
			expectErr: "xml is required",
		},
		{
			name: "text contentType without text field",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "text/plain",
			},
			expectErr: "text is required",
		},
		{
			name: "form data contentType without formData field",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"contentType": "application/x-www-form-urlencoded",
			},
			expectErr: "form data is required",
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

	//
	// Create test server
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/test", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"hello": "world"})
	}))
	defer server.Close()

	//
	// Execute component
	//
	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "GET",
		"url":    server.URL + "/test",
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)

	//
	// Verify output
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")

	payload := stateCtx.Payloads[0].(map[string]any)
	response := payload["data"].(map[string]any)
	assert.Equal(t, 200, response["status"])
	assert.NotNil(t, response["headers"])
	assert.NotNil(t, response["body"])

	//
	// Verify body was parsed as JSON
	//
	body := response["body"].(map[string]any)
	assert.Equal(t, "world", body["hello"])
}

func TestHTTP__Execute__ReadsResponseBodyBeforeContextCancellation(t *testing.T) {
	h := &HTTP{}
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method": "GET",
			"url":    "https://api.example.com/test",
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		HTTP:           &contextBoundHTTPClient{},
	}

	err := h.Execute(ctx)
	require.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.Equal(t, "http.request.finished", stateCtx.Type)

	payload := stateCtx.Payloads[0].(map[string]any)
	response := payload["data"].(map[string]any)
	assert.Equal(t, 200, response["status"])

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
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, "http.request.finished", stateCtx.Type)
}

func TestHTTP__Execute__POST_JSON(t *testing.T) {
	//
	// Create test server
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var requestData map[string]any
		err = json.Unmarshal(body, &requestData)
		require.NoError(t, err)
		assert.Equal(t, "bar", requestData["foo"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"result": "created"})
	}))

	defer server.Close()

	//
	// Execute component
	//
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method":      "POST",
		"url":         server.URL,
		"contentType": "application/json",
		"json":        map[string]any{"foo": "bar"},
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	//
	// Verify response
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")

	payload := stateCtx.Payloads[0].(map[string]any)
	response := payload["data"].(map[string]any)
	assert.Equal(t, 201, response["status"])
}

func TestHTTP__Execute__POST_XML(t *testing.T) {
	xmlPayload := `<?xml version="1.0"?><root><element>value</element></root>`

	//
	// Create test server
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/xml", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, xmlPayload, string(body))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<response>OK</response>"))
	}))

	defer server.Close()

	//
	// Execute component
	//
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method":      "POST",
		"url":         server.URL,
		"contentType": "application/xml",
		"xml":         xmlPayload,
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	//
	// Verify response.
	// Body should be a string since it's XML.
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")
	payload := stateCtx.Payloads[0].(map[string]any)
	response := payload["data"].(map[string]any)
	assert.Equal(t, 200, response["status"])
	body := response["body"].(string)
	assert.Contains(t, body, "OK")
}

func TestHTTP__Execute__POST_PlainText(t *testing.T) {
	textPayload := "Hello, World!"

	//
	// Create test server
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, textPayload, string(body))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Received"))
	}))

	defer server.Close()

	//
	// Execute component
	//
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method":      "POST",
		"url":         server.URL,
		"contentType": "text/plain",
		"text":        textPayload,
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
}

func TestHTTP__Execute__POST_FormData(t *testing.T) {
	//
	// Create test server
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "john", r.FormValue("username"))
		assert.Equal(t, "secret123", r.FormValue("password"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
	}))
	defer server.Close()

	//
	// Execute component
	//
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method":      "POST",
		"url":         server.URL,
		"contentType": "application/x-www-form-urlencoded",
		"formData": []map[string]any{
			{"key": "username", "value": "john"},
			{"key": "password", "value": "secret123"},
		},
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
}

func TestHTTP__Execute__WithCustomHeaders(t *testing.T) {
	//
	// Create test server.
	// Here, we verify that the headers are sent.
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "CustomAgent/1.0", r.Header.Get("User-Agent"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"authenticated": "true"})
	}))
	defer server.Close()

	//
	// Execute component
	//
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "GET",
		"url":    server.URL,
		"headers": []map[string]any{
			{"name": "Authorization", "value": "Bearer token123"},
			{"name": "Accept", "value": "application/json"},
			{"name": "User-Agent", "value": "CustomAgent/1.0"},
		},
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
}

func TestHTTP__Execute__HeadersOverrideContentType(t *testing.T) {
	//
	// Create test server
	// Custom header should override the content type
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/custom", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	//
	// Execute component
	//
	h := &HTTP{}

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
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
}

func TestHTTP__Execute__NonJSONResponse(t *testing.T) {
	//
	// Create test server that returns plain text
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Plain text response"))
	}))
	defer server.Close()

	//
	// Execute component
	//
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "GET",
		"url":    server.URL,
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	//
	// Verify body is stored as string
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")
	payload := stateCtx.Payloads[0].(map[string]any)
	response := payload["data"].(map[string]any)
	body := response["body"].(string)
	assert.Equal(t, "Plain text response", body)
}

func TestHTTP__Execute__EmptyResponse(t *testing.T) {
	//
	// Create test server with empty response
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	//
	// Execute component
	//
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "DELETE",
		"url":    server.URL,
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	//
	// Verify response structure
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")
	payload := stateCtx.Payloads[0].(map[string]any)
	response := payload["data"].(map[string]any)
	assert.Equal(t, 204, response["status"])
	assert.Nil(t, response["body"])
}

func TestHTTP__Execute__HTTPError(t *testing.T) {
	//
	// Create test server that returns error
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	defer server.Close()

	//
	// Execute component
	// Component should fail on HTTP errors (non-2xx by default)
	//
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "GET",
		"url":    server.URL,
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.False(t, stateCtx.Passed)

	// The failed request should set failure state but no event is emitted
	// because the function returns early after calling Fail()
}

func TestHTTP__Execute__InvalidURL(t *testing.T) {
	h := &HTTP{}

	ctx, stateCtx, _ := createExecutionContext(map[string]any{
		"method": "GET",
		"url":    "://invalid-url",
	})

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.False(t, stateCtx.Passed)
}

func TestHTTP__Setup__TimeoutAndRetryConfiguration(t *testing.T) {
	h := &HTTP{}

	tests := []struct {
		name   string
		config map[string]any
	}{
		{
			name: "timeout strategy with valid configuration",
			config: map[string]any{
				"method":          "GET",
				"url":             "https://api.example.com",
				"timeoutStrategy": "fixed",
				"timeoutSeconds":  30,
				"retries":         3,
			},
		},
		{
			name: "exponential timeout strategy",
			config: map[string]any{
				"method":          "GET",
				"url":             "https://api.example.com",
				"timeoutStrategy": "exponential",
				"timeoutSeconds":  5,
				"retries":         2,
			},
		},
		{
			name: "without timeout strategy (default behavior)",
			config: map[string]any{
				"method": "GET",
				"url":    "https://api.example.com",
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

func TestHTTP__Execute__WithoutRetryStrategy(t *testing.T) {
	h := &HTTP{}

	//
	// Create test server that succeeds immediately
	//
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"result": "success"})
	}))
	defer server.Close()

	//
	// Execute component without timeout strategy
	//
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method": "GET",
			"url":    server.URL,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		HTTP:           &http.Client{},
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	// Verify metadata contains retry information even without strategy
	metadata := metadataCtx.Get()
	assert.NotNil(t, metadata)

	var retryMeta RetryMetadata
	err = mapstructure.Decode(metadata, &retryMeta)
	require.NoError(t, err)
	assert.Equal(t, "success", retryMeta.Result)
	assert.Equal(t, 200, retryMeta.FinalStatus)
	assert.Equal(t, 0, retryMeta.TotalRetries)
}

func TestHTTP__Execute__FixedTimeoutStrategy_Success(t *testing.T) {
	h := &HTTP{}

	//
	// Create test server that succeeds on first attempt
	//
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"result": "success"})
	}))
	defer server.Close()

	//
	// Execute component with fixed timeout strategy
	//
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":          "GET",
			"url":             server.URL,
			"timeoutStrategy": "fixed",
			"timeoutSeconds":  5,
			"retries":         2,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		HTTP:           &http.Client{},
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	// Should only make one request since it succeeds
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))

	// Verify response
	assert.Equal(t, stateCtx.Type, "http.request.finished")
	payload := stateCtx.Payloads[0].(map[string]any)
	response := payload["data"].(map[string]any)
	assert.Equal(t, 200, response["status"])
}

func TestHTTP__Execute__ExponentialTimeoutStrategy_Success(t *testing.T) {
	h := &HTTP{}

	//
	// Create test server that succeeds on first attempt
	//
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"result": "success"})
	}))
	defer server.Close()

	//
	// Execute component with exponential timeout strategy
	//
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":          "GET",
			"url":             server.URL,
			"timeoutStrategy": "exponential",
			"timeoutSeconds":  2,
			"retries":         3,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		HTTP:           &http.Client{},
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	// Should only make one request since it succeeds
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

func TestHTTP__HandleAction__RetryRequest_SuccessOnRetry(t *testing.T) {
	h := &HTTP{}

	//
	// Create test server that fails first time, succeeds on retry
	//
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 1 {
			// Fail first request
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			// Succeed on retry
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"result": "success on retry"})
		}
	}))
	defer server.Close()

	//
	// Execute initial request (should fail and schedule retry)
	//
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	requestCtx := &contexts.RequestContext{}

	httpCtx := &http.Client{}
	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":          "GET",
			"url":             server.URL,
			"timeoutStrategy": "fixed",
			"timeoutSeconds":  1,
			"retries":         1,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx,
		HTTP:           httpCtx,
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.False(t, stateCtx.Finished)

	assert.Equal(t, "retryRequest", requestCtx.Action)
	assert.Equal(t, 1*time.Second, requestCtx.Duration)

	actionCtx := core.ActionContext{
		Name:           "retryRequest",
		Configuration:  ctx.Configuration,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx,
		HTTP:           httpCtx,
	}

	err = h.HandleAction(actionCtx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	// Should have made 2 requests total
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))

	// Verify final response
	assert.Equal(t, stateCtx.Type, "http.request.finished")

	// Verify metadata shows successful completion after retry
	metadata := metadataCtx.Get()
	var retryMeta RetryMetadata
	err = mapstructure.Decode(metadata, &retryMeta)
	require.NoError(t, err)
	assert.Equal(t, "success", retryMeta.Result)
	assert.Equal(t, 200, retryMeta.FinalStatus)
	assert.Equal(t, 1, retryMeta.TotalRetries)
}

func TestHTTP__HandleAction__RetryRequest_ExhaustedRetries(t *testing.T) {
	h := &HTTP{}

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "server error"})
	}))
	defer server.Close()

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	requestCtx := &contexts.RequestContext{}
	httpCtx := &http.Client{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":          "GET",
			"url":             server.URL,
			"timeoutStrategy": "fixed",
			"timeoutSeconds":  1,
			"retries":         2,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx,
		HTTP:           httpCtx,
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)

	actionCtx := core.ActionContext{
		Name:           "retryRequest",
		Configuration:  ctx.Configuration,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       &contexts.RequestContext{},
		HTTP:           httpCtx,
	}

	err = h.HandleAction(actionCtx)
	assert.NoError(t, err)

	actionCtx.Requests = &contexts.RequestContext{}
	err = h.HandleAction(actionCtx)
	assert.NoError(t, err)
	assert.False(t, stateCtx.Passed)

	// Should have made 3 requests total (initial + 2 retries)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))

	// Verify metadata shows failure after exhausted retries
	metadata := metadataCtx.Get()
	var retryMeta RetryMetadata
	err = mapstructure.Decode(metadata, &retryMeta)
	require.NoError(t, err)
	assert.Equal(t, "failed", retryMeta.Result)
	assert.Equal(t, 500, retryMeta.FinalStatus)
	assert.Equal(t, 2, retryMeta.TotalRetries)
}

func TestHTTP__CalculateTimeoutForAttempt__Fixed(t *testing.T) {
	h := &HTTP{}

	// Fixed strategy should always return the base timeout
	timeout1 := h.calculateTimeoutForAttempt("fixed", 5, 0)
	timeout2 := h.calculateTimeoutForAttempt("fixed", 5, 1)
	timeout3 := h.calculateTimeoutForAttempt("fixed", 5, 2)

	assert.Equal(t, 5*time.Second, timeout1)
	assert.Equal(t, 5*time.Second, timeout2)
	assert.Equal(t, 5*time.Second, timeout3)
}

func TestHTTP__CalculateTimeoutForAttempt__Exponential(t *testing.T) {
	h := &HTTP{}

	// Exponential strategy: timeout * 2^attempt
	timeout0 := h.calculateTimeoutForAttempt("exponential", 5, 0)
	timeout1 := h.calculateTimeoutForAttempt("exponential", 5, 1)
	timeout2 := h.calculateTimeoutForAttempt("exponential", 5, 2)

	assert.Equal(t, 5*time.Second, timeout0)
	assert.Equal(t, 10*time.Second, timeout1)
	assert.Equal(t, 20*time.Second, timeout2)

	// Test capping at 120 seconds
	timeout5 := h.calculateTimeoutForAttempt("exponential", 30, 5)
	assert.Equal(t, 120*time.Second, timeout5)
}

func TestHTTP__Actions__ReturnsRetryAction(t *testing.T) {
	h := &HTTP{}
	actions := h.Actions()

	assert.Len(t, actions, 1)
	assert.Equal(t, "retryRequest", actions[0].Name)
}

func TestHTTP__HandleAction__UnknownAction(t *testing.T) {
	h := &HTTP{}

	ctx := core.ActionContext{
		Name: "unknownAction",
	}

	err := h.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action: unknownAction")
}

func TestHTTP__RetryProgression_ExponentialStrategy(t *testing.T) {
	h := &HTTP{}

	//
	// Create test server that fails exactly 3 times, then succeeds
	//
	var requestCount int32
	var requestTimes []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		requestTimes = append(requestTimes, time.Now())

		if count <= 3 {
			// Fail first 3 requests
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			// Succeed on 4th request
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"result": "success after retries"})
		}
	}))
	defer server.Close()

	//
	// Execute with exponential strategy (3 retries)
	//
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	requestCtx := &contexts.RequestContext{}
	httpCtx := &http.Client{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":          "GET",
			"url":             server.URL,
			"timeoutStrategy": "exponential",
			"timeoutSeconds":  2,
			"retries":         3,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx,
		HTTP:           httpCtx,
	}

	// Execute initial request (should fail and schedule retry)
	err := h.Execute(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "retryRequest", requestCtx.Action)

	requestCtx1 := &contexts.RequestContext{}
	actionCtx := core.ActionContext{
		Name:           "retryRequest",
		Configuration:  ctx.Configuration,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx1,
		HTTP:           httpCtx,
	}

	err = h.HandleAction(actionCtx)
	assert.NoError(t, err)
	assert.Equal(t, "retryRequest", requestCtx1.Action)

	requestCtx2 := &contexts.RequestContext{}
	actionCtx.Requests = requestCtx2
	err = h.HandleAction(actionCtx)
	assert.NoError(t, err)

	assert.Equal(t, "retryRequest", requestCtx2.Action)

	requestCtx3 := &contexts.RequestContext{}
	actionCtx.Requests = requestCtx3
	err = h.HandleAction(actionCtx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	// Verify all attempts were made
	assert.Equal(t, int32(4), atomic.LoadInt32(&requestCount))

	// Verify final response
	assert.Equal(t, "http.request.finished", stateCtx.Type)
	payload := stateCtx.Payloads[0].(map[string]any)
	response := payload["data"].(map[string]any)
	assert.Equal(t, 200, response["status"])

	// Verify metadata shows success after 3 retries
	metadata := metadataCtx.Get()
	var retryMeta RetryMetadata
	err = mapstructure.Decode(metadata, &retryMeta)
	require.NoError(t, err)
	assert.Equal(t, "success", retryMeta.Result)
	assert.Equal(t, 200, retryMeta.FinalStatus)
	assert.Equal(t, 3, retryMeta.TotalRetries)
}

func TestHTTP__RetryProgression_NetworkError(t *testing.T) {
	h := &HTTP{}

	//
	// Test retry on network errors (not just HTTP status errors)
	//
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	requestCtx := &contexts.RequestContext{}
	httpCtx := &http.Client{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":          "GET",
			"url":             "http://invalid-host-that-does-not-exist.com",
			"timeoutStrategy": "fixed",
			"timeoutSeconds":  1,
			"retries":         2,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       requestCtx,
		HTTP:           httpCtx,
	}

	// Execute initial request (should fail with network error and schedule retry)
	err := h.Execute(ctx)
	assert.NoError(t, err)

	// Verify retry is scheduled
	assert.Equal(t, "retryRequest", requestCtx.Action)
	assert.Equal(t, 1*time.Second, requestCtx.Duration)

	// Simulate retries until exhaustion
	for i := 0; i < 2; i++ {
		actionCtx := core.ActionContext{
			Name:           "retryRequest",
			Configuration:  ctx.Configuration,
			ExecutionState: stateCtx,
			Metadata:       metadataCtx,
			Requests:       &contexts.RequestContext{},
			HTTP:           httpCtx,
		}

		if i == 1 {
			// Last retry should fail but return nil
			err = h.HandleAction(actionCtx)
			assert.NoError(t, err)
			assert.False(t, stateCtx.Passed)
		} else {
			// Earlier retries should schedule next retry
			err = h.HandleAction(actionCtx)
			assert.NoError(t, err)
		}
	}
}

func TestHTTP__RetryMetadata_Progression(t *testing.T) {
	h := &HTTP{}

	//
	// Test that retry metadata correctly tracks attempts
	//
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	httpCtx := &http.Client{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":          "GET",
			"url":             server.URL,
			"timeoutStrategy": "fixed",
			"timeoutSeconds":  1,
			"retries":         2,
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       &contexts.RequestContext{},
		HTTP:           httpCtx,
	}

	// Execute initial request (attempt 0)
	err := h.Execute(ctx)
	assert.NoError(t, err)

	// Check initial metadata
	metadata := metadataCtx.Get()
	var retryMeta RetryMetadata
	err = mapstructure.Decode(metadata, &retryMeta)
	require.NoError(t, err)
	assert.Equal(t, 1, retryMeta.Attempt)
	assert.Equal(t, 1, retryMeta.TotalRetries)
	assert.Equal(t, 2, retryMeta.MaxRetries)
	assert.Equal(t, "fixed", retryMeta.TimeoutStrategy)
	assert.Equal(t, "HTTP status 500", retryMeta.LastError)

	// Simulate first retry (attempt 1)
	actionCtx := core.ActionContext{
		Name:           "retryRequest",
		Configuration:  ctx.Configuration,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Requests:       &contexts.RequestContext{},
		HTTP:           httpCtx,
	}

	err = h.HandleAction(actionCtx)
	assert.NoError(t, err)

	metadata = metadataCtx.Get()
	err = mapstructure.Decode(metadata, &retryMeta)
	require.NoError(t, err)
	assert.Equal(t, 2, retryMeta.Attempt)
	assert.Equal(t, 2, retryMeta.TotalRetries)
	assert.Equal(t, "HTTP status 500", retryMeta.LastError)

	actionCtx.Requests = &contexts.RequestContext{}
	err = h.HandleAction(actionCtx)
	assert.NoError(t, err)
	assert.False(t, stateCtx.Passed)

	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}
