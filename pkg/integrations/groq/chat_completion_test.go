package groq

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type groqUsageRecorder struct {
	records []core.UsageRecord
}

func (r *groqUsageRecorder) Record(record core.UsageRecord) error {
	r.records = append(r.records, record)
	return nil
}

func TestChatCompletionConfiguration(t *testing.T) {
	fields := (&ChatCompletion{}).Configuration()
	fieldTypes := make(map[string]string, len(fields))
	for _, field := range fields {
		fieldTypes[field.Name] = field.Type
	}

	assert.Equal(t, configuration.FieldTypeIntegrationResource, fieldTypes["model"])
	assert.Equal(t, configuration.FieldTypeText, fieldTypes["input"])
	assert.Equal(t, configuration.FieldTypeText, fieldTypes["systemPrompt"])
}

func TestChatCompletionSetupValidation(t *testing.T) {
	tests := []struct {
		name          string
		configuration map[string]any
		wantError     string
	}{
		{
			name:          "missing model",
			configuration: map[string]any{"input": "Hello"},
			wantError:     "model is required",
		},
		{
			name:          "missing input",
			configuration: map[string]any{"model": "llama-3.3-70b-versatile"},
			wantError:     "input is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&ChatCompletion{}).Setup(core.SetupContext{Configuration: tt.configuration})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}

	t.Run("accepts a valid configuration and records model metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := (&ChatCompletion{}).Setup(core.SetupContext{
			Configuration: map[string]any{
				"model":        "llama-3.3-70b-versatile",
				"input":        "Hello",
				"systemPrompt": "Be concise.",
			},
			Metadata: metadata,
		})

		require.NoError(t, err)
		assert.Equal(t, ChatCompletionNodeMetadata{Model: "llama-3.3-70b-versatile"}, metadata.Metadata)
	})
}

func TestChatCompletionExecute(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id":"chatcmpl-123",
				"model":"llama-3.3-70b-versatile",
				"choices":[{
					"index":0,
					"message":{"role":"assistant","content":"Hello from Groq"},
					"finish_reason":"stop"
				}],
				"usage":{"prompt_tokens":4,"completion_tokens":3,"total_tokens":7}
			}`)),
		}},
	}
	usage := &groqUsageRecorder{}
	executionState := &contexts.ExecutionStateContext{}
	executionContext := core.ExecutionContext{
		Configuration: map[string]any{
			"model":        "llama-3.3-70b-versatile",
			"input":        "Say hello.",
			"systemPrompt": "Be concise.",
		},
		ExecutionState: executionState,
		HTTP:           httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "gsk-test"},
		},
		Usage: usage,
	}

	err := (&ChatCompletion{}).Execute(executionContext)

	require.NoError(t, err)
	assert.True(t, executionState.Finished)
	assert.True(t, executionState.Passed)
	assert.Equal(t, ChatCompletionPayloadType, executionState.Type)
	require.Len(t, executionState.Payloads, 1)
	wrapper, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	payload, ok := wrapper["data"].(ChatCompletionPayload)
	require.True(t, ok)
	assert.Equal(t, "chatcmpl-123", payload.ID)
	assert.Equal(t, "llama-3.3-70b-versatile", payload.Model)
	assert.Equal(t, "Hello from Groq", payload.Text)
	require.NotNil(t, payload.Usage)
	assert.Equal(t, 7, payload.Usage.TotalTokens)
	require.NotNil(t, payload.Response)
	assert.Equal(t, "chatcmpl-123", payload.Response.ID)

	require.Len(t, usage.records, 1)
	assert.Equal(t, usage.records[0], core.UsageRecord{
		Provider:     "groq",
		Model:        "llama-3.3-70b-versatile",
		InputTokens:  4,
		OutputTokens: 3,
		TotalTokens:  7,
	})

	require.Len(t, httpContext.Requests, 1)
	request := httpContext.Requests[0]
	assert.Equal(t, http.MethodPost, request.Method)
	assert.Equal(t, defaultBaseURL+"/chat/completions", request.URL.String())
	assert.Equal(t, "Bearer gsk-test", request.Header.Get("Authorization"))
	var body ChatCompletionRequest
	require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
	require.Len(t, body.Messages, 2)
	assert.Equal(t, ChatMessage{Role: "system", Content: "Be concise."}, body.Messages[0])
	assert.Equal(t, ChatMessage{Role: "user", Content: "Say hello."}, body.Messages[1])
}

func TestChatCompletionExecuteAPIError(t *testing.T) {
	executionState := &contexts.ExecutionStateContext{}
	err := (&ChatCompletion{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"model": "llama-3.3-70b-versatile",
			"input": "Hello",
		},
		ExecutionState: executionState,
		HTTP: &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"rate limited"}}`)),
		}}},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "gsk-test"}},
	})

	require.Error(t, err)
	assert.Equal(t, "Groq API rate limit reached (429); wait and try again", err.Error())
	assert.Empty(t, executionState.Payloads)
}

func TestChatCompletionExecuteRequiresChoices(t *testing.T) {
	err := (&ChatCompletion{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"model": "llama-3.3-70b-versatile",
			"input": "Hello",
		},
		ExecutionState: &contexts.ExecutionStateContext{},
		HTTP: &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"chatcmpl-123","model":"llama-3.3-70b-versatile","choices":[]}`)),
		}}},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "gsk-test"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}
