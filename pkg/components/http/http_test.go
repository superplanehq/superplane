package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

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
				"sendBody":    true,
				"contentType": "application/json",
				"json":        map[string]any{"key": "value"},
			},
		},
		{
			name: "POST with XML",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"sendBody":    true,
				"contentType": "application/xml",
				"xml":         "<root><element>value</element></root>",
			},
		},
		{
			name: "POST with plain text",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"sendBody":    true,
				"contentType": "text/plain",
				"text":        "plain text content",
			},
		},
		{
			name: "POST with form data",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"sendBody":    true,
				"contentType": "application/x-www-form-urlencoded",
				"formData":    []map[string]any{{"key": "username", "value": "john"}},
			},
		},
		{
			name: "with headers",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"sendHeaders": true,
				"headers":     []map[string]any{{"name": "Authorization", "value": "Bearer token"}},
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
			name: "sendBody without contentType",
			config: map[string]any{
				"method":   "POST",
				"url":      "https://api.example.com",
				"sendBody": true,
			},
			expectErr: "content type is required when sending a body",
		},
		{
			name: "JSON contentType without json field",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"sendBody":    true,
				"contentType": "application/json",
			},
			expectErr: "json is required",
		},
		{
			name: "XML contentType without xml field",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"sendBody":    true,
				"contentType": "application/xml",
			},
			expectErr: "xml is required",
		},
		{
			name: "text contentType without text field",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"sendBody":    true,
				"contentType": "text/plain",
			},
			expectErr: "text is required",
		},
		{
			name: "form data contentType without formData field",
			config: map[string]any{
				"method":      "POST",
				"url":         "https://api.example.com",
				"sendBody":    true,
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
	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method": "GET",
			"url":    server.URL + "/test",
		},
		ExecutionStateContext: stateCtx,
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)

	//
	// Verify output
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")

	response := stateCtx.Payloads[0].(map[string]any)
	assert.Equal(t, 200, response["status"])
	assert.NotNil(t, response["headers"])
	assert.NotNil(t, response["body"])

	//
	// Verify body was parsed as JSON
	//
	body := response["body"].(map[string]any)
	assert.Equal(t, "world", body["hello"])
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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":      "POST",
			"url":         server.URL,
			"sendBody":    true,
			"contentType": "application/json",
			"json":        map[string]any{"foo": "bar"},
		},
		ExecutionStateContext: stateCtx,
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	//
	// Verify response
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")

	response := stateCtx.Payloads[0].(map[string]any)
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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":      "POST",
			"url":         server.URL,
			"sendBody":    true,
			"contentType": "application/xml",
			"xml":         xmlPayload,
		},
		ExecutionStateContext: stateCtx,
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	//
	// Verify response.
	// Body should be a string since it's XML.
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")
	response := stateCtx.Payloads[0].(map[string]any)
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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":      "POST",
			"url":         server.URL,
			"sendBody":    true,
			"contentType": "text/plain",
			"text":        textPayload,
		},
		ExecutionStateContext: stateCtx,
	}

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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":      "POST",
			"url":         server.URL,
			"sendBody":    true,
			"contentType": "application/x-www-form-urlencoded",
			"formData": []map[string]any{
				{"key": "username", "value": "john"},
				{"key": "password", "value": "secret123"},
			},
		},
		ExecutionStateContext: stateCtx,
	}

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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":      "GET",
			"url":         server.URL,
			"sendHeaders": true,
			"headers": []map[string]any{
				{"name": "Authorization", "value": "Bearer token123"},
				{"name": "Accept", "value": "application/json"},
				{"name": "User-Agent", "value": "CustomAgent/1.0"},
			},
		},
		ExecutionStateContext: stateCtx,
	}

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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method":      "POST",
			"url":         server.URL,
			"sendBody":    true,
			"contentType": "application/json",
			"json":        map[string]any{"test": "data"},
			"sendHeaders": true,
			"headers": []map[string]any{
				{"name": "Content-Type", "value": "application/custom"},
			},
		},
		ExecutionStateContext: stateCtx,
	}

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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method": "GET",
			"url":    server.URL,
		},
		ExecutionStateContext: stateCtx,
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	//
	// Verify body is stored as string
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")
	response := stateCtx.Payloads[0].(map[string]any)
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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method": "DELETE",
			"url":    server.URL,
		},
		ExecutionStateContext: stateCtx,
	}

	err := h.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)

	//
	// Verify response structure
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.finished")
	response := stateCtx.Payloads[0].(map[string]any)
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
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method": "GET",
			"url":    server.URL,
		},
		ExecutionStateContext: stateCtx,
	}

	err := h.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed with status 404")

	//
	// Verify status code is captured and failed event is emitted
	//
	assert.Equal(t, stateCtx.Channel, core.DefaultOutputChannel.Name)
	assert.Equal(t, stateCtx.Type, "http.request.failed")
	response := stateCtx.Payloads[0].(map[string]any)
	assert.Equal(t, 404, response["status"])
}

func TestHTTP__Execute__InvalidURL(t *testing.T) {
	h := &HTTP{}
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"method": "GET",
			"url":    "://invalid-url",
		},
		ExecutionStateContext: stateCtx,
	}

	err := h.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}
