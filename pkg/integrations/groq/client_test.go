package groq

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestNewClient(t *testing.T) {
	t.Run("reads the API key and uses the Groq API base URL", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "gsk-test"},
		})

		require.NoError(t, err)
		assert.Equal(t, "gsk-test", client.APIKey)
		assert.Equal(t, defaultBaseURL, client.BaseURL)
	})

	t.Run("requires an integration context", func(t *testing.T) {
		_, err := NewClient(&contexts.HTTPContext{}, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no integration context")
	})

	t.Run("requires an API key", func(t *testing.T) {
		_, err := NewClient(&contexts.HTTPContext{}, &contexts.IntegrationContext{
			Configuration: map[string]any{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiKey")
	})
}

func TestClientVerify(t *testing.T) {
	t.Run("authenticates the models request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
			}},
		}
		client := &Client{APIKey: "gsk-test", BaseURL: defaultBaseURL, http: httpContext}

		require.NoError(t, client.Verify())
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, defaultBaseURL+"/models", httpContext.Requests[0].URL.String())
		assert.Equal(t, "Bearer gsk-test", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("returns API errors with status context", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"invalid key"}}`)),
			}},
		}
		client := &Client{APIKey: "bad-key", BaseURL: defaultBaseURL, http: httpContext}

		err := client.Verify()

		require.Error(t, err)
		assert.Equal(t, "Groq API rejected the request (401); verify your API key", err.Error())
	})
}

func TestClientListModels(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"object": "list",
				"data": [{
					"id": "llama-3.3-70b-versatile",
					"active": true,
					"context_window": 131072,
					"max_completion_tokens": 32768
				}]
			}`)),
		}},
	}
	client := &Client{APIKey: "gsk-test", BaseURL: defaultBaseURL, http: httpContext}

	models, err := client.ListModels()

	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "llama-3.3-70b-versatile", models[0].ID)
	assert.NotNil(t, models[0].Active)
	assert.Equal(t, 131072, models[0].ContextWindow)
}

func TestClientCreateChatCompletion(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "chatcmpl-123",
				"model": "llama-3.3-70b-versatile",
				"choices": [{
					"index": 0,
					"message": {"role": "assistant", "content": "Hello from Groq"},
					"finish_reason": "stop"
				}],
				"usage": {"prompt_tokens": 4, "completion_tokens": 3, "total_tokens": 7}
			}`)),
		}},
	}
	client := &Client{APIKey: "gsk-test", BaseURL: defaultBaseURL, http: httpContext}
	req := ChatCompletionRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []ChatMessage{
			{Role: "system", Content: "Be concise."},
			{Role: "user", Content: "Say hello."},
		},
	}

	response, err := client.CreateChatCompletion(req)

	require.NoError(t, err)
	assert.Equal(t, "chatcmpl-123", response.ID)
	assert.Equal(t, "Hello from Groq", response.Choices[0].Message.Content)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, defaultBaseURL+"/chat/completions", httpContext.Requests[0].URL.String())
	assert.Equal(t, "Bearer gsk-test", httpContext.Requests[0].Header.Get("Authorization"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&body))
	assert.Equal(t, "llama-3.3-70b-versatile", body["model"])
	assert.Equal(t, []any{
		map[string]any{"role": "system", "content": "Be concise."},
		map[string]any{"role": "user", "content": "Say hello."},
	}, body["messages"])
}
