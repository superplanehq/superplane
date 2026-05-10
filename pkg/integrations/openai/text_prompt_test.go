package openai

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

func Test__CreateResponse__Configuration(t *testing.T) {
	c := &CreateResponse{}

	fields := c.Configuration()

	require.Len(t, fields, 2)

	model := fields[0]
	assert.Equal(t, "model", model.Name)
	assert.Equal(t, "Model", model.Label)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, model.Type)
	assert.True(t, model.Required)
	assert.Equal(t, "gpt-5.2", model.Default)
	require.NotNil(t, model.TypeOptions)
	require.NotNil(t, model.TypeOptions.Resource)
	assert.Equal(t, "model", model.TypeOptions.Resource.Type)

	input := fields[1]
	assert.Equal(t, "input", input.Name)
	assert.Equal(t, "Prompt", input.Label)
	assert.Equal(t, configuration.FieldTypeText, input.Type)
	assert.True(t, input.Required)
}

func Test__CreateResponse__Setup(t *testing.T) {
	c := &CreateResponse{}

	tests := []struct {
		name        string
		config      map[string]any
		expectedErr string
	}{
		{
			name: "valid config",
			config: map[string]any{
				"model": "gpt-5.2",
				"input": "Explain SuperPlane canvases.",
			},
		},
		{
			name: "missing model",
			config: map[string]any{
				"input": "Explain SuperPlane canvases.",
			},
			expectedErr: "model is required",
		},
		{
			name: "missing input",
			config: map[string]any{
				"model": "gpt-5.2",
			},
			expectedErr: "input is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Setup(core.SetupContext{Configuration: tt.config})

			if tt.expectedErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.expectedErr)
		})
	}
}

func Test__CreateResponse__Execute(t *testing.T) {
	c := &CreateResponse{}

	tests := []struct {
		name           string
		config         map[string]any
		responseStatus int
		responseBody   string
		expectedErr    string
		validate       func(*testing.T, *contexts.HTTPContext, *contexts.ExecutionStateContext)
	}{
		{
			name: "success emits response payload",
			config: map[string]any{
				"model": "gpt-5.2",
				"input": "Explain SuperPlane canvases.",
			},
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "resp_123",
				"model": "gpt-5.2-2025-12-11",
				"output_text": "SuperPlane canvases model operational workflows.",
				"usage": {
					"input_tokens": 18,
					"output_tokens": 30,
					"total_tokens": 48
				}
			}`,
			validate: func(t *testing.T, httpCtx *contexts.HTTPContext, execState *contexts.ExecutionStateContext) {
				require.True(t, execState.Finished)
				require.True(t, execState.Passed)
				assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
				assert.Equal(t, ResponsePayloadType, execState.Type)
				require.Len(t, execState.Payloads, 1)

				wrapped, ok := execState.Payloads[0].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, ResponsePayloadType, wrapped["type"])

				payload, ok := wrapped["data"].(ResponsePayload)
				require.True(t, ok)
				assert.Equal(t, "resp_123", payload.ID)
				assert.Equal(t, "gpt-5.2-2025-12-11", payload.Model)
				assert.Equal(t, "SuperPlane canvases model operational workflows.", payload.Text)
				require.NotNil(t, payload.Usage)
				assert.Equal(t, 18, payload.Usage.InputTokens)
				assert.Equal(t, 30, payload.Usage.OutputTokens)
				assert.Equal(t, 48, payload.Usage.TotalTokens)
				require.NotNil(t, payload.Response)
				assert.Equal(t, "resp_123", payload.Response.ID)
				assert.Equal(t, "gpt-5.2-2025-12-11", payload.Response.Model)

				require.Len(t, httpCtx.Requests, 1)
				request := httpCtx.Requests[0]
				assert.Equal(t, http.MethodPost, request.Method)
				assert.Equal(t, defaultBaseURL+"/responses", request.URL.String())
				assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer test-key", request.Header.Get("Authorization"))

				body, err := io.ReadAll(request.Body)
				require.NoError(t, err)

				var sent CreateResponseRequest
				require.NoError(t, json.Unmarshal(body, &sent))
				assert.Equal(t, "gpt-5.2", sent.Model)
				assert.Equal(t, "Explain SuperPlane canvases.", sent.Input)
			},
		},
		{
			name: "missing model returns validation error before request",
			config: map[string]any{
				"input": "Explain SuperPlane canvases.",
			},
			expectedErr: "model is required",
		},
		{
			name: "missing input returns validation error before request",
			config: map[string]any{
				"model": "gpt-5.2",
			},
			expectedErr: "input is required",
		},
		{
			name: "api error is returned",
			config: map[string]any{
				"model": "gpt-5.2",
				"input": "Explain SuperPlane canvases.",
			},
			responseStatus: http.StatusTooManyRequests,
			responseBody:   `{"error":{"code":"insufficient_quota"}}`,
			expectedErr:    "request got 429 code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var responses []*http.Response
			if tt.responseStatus != 0 {
				responses = []*http.Response{
					{
						StatusCode: tt.responseStatus,
						Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
					},
				}
			}

			httpCtx := &contexts.HTTPContext{Responses: responses}
			execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
			integrationCtx := &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			}

			err := c.Execute(core.ExecutionContext{
				Configuration:  tt.config,
				ExecutionState: execState,
				HTTP:           httpCtx,
				Integration:    integrationCtx,
			})

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				if tt.responseStatus == 0 {
					assert.Empty(t, httpCtx.Requests, "validation should fail before any HTTP request is made")
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, httpCtx, execState)
			}
		})
	}
}

func Test__ExtractResponseText(t *testing.T) {
	tests := []struct {
		name     string
		response *OpenAIResponse
		expected string
	}{
		{
			name:     "nil response",
			response: nil,
			expected: "",
		},
		{
			name: "uses output text first",
			response: &OpenAIResponse{
				OutputText: "from output_text",
				Output: []ResponseOutput{
					{Content: []ResponseContent{{Type: "output_text", Text: "from content"}}},
				},
			},
			expected: "from output_text",
		},
		{
			name: "joins text content blocks",
			response: &OpenAIResponse{
				Output: []ResponseOutput{
					{
						Content: []ResponseContent{
							{Type: "output_text", Text: "first"},
							{Type: "text", Text: "second"},
						},
					},
				},
			},
			expected: "first\nsecond",
		},
		{
			name: "joins text across multiple output entries",
			response: &OpenAIResponse{
				Output: []ResponseOutput{
					{Content: []ResponseContent{{Type: "output_text", Text: "first"}}},
					{Content: []ResponseContent{{Type: "text", Text: "second"}}},
				},
			},
			expected: "first\nsecond",
		},
		{
			name: "ignores unsupported and empty content",
			response: &OpenAIResponse{
				Output: []ResponseOutput{
					{
						Content: []ResponseContent{
							{Type: "image", Text: "ignored"},
							{Type: "output_text"},
							{Text: "included"},
						},
					},
				},
			},
			expected: "included",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractResponseText(tt.response))
		})
	}
}
