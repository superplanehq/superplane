package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestTextPrompt_Configuration(t *testing.T) {
	c := &TextPrompt{}
	config := c.Configuration()

	expectedFields := map[string]struct {
		Required bool
		Type     string
	}{
		"model":         {true, string(configuration.FieldTypeIntegrationResource)},
		"prompt":        {true, string(configuration.FieldTypeText)},
		"systemMessage": {false, string(configuration.FieldTypeText)},
		"maxTokens":     {false, string(configuration.FieldTypeNumber)},
		"temperature":   {false, string(configuration.FieldTypeNumber)},
		"files":         {false, string(configuration.FieldTypeList)},
		"codeExecution": {false, string(configuration.FieldTypeBool)},
		"outputSchema":  {false, string(configuration.FieldTypeText)},
	}

	for _, field := range config {
		expected, ok := expectedFields[field.Name]
		if !ok {
			t.Errorf("unexpected field: %s", field.Name)
			continue
		}
		if field.Required != expected.Required {
			t.Errorf("field %s: expected required %v, got %v", field.Name, expected.Required, field.Required)
		}
		if string(field.Type) != expected.Type {
			t.Errorf("field %s: expected type %s, got %s", field.Name, expected.Type, field.Type)
		}
	}
}

func TestTextPrompt_Setup(t *testing.T) {
	c := &TextPrompt{}

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
	}{
		{
			name: "Valid Config",
			config: map[string]interface{}{
				"model":  "claude-3-opus",
				"prompt": "Hello",
			},
			expectError: false,
		},
		{
			name: "Missing Model",
			config: map[string]interface{}{
				"prompt": "Hello",
			},
			expectError: true,
		},
		{
			name: "Missing Prompt",
			config: map[string]interface{}{
				"model": "claude-3-opus",
			},
			expectError: true,
		},
		{
			name: "Valid output schema",
			config: map[string]interface{}{
				"model":        "claude-3-opus",
				"prompt":       "Hello",
				"outputSchema": `{"type":"object","properties":{"sentiment":{"type":"string"}},"required":["sentiment"]}`,
			},
			expectError: false,
		},
		{
			name: "Invalid JSON schema",
			config: map[string]interface{}{
				"model":        "claude-3-opus",
				"prompt":       "Hello",
				"outputSchema": `{"type":"object",}`,
			},
			expectError: true,
		},
		{
			name: "Schema root not an object",
			config: map[string]interface{}{
				"model":        "claude-3-opus",
				"prompt":       "Hello",
				"outputSchema": `["a","b"]`,
			},
			expectError: true,
		},
		{
			name: "Schema missing properties",
			config: map[string]interface{}{
				"model":        "claude-3-opus",
				"prompt":       "Hello",
				"outputSchema": `{"type":"object"}`,
			},
			expectError: true,
		},
		{
			name: "Schema with unresolved expression is allowed",
			config: map[string]interface{}{
				"model":        "claude-3-opus",
				"prompt":       "Hello",
				"outputSchema": `{"type":"object","properties":{"x":{"type":"string","description":"{{ inputs.desc }}"}}}`,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := core.SetupContext{
				Configuration: tt.config,
			}
			err := c.Setup(ctx)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTextPrompt_Execute(t *testing.T) {
	c := &TextPrompt{}

	// Helper to create a valid response JSON
	validResponseJSON := `{
		"id": "msg_01",
		"type": "message",
		"role": "assistant",
		"model": "claude-3-test",
		"content": [
			{"type": "text", "text": "Hello world"}
		],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`

	tests := []struct {
		name            string
		config          map[string]interface{}
		responseStatus  int
		responseBody    string
		expectError     bool
		expectEmission  bool
		validatePayload func(*testing.T, MessagePayload)
	}{
		{
			name: "Success",
			config: map[string]interface{}{
				"model":         "claude-3-test",
				"prompt":        "Say hello",
				"maxTokens":     500,
				"systemMessage": "You are a bot",
				"temperature":   0.7,
			},
			responseStatus: 200,
			responseBody:   validResponseJSON,
			expectError:    false,
			expectEmission: true,
			validatePayload: func(t *testing.T, p MessagePayload) {
				if p.Text != "Hello world" {
					t.Errorf("expected text 'Hello world', got '%s'", p.Text)
				}
				if p.ID != "msg_01" {
					t.Errorf("expected ID 'msg_01', got '%s'", p.ID)
				}
				if p.Usage.InputTokens != 10 {
					t.Errorf("expected usage 10, got %d", p.Usage.InputTokens)
				}
			},
		},
		{
			name: "Missing Configuration in Execute",
			config: map[string]interface{}{
				"model": "", // Invalid
			},
			expectError: true,
		},
		{
			name: "API Error",
			config: map[string]interface{}{
				"model":  "claude-3-test",
				"prompt": "fail me",
			},
			responseStatus: 500,
			responseBody:   `{"error": {"message": "internal error"}}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
			integrationCtx := &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			}

			var responses []*http.Response
			if tt.responseStatus != 0 {
				responses = []*http.Response{
					{
						StatusCode: tt.responseStatus,
						Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
					},
				}
			}
			httpCtx := &contexts.HTTPContext{Responses: responses}

			ctx := core.ExecutionContext{
				Configuration:  tt.config,
				ExecutionState: execState,
				HTTP:           httpCtx,
				Integration:    integrationCtx,
			}

			err := c.Execute(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.expectEmission {
				if execState.Type != MessagePayloadType {
					t.Errorf("expected emitted type %s, got %s", MessagePayloadType, execState.Type)
				}
				if len(execState.Payloads) != 1 {
					t.Errorf("expected 1 payload, got %d", len(execState.Payloads))
				} else if tt.validatePayload != nil {
					wrapped, ok := execState.Payloads[0].(map[string]any)
					if !ok {
						t.Error("emitted payload wrapper is not map[string]any")
						return
					}
					data, ok := wrapped["data"].(MessagePayload)
					if !ok {
						t.Error("emitted payload data is not MessagePayload")
						return
					}
					tt.validatePayload(t, data)
				}
				// Verify request body was sent correctly (e.g. Success case)
				if len(httpCtx.Requests) == 1 && tt.validatePayload != nil {
					bodyBytes, _ := io.ReadAll(httpCtx.Requests[0].Body)
					var sent CreateMessageRequest
					if err := json.Unmarshal(bodyBytes, &sent); err != nil {
						t.Errorf("failed to unmarshal sent body: %v", err)
					} else if sent.Model != "claude-3-test" || sent.MaxTokens != 500 || sent.System != "You are a bot" {
						t.Errorf("request body mismatch: model=%s max_tokens=%d system=%s", sent.Model, sent.MaxTokens, sent.System)
					}
				}
			}
		})
	}
}

func TestTextPrompt_StructuredOutput(t *testing.T) {
	c := &TextPrompt{}

	outputSchema := `{"type":"object","properties":{"sentiment":{"type":"string"}},"required":["sentiment"]}`

	run := func(t *testing.T, responseBody string) (MessagePayload, []byte) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(responseBody))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"model":        "claude-3-test",
				"prompt":       "classify",
				"outputSchema": outputSchema,
			},
			ExecutionState: execState,
			HTTP:           httpCtx,
			Integration:    integrationCtx,
		}
		if err := c.Execute(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wrapped, ok := execState.Payloads[0].(map[string]any)
		if !ok {
			t.Fatal("emitted payload wrapper is not map[string]any")
		}
		payload, ok := wrapped["data"].(MessagePayload)
		if !ok {
			t.Fatal("emitted payload data is not MessagePayload")
		}
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		return payload, body
	}

	t.Run("parses JSON output and sends output_config", func(t *testing.T) {
		payload, body := run(t, `{
			"id": "msg_1",
			"model": "claude-3-test",
			"content": [{"type": "text", "text": "{\"sentiment\":\"positive\"}"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 1, "output_tokens": 1}
		}`)

		var sent CreateMessageRequest
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("failed to unmarshal sent body: %v", err)
		}
		if sent.OutputConfig == nil || sent.OutputConfig.Format == nil || sent.OutputConfig.Format.Type != "json_schema" {
			t.Fatalf("expected output_config.format.type=json_schema in request, got %+v", sent.OutputConfig)
		}
		parsed, ok := payload.Parsed.(map[string]any)
		if !ok {
			t.Fatalf("expected Parsed to be an object, got %T", payload.Parsed)
		}
		if parsed["sentiment"] != "positive" {
			t.Errorf("expected sentiment=positive, got %v", parsed["sentiment"])
		}
		if payload.Text != `{"sentiment":"positive"}` {
			t.Errorf("expected raw text preserved, got %q", payload.Text)
		}
	})

	t.Run("refusal leaves Parsed nil but keeps text", func(t *testing.T) {
		payload, _ := run(t, `{
			"id": "msg_2",
			"model": "claude-3-test",
			"content": [{"type": "text", "text": "I can't help with that."}],
			"stop_reason": "refusal",
			"usage": {"input_tokens": 1, "output_tokens": 1}
		}`)
		if payload.Parsed != nil {
			t.Errorf("expected Parsed nil on refusal, got %v", payload.Parsed)
		}
		if payload.Text != "I can't help with that." {
			t.Errorf("expected refusal text preserved, got %q", payload.Text)
		}
	})
}

func TestExtractMessageText(t *testing.T) {
	tests := []struct {
		name     string
		response *CreateMessageResponse
		expected string
	}{
		{
			name:     "Nil Response",
			response: nil,
			expected: "",
		},
		{
			name: "Single Text Block",
			response: &CreateMessageResponse{
				Content: []MessageContent{
					{Type: "text", Text: "Hello"},
				},
			},
			expected: "Hello",
		},
		{
			name: "Multiple Text Blocks",
			response: &CreateMessageResponse{
				Content: []MessageContent{
					{Type: "text", Text: "Hello"},
					{Type: "text", Text: "World"},
				},
			},
			expected: "Hello\nWorld",
		},
		{
			name: "Mixed Blocks (ignore non-text if any)",
			response: &CreateMessageResponse{
				Content: []MessageContent{
					{Type: "image", Text: ""}, // hypothetical non-text
					{Type: "text", Text: "Real Text"},
				},
			},
			expected: "Real Text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMessageText(tt.response)
			if got != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

type fakeFiles struct{ data map[string][]byte }

func (f *fakeFiles) List() ([]string, error) {
	out := make([]string, 0, len(f.data))
	for k := range f.data {
		out = append(out, k)
	}
	return out, nil
}

func (f *fakeFiles) Read(path string) (io.ReadCloser, error) {
	b, ok := f.data[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// uploadPartContentType parses the multipart upload body and returns the
// Content-Type header of the "file" part.
func uploadPartContentType(t *testing.T, req *http.Request) string {
	t.Helper()
	_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse upload Content-Type: %v", err)
	}
	mr := multipart.NewReader(req.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read multipart part: %v", err)
		}
		if part.FormName() == "file" {
			return part.Header.Get("Content-Type")
		}
	}
	return ""
}

func TestTextPrompt_Attachments(t *testing.T) {
	c := &TextPrompt{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"id":"file_abc"}`))},                                                                                                                        // upload
		{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"id":"msg_1","model":"m","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))}, // message
		{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{}`))},                                                                                                                                       // delete (cleanup)
	}}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":  "m",
			"prompt": "describe",
			"files":  []any{"img.png"},
		},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		Files:          &fakeFiles{data: map[string][]byte{"img.png": []byte("pngdata")}},
	}

	if err := c.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(httpCtx.Requests) != 3 {
		t.Fatalf("expected 3 requests (upload, message, delete), got %d", len(httpCtx.Requests))
	}

	// 1. upload to the Files API with the files beta header
	up := httpCtx.Requests[0]
	if !strings.Contains(up.URL.String(), "/files") {
		t.Errorf("request 0 should be a /files upload, got %s", up.URL)
	}
	if up.Header.Get("anthropic-beta") != "files-api-2025-04-14" {
		t.Errorf("upload missing files beta header")
	}
	// The multipart file part must carry the detected MIME type, otherwise the
	// provider stores the file as application/octet-stream and rejects it.
	if ct := uploadPartContentType(t, up); ct != "image/png" {
		t.Errorf("upload file part Content-Type = %q, want image/png", ct)
	}

	// 2. message references the uploaded file via an image block + file_id
	msg := httpCtx.Requests[1]
	if !strings.Contains(msg.URL.String(), "/messages") {
		t.Errorf("request 1 should be /messages, got %s", msg.URL)
	}
	if msg.Header.Get("anthropic-beta") != "files-api-2025-04-14" {
		t.Errorf("message missing files beta header")
	}
	body, _ := io.ReadAll(msg.Body)
	bodyStr := string(body)
	for _, want := range []string{`"type":"image"`, `"type":"file"`, `"file_id":"file_abc"`} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("message body missing %s: %s", want, bodyStr)
		}
	}

	// 3. uploaded file is cleaned up
	del := httpCtx.Requests[2]
	if del.Method != http.MethodDelete || !strings.Contains(del.URL.String(), "/files/file_abc") {
		t.Errorf("request 2 should be DELETE /files/file_abc, got %s %s", del.Method, del.URL)
	}
}

func TestTextPrompt_NodeMetadata(t *testing.T) {
	c := &TextPrompt{}
	md := &contexts.MetadataContext{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"model":        "claude-3-test",
			"prompt":       "hi",
			"maxTokens":    500,
			"outputSchema": `{"type":"object","properties":{"x":{"type":"string"}},"required":["x"]}`,
		},
		Metadata: md,
	}

	if err := c.Setup(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta, ok := md.Metadata.(TextPromptNodeMetadata)
	if !ok {
		t.Fatalf("expected TextPromptNodeMetadata, got %T", md.Metadata)
	}
	if meta.Model != "claude-3-test" {
		t.Errorf("expected model claude-3-test, got %q", meta.Model)
	}
	if meta.MaxTokens != 500 {
		t.Errorf("expected maxTokens 500, got %d", meta.MaxTokens)
	}
	if !meta.StructuredOutput {
		t.Error("expected structuredOutput true")
	}
}

func TestTextPrompt_CodeExecution(t *testing.T) {
	c := &TextPrompt{}

	// Response with a bash code execution tool result referencing a generated file.
	codeExecutionResponseJSON := `{
		"id": "msg_02",
		"type": "message",
		"role": "assistant",
		"model": "claude-3-test",
		"content": [
			{"type": "server_tool_use", "name": "bash_code_execution"},
			{"type": "bash_code_execution_tool_result", "content": {"type": "bash_code_execution_result", "stdout": "ok", "stderr": "", "return_code": 0, "content": [{"type": "code_execution_output", "file_id": "file_gen1"}]}},
			{"type": "text", "text": "Chart created"}
		],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`
	fileMetadataJSON := `{"id": "file_gen1", "type": "file", "filename": "chart.png", "mime_type": "image/png", "size_bytes": 34567, "created_at": "2026-07-10T12:00:00Z", "downloadable": true}`

	t.Run("enabled adds tool and emits artifacts", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(codeExecutionResponseJSON))},
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fileMetadataJSON))},
			},
		}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"model":         "claude-3-test",
				"prompt":        "Make a chart",
				"codeExecution": true,
			},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			ExecutionState: executionState,
		}

		if err := c.Execute(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The request body must carry the code execution tool and the files beta header.
		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		tools, ok := req["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("expected one tool in request, got %v", req["tools"])
		}
		tool := tools[0].(map[string]any)
		if tool["type"] != codeExecutionToolType || tool["name"] != "code_execution" {
			t.Errorf("unexpected tool: %v", tool)
		}
		if got := httpContext.Requests[0].Header.Get("anthropic-beta"); got != anthropicFilesBeta {
			t.Errorf("expected files beta header, got %q", got)
		}

		// The follow-up metadata lookup resolves the artifact.
		if len(httpContext.Requests) != 2 {
			t.Fatalf("expected 2 requests, got %d", len(httpContext.Requests))
		}
		payload := executionState.Payloads[0].(map[string]any)["data"].(MessagePayload)
		if len(payload.Artifacts) != 1 {
			t.Fatalf("expected 1 artifact, got %d", len(payload.Artifacts))
		}
		artifact := payload.Artifacts[0]
		if artifact.FileID != "file_gen1" || artifact.Filename != "chart.png" || artifact.MimeType != "image/png" || artifact.SizeBytes != 34567 {
			t.Errorf("unexpected artifact: %+v", artifact)
		}
		if artifact.DownloadURL != "https://api.anthropic.com/v1/files/file_gen1/content" {
			t.Errorf("unexpected download URL: %s", artifact.DownloadURL)
		}
	})

	t.Run("metadata failure still yields artifact with id and link", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(codeExecutionResponseJSON))},
				{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`boom`))},
			},
		}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"model":         "claude-3-test",
				"prompt":        "Make a chart",
				"codeExecution": true,
			},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			ExecutionState: executionState,
		}

		if err := c.Execute(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		payload := executionState.Payloads[0].(map[string]any)["data"].(MessagePayload)
		if len(payload.Artifacts) != 1 {
			t.Fatalf("expected 1 artifact, got %d", len(payload.Artifacts))
		}
		if payload.Artifacts[0].FileID != "file_gen1" || payload.Artifacts[0].Filename != "" {
			t.Errorf("unexpected artifact: %+v", payload.Artifacts[0])
		}
	})

	t.Run("disabled sends no tools and no artifacts", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"msg_03","content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))},
			},
		}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"model":  "claude-3-test",
				"prompt": "hi",
			},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			ExecutionState: executionState,
		}

		if err := c.Execute(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		if strings.Contains(string(body), `"tools"`) {
			t.Error("expected no tools in request body")
		}
		payload := executionState.Payloads[0].(map[string]any)["data"].(MessagePayload)
		if payload.Artifacts != nil {
			t.Errorf("expected no artifacts, got %+v", payload.Artifacts)
		}
	})
}

func TestExtractGeneratedFileIDs_LegacyBlock(t *testing.T) {
	response := &CreateMessageResponse{
		Content: []MessageContent{
			{Type: "code_execution_tool_result", Content: json.RawMessage(`{"type":"code_execution_result","content":[{"type":"code_execution_output","file_id":"file_a"},{"type":"code_execution_output","file_id":"file_a"},{"type":"code_execution_output","file_id":"file_b"}]}`)},
		},
	}
	ids := extractGeneratedFileIDs(response)
	if len(ids) != 2 || ids[0] != "file_a" || ids[1] != "file_b" {
		t.Errorf("expected deduplicated [file_a file_b], got %v", ids)
	}
}
