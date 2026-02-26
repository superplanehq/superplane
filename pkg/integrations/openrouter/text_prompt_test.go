package openrouter

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestTextPrompt_Name(t *testing.T) {
	c := &TextPrompt{}
	require.Equal(t, "openrouter.textPrompt", c.Name())
}

func TestTextPrompt_Label(t *testing.T) {
	c := &TextPrompt{}
	require.Equal(t, "Text Prompt", c.Label())
}

func TestTextPrompt_ExampleOutput(t *testing.T) {
	c := &TextPrompt{}
	out := c.ExampleOutput()
	require.NotNil(t, out)
	require.Equal(t, MessagePayloadType, out["type"])
	data, ok := out["data"].(map[string]any)
	require.True(t, ok, "example output data should be a map")
	require.Contains(t, data, "id")
	require.Contains(t, data, "model")
	require.Contains(t, data, "text")
	require.Contains(t, data, "usage")
	require.Equal(t, "openai/gpt-4o", data["model"])
	require.Equal(t, "stop", data["finishReason"])
}

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
	}
	for _, field := range config {
		expected, ok := expectedFields[field.Name]
		require.True(t, ok, "unexpected field %s", field.Name)
		require.Equal(t, expected.Required, field.Required, "field %s required", field.Name)
		require.Equal(t, expected.Type, string(field.Type), "field %s type", field.Name)
	}
}

func TestTextPrompt_Setup(t *testing.T) {
	c := &TextPrompt{}

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
	}{
		{
			name: "valid",
			config: map[string]any{
				"model":  "openai/gpt-4o",
				"prompt": "Hello",
			},
			expectError: false,
		},
		{
			name:        "missing model",
			config:      map[string]any{"prompt": "Hi"},
			expectError: true,
		},
		{
			name:        "missing prompt",
			config:      map[string]any{"model": "openai/gpt-4o"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := core.SetupContext{Configuration: tt.config}
			err := c.Setup(ctx)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestTextPrompt_Execute(t *testing.T) {
	c := &TextPrompt{}
	validResp := `{
		"id":"gen-1",
		"model":"openai/gpt-4o",
		"choices":[{"index":0,"message":{"role":"assistant","content":"Generated reply"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}
	}`

	tests := []struct {
		name           string
		config         map[string]any
		responseStatus int
		responseBody   string
		expectError    bool
		expectEmit     bool
		checkPayload   func(t *testing.T, p MessagePayload)
	}{
		{
			name: "success",
			config: map[string]any{
				"model":         "openai/gpt-4o",
				"prompt":        "Say hello",
				"systemMessage": "You are helpful",
				"maxTokens":     256,
			},
			responseStatus: http.StatusOK,
			responseBody:   validResp,
			expectError:    false,
			expectEmit:     true,
			checkPayload: func(t *testing.T, p MessagePayload) {
				require.Equal(t, "Generated reply", p.Text)
				require.Equal(t, "gen-1", p.ID)
				require.Equal(t, "openai/gpt-4o", p.Model)
				require.Equal(t, "stop", p.FinishReason)
				require.NotNil(t, p.Usage)
				require.Equal(t, 5, p.Usage.PromptTokens)
				require.Equal(t, 2, p.Usage.CompletionTokens)
			},
		},
		{
			name: "default maxTokens when zero",
			config: map[string]any{
				"model":  "openai/gpt-4o",
				"prompt": "Hi",
			},
			responseStatus: http.StatusOK,
			responseBody:   validResp,
			expectError:    false,
			expectEmit:     true,
			checkPayload: func(t *testing.T, p MessagePayload) {
				require.Equal(t, "Generated reply", p.Text)
			},
		},
		{
			name: "missing model",
			config: map[string]any{
				"model":  "",
				"prompt": "Hi",
			},
			expectError: true,
		},
		{
			name: "missing prompt",
			config: map[string]any{
				"model":  "openai/gpt-4o",
				"prompt": "",
			},
			expectError: true,
		},
		{
			name: "maxTokens less than 1 fails",
			config: map[string]any{
				"model":     "openai/gpt-4o",
				"prompt":    "Hi",
				"maxTokens": -1,
			},
			expectError: true,
		},
		{
			name: "API error",
			config: map[string]any{
				"model":  "openai/gpt-4o",
				"prompt": "Hi",
			},
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"error":{"message":"internal"}}`,
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
				responses = []*http.Response{{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
				}}
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
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.expectEmit {
				require.Equal(t, MessagePayloadType, execState.Type)
				require.Len(t, execState.Payloads, 1)
				wrapped, ok := execState.Payloads[0].(map[string]any)
				require.True(t, ok)
				data, ok := wrapped["data"].(MessagePayload)
				require.True(t, ok)
				if tt.checkPayload != nil {
					tt.checkPayload(t, data)
				}
			}
		})
	}
}

func TestTextPrompt_Execute_RequestBody(t *testing.T) {
	validResp := `{
		"id":"gen-1",
		"model":"openai/gpt-4o",
		"choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],
		"usage":{}
	}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(validResp)),
		}},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":         "anthropic/claude-3.5-sonnet",
			"prompt":        "User message",
			"systemMessage": "System message",
			"maxTokens":     512,
		},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	}

	err := (&TextPrompt{}).Execute(ctx)
	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)

	bodyBytes, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	var sent ChatCompletionsRequest
	require.NoError(t, json.Unmarshal(bodyBytes, &sent))
	require.Equal(t, "anthropic/claude-3.5-sonnet", sent.Model)
	require.Equal(t, 512, sent.MaxTokens)
	require.Len(t, sent.Messages, 2)
	require.Equal(t, "system", sent.Messages[0].Role)
	require.Equal(t, "System message", sent.Messages[0].Content)
	require.Equal(t, "user", sent.Messages[1].Role)
	require.Equal(t, "User message", sent.Messages[1].Content)
}

func TestExtractMessageText(t *testing.T) {
	tests := []struct {
		name     string
		resp     *ChatCompletionsResponse
		expected string
	}{
		{
			name:     "nil",
			resp:     nil,
			expected: "",
		},
		{
			name:     "empty choices",
			resp:     &ChatCompletionsResponse{Choices: []Choice{}},
			expected: "",
		},
		{
			name: "single choice",
			resp: &ChatCompletionsResponse{
				Choices: []Choice{{
					Message: Message{Role: "assistant", Content: "Hello"},
				}},
			},
			expected: "Hello",
		},
		{
			name: "first choice used",
			resp: &ChatCompletionsResponse{
				Choices: []Choice{
					{Message: Message{Content: "First"}},
					{Message: Message{Content: "Second"}},
				},
			},
			expected: "First",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMessageText(tt.resp)
			require.Equal(t, tt.expected, got)
		})
	}
}
