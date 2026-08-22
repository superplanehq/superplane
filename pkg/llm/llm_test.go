package llm

import (
	"context"
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

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNew_RequiresHTTPAndAPIKey(t *testing.T) {
	_, err := New(nil, ProviderOpenAI, Credentials{APIKey: "sk"})
	require.Error(t, err)

	_, err = New(&contexts.HTTPContext{}, ProviderOpenAI, Credentials{})
	require.Error(t, err)

	_, err = New(&contexts.HTTPContext{}, "bedrock", Credentials{APIKey: "sk"})
	require.ErrorContains(t, err, "unsupported llm provider")
}

func TestValidateBaseURL(t *testing.T) {
	require.NoError(t, ValidateBaseURL(""))
	require.NoError(t, ValidateBaseURL("https://proxy.example/v1"))
	require.Error(t, ValidateBaseURL("ftp://proxy.example"))
	require.Error(t, ValidateBaseURL("https://localhost/v1"))
	require.Error(t, ValidateBaseURL("http://127.0.0.1/v1"))
	require.Error(t, ValidateBaseURL("http://10.0.0.5/v1"))
}

func TestNew_UsesDefaultBaseURLs(t *testing.T) {
	httpCtx := &contexts.HTTPContext{}

	anthropic, err := New(httpCtx, ProviderAnthropic, Credentials{APIKey: "sk-ant"})
	require.NoError(t, err)
	assert.Equal(t, ProviderAnthropic, anthropic.Provider())
	assert.Equal(t, defaultAnthropicBaseURL, anthropic.(*anthropicClient).baseURL)

	openai, err := New(httpCtx, ProviderOpenAI, Credentials{APIKey: "sk"})
	require.NoError(t, err)
	assert.Equal(t, defaultOpenAIBaseURL, openai.(*openAICompatClient).baseURL)

	openrouter, err := New(httpCtx, ProviderOpenRouter, Credentials{APIKey: "sk-or"})
	require.NoError(t, err)
	assert.Equal(t, defaultOpenRouterBaseURL, openrouter.(*openAICompatClient).baseURL)
}

func TestAnthropic_ListModelsAndComplete(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"data":[{"id":"claude-sonnet-4-6"},{"id":"claude-opus-4-6"}]}`),
			jsonResponse(http.StatusOK, `{
				"model":"claude-sonnet-4-6",
				"content":[{"type":"text","text":"hello"}],
				"usage":{"input_tokens":12,"output_tokens":3,"cache_read_input_tokens":2,"cache_creation_input_tokens":1}
			}`),
		},
	}

	client, err := New(httpCtx, ProviderAnthropic, Credentials{APIKey: "sk-ant"})
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "claude-sonnet-4-6", models[0].ID)

	resp, err := client.Complete(context.Background(), CompleteRequest{Model: "claude-sonnet-4-6", Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "hello", resp.Text)
	assert.Equal(t, int64(12), resp.Usage.InputTokens)
	assert.Equal(t, int64(3), resp.Usage.OutputTokens)
	assert.Equal(t, int64(2), resp.Usage.CacheReadTokens)
	assert.Equal(t, int64(1), resp.Usage.CacheWriteTokens)
	assert.Equal(t, int64(18), resp.Usage.TotalTokens)

	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "sk-ant", httpCtx.Requests[0].Header.Get("x-api-key"))
	assert.Equal(t, defaultAnthropicBaseURL+"/models", httpCtx.Requests[0].URL.String())
	assert.Equal(t, defaultAnthropicBaseURL+"/messages", httpCtx.Requests[1].URL.String())
}

func TestOpenAI_ListModelsAndComplete(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"data":[{"id":"gpt-5-mini"}]}`),
			jsonResponse(http.StatusOK, `{
				"model":"gpt-5-mini",
				"choices":[{"message":{"content":"ok"}}],
				"usage":{
					"prompt_tokens":10,
					"completion_tokens":4,
					"total_tokens":14,
					"prompt_tokens_details":{"cached_tokens":2},
					"completion_tokens_details":{"reasoning_tokens":1}
				}
			}`),
		},
	}

	client, err := New(httpCtx, ProviderOpenAI, Credentials{APIKey: "sk-openai"})
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "gpt-5-mini", models[0].ID)

	resp, err := client.Complete(context.Background(), CompleteRequest{Model: "gpt-5-mini", Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Text)
	assert.Equal(t, int64(10), resp.Usage.InputTokens)
	assert.Equal(t, int64(4), resp.Usage.OutputTokens)
	assert.Equal(t, int64(2), resp.Usage.CacheReadTokens)
	assert.Equal(t, int64(1), resp.Usage.ReasoningTokens)
	assert.Equal(t, int64(14), resp.Usage.TotalTokens)

	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "Bearer sk-openai", httpCtx.Requests[0].Header.Get("Authorization"))
	assert.Equal(t, defaultOpenAIBaseURL+"/chat/completions", httpCtx.Requests[1].URL.String())
}

func TestOpenRouter_StripsProviderPrefixInUsageRecordAndSetsHeaders(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{
				"model":"openai/gpt-5-mini",
				"choices":[{"message":{"content":"done"}}],
				"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6,"cost":0.000012}
			}`),
		},
	}

	client, err := New(httpCtx, ProviderOpenRouter, Credentials{APIKey: "sk-or"})
	require.NoError(t, err)

	resp, err := client.Complete(context.Background(), CompleteRequest{Model: "openai/gpt-5-mini", Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "openai/gpt-5-mini", resp.Model)
	require.NotNil(t, resp.Usage.CostMicros)
	assert.Equal(t, int64(12), *resp.Usage.CostMicros)

	record := resp.Usage.ToRecord(client.Provider(), resp.Model)
	assert.Equal(t, ProviderOpenRouter, record.Provider)
	assert.Equal(t, "openai/gpt-5-mini", record.Model)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, defaultOpenRouterBaseURL+"/chat/completions", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "https://superplane.com", httpCtx.Requests[0].Header.Get("HTTP-Referer"))
	assert.Equal(t, "SuperPlane", httpCtx.Requests[0].Header.Get("X-Title"))
}

func TestOpenAI_StreamEmitsDeltasAndUsage(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
		`data: {"choices":[{"delta":{"content":"lo"}}]}`,
		`data: {"choices":[],"usage":{"prompt_tokens":2,"completion_tokens":2,"total_tokens":4}}`,
		`data: [DONE]`,
		"",
	}, "\n")
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{jsonResponse(http.StatusOK, sse)}}

	client, err := New(httpCtx, ProviderOpenAI, Credentials{APIKey: "sk"})
	require.NoError(t, err)

	var deltas []string
	var usage *Usage
	err = client.Stream(context.Background(), CompleteRequest{Model: "gpt-5-mini", Prompt: "hi"}, func(ev StreamEvent) error {
		if ev.Delta != "" {
			deltas = append(deltas, ev.Delta)
		}
		if ev.Usage != nil {
			usage = ev.Usage
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Hel", "lo"}, deltas)
	require.NotNil(t, usage)
	assert.Equal(t, int64(4), usage.TotalTokens)

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"stream":true`)
}

func TestAnthropic_StreamEmitsDeltasAndUsage(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hi"}}`,
		`data: {"type":"message_delta","usage":{"input_tokens":8,"output_tokens":1}}`,
		"",
	}, "\n")
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{jsonResponse(http.StatusOK, sse)}}

	client, err := New(httpCtx, ProviderAnthropic, Credentials{APIKey: "sk-ant"})
	require.NoError(t, err)

	var deltas []string
	var usage *Usage
	err = client.Stream(context.Background(), CompleteRequest{Model: "claude-sonnet-4-6", Prompt: "hi"}, func(ev StreamEvent) error {
		if ev.Delta != "" {
			deltas = append(deltas, ev.Delta)
		}
		if ev.Usage != nil {
			usage = ev.Usage
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Hi"}, deltas)
	require.NotNil(t, usage)
	assert.Equal(t, int64(8), usage.InputTokens)
}

func TestUsage_ToRecordFillsTotal(t *testing.T) {
	record := Usage{InputTokens: 10, OutputTokens: 5}.ToRecord(ProviderAnthropic, "claude-sonnet-4-6")
	assert.Equal(t, core.UsageRecord{
		Provider:     ProviderAnthropic,
		Model:        "claude-sonnet-4-6",
		InputTokens:  10,
		OutputTokens: 5,
		TotalTokens:  15,
	}, record)
}

func TestComplete_CustomBaseURL(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{jsonResponse(http.StatusOK, `{"data":[]}`)},
	}
	client, err := New(httpCtx, ProviderOpenAI, Credentials{APIKey: "sk", BaseURL: "https://proxy.example/v1/"})
	require.NoError(t, err)
	_, err = client.ListModels(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "https://proxy.example/v1/models", httpCtx.Requests[0].URL.String())
}

func TestComplete_ErrorStatus(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{jsonResponse(http.StatusUnauthorized, `{"error":{"message":"bad key"}}`)},
	}
	client, err := New(httpCtx, ProviderAnthropic, Credentials{APIKey: "sk"})
	require.NoError(t, err)
	_, err = client.ListModels(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestOpenAIChatRequestJSON(t *testing.T) {
	client := newOpenAICompatClient(&contexts.HTTPContext{}, ProviderOpenAI, defaultOpenAIBaseURL, Credentials{APIKey: "sk"})
	payload, err := json.Marshal(client.chatRequest(CompleteRequest{Model: "gpt-5-mini", System: "sys", Prompt: "hi"}, false))
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"role":"system"`)
	assert.NotContains(t, string(payload), `"stream":true`)
}
