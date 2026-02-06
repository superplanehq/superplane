package claude

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// --- Mocks ---

// mockExecutionState implements core.ExecutionStateContext
type mockExecutionState struct {
	EmittedChannel      string
	EmittedType         string
	EmittedPayloads     []any
	Finished            bool
	Failed              bool
	FailReason, FailMsg string
}

func (m *mockExecutionState) IsFinished() bool              { return m.Finished }
func (m *mockExecutionState) SetKV(key, value string) error { return nil }

func (m *mockExecutionState) Emit(channel, payloadType string, payloads []any) error {
	m.EmittedChannel = channel
	m.EmittedType = payloadType
	m.EmittedPayloads = payloads
	return nil
}

func (m *mockExecutionState) Pass() error {
	m.Finished = true
	return nil
}

func (m *mockExecutionState) Fail(reason, message string) error {
	m.Finished = true
	m.Failed = true
	m.FailReason = reason
	m.FailMsg = message
	return nil
}

// --- Tests ---

func TestCreateMessage_Configuration(t *testing.T) {
	c := &CreateMessage{}
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

func TestCreateMessage_Setup(t *testing.T) {
	c := &CreateMessage{}

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

func TestCreateMessage_Execute(t *testing.T) {
	c := &CreateMessage{}

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
		mockResponse    func(*http.Request) *http.Response
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
			mockResponse: func(req *http.Request) *http.Response {
				// Verify request body
				body, _ := io.ReadAll(req.Body)
				var sent CreateMessageRequest
				json.Unmarshal(body, &sent)

				if sent.Model != "claude-3-test" || sent.MaxTokens != 500 || sent.System != "You are a bot" {
					return &http.Response{StatusCode: 400, Body: io.NopCloser(bytes.NewBufferString("bad request body"))}
				}

				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(validResponseJSON)),
				}
			},
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
			mockResponse: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error": {"message": "internal error"}}`)),
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Mocks
			mockState := &mockExecutionState{}
			mockHTTP := &mockHTTPContext{RoundTripFunc: tt.mockResponse}
			mockInt := &mockIntegrationContext{
				config: map[string][]byte{
					"apiKey": []byte("test-key"),
				},
			}

			ctx := core.ExecutionContext{
				Configuration:  tt.config,
				ExecutionState: mockState,
				HTTP:           mockHTTP,
				Integration:    mockInt,
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
				if mockState.EmittedType != MessagePayloadType {
					t.Errorf("expected emitted type %s, got %s", MessagePayloadType, mockState.EmittedType)
				}
				if len(mockState.EmittedPayloads) != 1 {
					t.Errorf("expected 1 payload, got %d", len(mockState.EmittedPayloads))
				} else if tt.validatePayload != nil {
					// Convert payload back to struct for validation
					// In real execution this is passed as any, here we cast it
					payload, ok := mockState.EmittedPayloads[0].(MessagePayload)
					if !ok {
						t.Error("emitted payload is not MessagePayload")
					} else {
						tt.validatePayload(t, payload)
					}
				}
			}
		})
	}
}

func TestExtractMessageText(t *testing.T) {
	// This function is unexported, but accessible within the package_test if package is same
	// If the test file is package claude_test, we can't access it.
	// Assuming package claude per user instruction.

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
