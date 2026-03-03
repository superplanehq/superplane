package openrouter

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestNewClient(t *testing.T) {
	httpCtx := &contexts.HTTPContext{}

	tests := []struct {
		name           string
		integrationCtx core.IntegrationContext
		expectError    bool
	}{
		{
			name: "success",
			integrationCtx: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "sk-or-123"},
			},
			expectError: false,
		},
		{
			name:           "nil context",
			integrationCtx: nil,
			expectError:    true,
		},
		{
			name: "missing API key",
			integrationCtx: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
			expectError: true,
		},
		{
			name: "config not string",
			integrationCtx: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": 42},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(httpCtx, tt.integrationCtx)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, client)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, "sk-or-123", client.APIKey)
			assert.Equal(t, defaultBaseURL, client.BaseURL)
		})
	}
}

func TestClient_Verify(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectError    bool
	}{
		{"success", http.StatusOK, `{}`, false},
		{"unauthorized", http.StatusUnauthorized, `{"error":{"message":"Invalid API key","code":401}}`, true},
		{"server error", http.StatusInternalServerError, `{"error":{"message":"internal"}}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
				}},
			}
			client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

			err := client.Verify()

			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClient_ListModels(t *testing.T) {
	jsonResp := `{"data":[{"id":"openai/gpt-4o"},{"id":"anthropic/claude-3.5-sonnet"}]}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
		}},
	}
	client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

	models, err := client.ListModels()
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "openai/gpt-4o", models[0].ID)
	assert.Equal(t, "anthropic/claude-3.5-sonnet", models[1].ID)
}

func TestClient_ListModels_InvalidJSON(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("not json")),
		}},
	}
	client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

	_, err := client.ListModels()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestClient_ChatCompletions(t *testing.T) {
	jsonResp := `{
		"id":"gen-1",
		"model":"openai/gpt-4o",
		"choices":[{"index":0,"message":{"role":"assistant","content":"Hi there"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}
	}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
		}},
	}
	client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

	req := ChatCompletionsRequest{
		Model: "openai/gpt-4o",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	}
	resp, err := client.ChatCompletions(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "gen-1", resp.ID)
	assert.Equal(t, "openai/gpt-4o", resp.Model)
	require.Len(t, resp.Choices, 1)
	assert.Equal(t, "Hi there", resp.Choices[0].Message.Content)
	assert.Equal(t, "stop", resp.Choices[0].FinishReason)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 3, resp.Usage.CompletionTokens)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	assert.Equal(t, "Bearer test-key", httpCtx.Requests[0].Header.Get("Authorization"))
}

func TestClient_ChatCompletions_ErrorResponse(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error":{"message":"Invalid API key","code":401}}`)),
		}},
	}
	client := &Client{APIKey: "bad-key", BaseURL: defaultBaseURL, http: httpCtx}

	_, err := client.ChatCompletions(ChatCompletionsRequest{Model: "openai/gpt-4o", Messages: []Message{{Role: "user", Content: "Hi"}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

func TestClient_GetRemainingCredits(t *testing.T) {
	jsonResp := `{"data":{"total_credits":100.0,"total_usage":25.5}}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
		}},
	}
	client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

	resp, err := client.GetRemainingCredits()
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 100.0, resp.Data.TotalCredits)
	assert.Equal(t, 25.5, resp.Data.TotalUsage)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, defaultBaseURL+"/credits", httpCtx.Requests[0].URL.String())
}

func TestClient_GetRemainingCredits_InvalidJSON(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("not json")),
		}},
	}
	client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

	_, err := client.GetRemainingCredits()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestClient_GetCurrentKeyDetails(t *testing.T) {
	jsonResp := `{
		"data":{
			"label":"My Key",
			"limit":100,
			"usage":25,
			"usage_daily":5,
			"usage_weekly":20,
			"usage_monthly":25,
			"is_free_tier":false,
			"is_management_key":false,
			"limit_remaining":75,
			"limit_reset":"monthly",
			"include_byok_in_limit":false,
			"expires_at":"2026-12-31T00:00:00Z"
		}
	}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
		}},
	}
	client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

	resp, err := client.GetCurrentKeyDetails()
	require.NoError(t, err)
	require.NotNil(t, resp)
	d := resp.Data
	assert.Equal(t, "My Key", d.Label)
	require.NotNil(t, d.Limit)
	assert.Equal(t, 100.0, *d.Limit)
	assert.Equal(t, 25.0, d.Usage)
	assert.Equal(t, 5.0, d.UsageDaily)
	require.NotNil(t, d.LimitRemaining)
	assert.Equal(t, 75.0, *d.LimitRemaining)
	require.NotNil(t, d.LimitReset)
	assert.Equal(t, "monthly", *d.LimitReset)
	require.NotNil(t, d.ExpiresAt)
	assert.Equal(t, "2026-12-31T00:00:00Z", *d.ExpiresAt)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, defaultBaseURL+"/key", httpCtx.Requests[0].URL.String())
}

func TestClient_GetCurrentKeyDetails_InvalidJSON(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("not json")),
		}},
	}
	client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

	_, err := client.GetCurrentKeyDetails()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestClient_doRequest_NonJSONErrorBody(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString("plain text error")),
		}},
	}
	client := &Client{APIKey: "test-key", BaseURL: defaultBaseURL, http: httpCtx}

	err := client.Verify()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "plain text error")
}
