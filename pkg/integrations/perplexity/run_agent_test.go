package perplexity

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func agentResponseJSON(id, model, outputText string) string {
	return `{
		"id": "` + id + `",
		"model": "` + model + `",
		"status": "completed",
		"output": [
			{
				"content": [
					{
						"type": "output_text",
						"text": "` + outputText + `",
						"annotations": [
							{"type": "citation", "url": "https://example.com/source"}
						]
					}
				]
			}
		],
		"usage": {"input_tokens": 100, "output_tokens": 50, "total_tokens": 150}
	}`
}

func TestRunAgent_WithPreset(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(agentResponseJSON("resp_1", "openai/gpt-5.2", "Answer here"))),
			},
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "pplx-test"},
	}

	c := &runAgent{}
	err := c.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"modelSource": "preset",
			"preset":      "pro-search",
			"input":       "What is Go?",
			"webSearch":   true,
			"fetchUrl":    true,
		},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	})

	require.NoError(t, err)

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)

	var sent AgentRequest
	require.NoError(t, json.Unmarshal(body, &sent))

	assert.Equal(t, "pro-search", sent.Preset)
	assert.Empty(t, sent.Model)
	assert.Equal(t, "What is Go?", sent.Input)
}

func TestRunAgent_WithModel(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(agentResponseJSON("resp_2", "anthropic/claude-haiku-4-5", "Model answer"))),
			},
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "pplx-test"},
	}

	c := &runAgent{}
	err := c.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"modelSource": "model",
			"model":       "anthropic/claude-haiku-4-5",
			"input":       "Explain recursion",
			"webSearch":   true,
			"fetchUrl":    false,
		},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	})

	require.NoError(t, err)

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)

	var sent AgentRequest
	require.NoError(t, json.Unmarshal(body, &sent))

	assert.Empty(t, sent.Preset)
	assert.Equal(t, "anthropic/claude-haiku-4-5", sent.Model)
}

func TestRunAgent_ToolFlags(t *testing.T) {
	tests := []struct {
		name          string
		webSearch     bool
		fetchURL      bool
		expectedTools []string
	}{
		{
			name:          "both enabled",
			webSearch:     true,
			fetchURL:      true,
			expectedTools: []string{"web_search", "fetch_url"},
		},
		{
			name:          "only web_search",
			webSearch:     true,
			fetchURL:      false,
			expectedTools: []string{"web_search"},
		},
		{
			name:          "only fetch_url",
			webSearch:     false,
			fetchURL:      true,
			expectedTools: []string{"fetch_url"},
		},
		{
			name:          "neither",
			webSearch:     false,
			fetchURL:      false,
			expectedTools: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(agentResponseJSON("r", "m", "ok"))),
					},
				},
			}
			execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
			integrationCtx := &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "key"},
			}

			c := &runAgent{}
			err := c.Execute(core.ExecutionContext{
				Configuration: map[string]any{
					"modelSource": "preset",
					"preset":      "fast-search",
					"input":       "test",
					"webSearch":   tt.webSearch,
					"fetchUrl":    tt.fetchURL,
				},
				ExecutionState: execState,
				HTTP:           httpCtx,
				Integration:    integrationCtx,
			})
			require.NoError(t, err)

			body, _ := io.ReadAll(httpCtx.Requests[0].Body)
			var sent AgentRequest
			require.NoError(t, json.Unmarshal(body, &sent))

			sentToolTypes := make([]string, 0, len(sent.Tools))
			for _, tool := range sent.Tools {
				sentToolTypes = append(sentToolTypes, tool.Type)
			}
			assert.Equal(t, tt.expectedTools, sentToolTypes)
		})
	}
}

func TestRunAgent_OutputParsing(t *testing.T) {
	responseJSON := `{
		"id": "resp_abc",
		"model": "openai/gpt-5.1",
		"status": "completed",
		"output": [
			{
				"content": [
					{
						"type": "output_text",
						"text": "Recent developments in AI include...",
						"annotations": [
							{"type": "citation", "url": "https://source1.com"},
							{"type": "citation", "url": "https://source2.com"}
						]
					}
				]
			}
		],
		"usage": {"input_tokens": 3681, "output_tokens": 780, "total_tokens": 4461, "cost": {"total_cost": 0.05}}
	}`

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(responseJSON)),
			},
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "pplx-test"},
	}

	c := &runAgent{}
	err := c.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"modelSource": "preset",
			"preset":      "pro-search",
			"input":       "Latest AI news",
			"webSearch":   true,
			"fetchUrl":    true,
		},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, AgentPayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)

	wrapped := execState.Payloads[0].(map[string]any)
	data := wrapped["data"].(agentPayload)

	assert.Equal(t, "resp_abc", data.ID)
	assert.Equal(t, "openai/gpt-5.1", data.Model)
	assert.Equal(t, "completed", data.Status)
	assert.Equal(t, "Recent developments in AI include...", data.Text)
	require.Len(t, data.Citations, 2)
	assert.Equal(t, "https://source1.com", data.Citations[0].URL)
	assert.Equal(t, "https://source2.com", data.Citations[1].URL)
	require.NotNil(t, data.Usage)
	assert.Equal(t, 4461, data.Usage.TotalTokens)
}

func TestRunAgent_MissingPreset(t *testing.T) {
	c := &runAgent{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"modelSource": "preset", "preset": "", "input": "test query"},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "preset is required")
}

func TestRunAgent_MissingModel(t *testing.T) {
	c := &runAgent{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"modelSource": "model", "model": "", "input": "test query"},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
}

func TestRunAgent_MissingInput(t *testing.T) {
	c := &runAgent{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"modelSource": "preset", "preset": "pro-search", "input": ""},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "input is required")
}

func TestRunAgent_APIError(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"error":"internal server error"}`)),
			},
		},
	}

	c := &runAgent{}
	err := c.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"modelSource": "preset",
			"preset":      "pro-search",
			"input":       "test query",
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
