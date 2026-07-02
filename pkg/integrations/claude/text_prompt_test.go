package claude

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
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
