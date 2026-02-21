package openai

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NewClient(t *testing.T) {
	httpCtx := &contexts.HTTPContext{}

	t.Run("success with default base URL", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-123"},
		}

		client, err := NewClient(httpCtx, integrationCtx)

		require.NoError(t, err)
		assert.Equal(t, "sk-123", client.APIKey)
		assert.Equal(t, defaultBaseURL, client.BaseURL)
	})

	t.Run("custom base URL", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":  "sk-123",
				"baseURL": "https://custom.example.com/v1",
			},
		}

		client, err := NewClient(httpCtx, integrationCtx)

		require.NoError(t, err)
		assert.Equal(t, "https://custom.example.com/v1", client.BaseURL)
	})

	t.Run("empty base URL falls back to default", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":  "sk-123",
				"baseURL": "",
			},
		}

		client, err := NewClient(httpCtx, integrationCtx)

		require.NoError(t, err)
		assert.Equal(t, defaultBaseURL, client.BaseURL)
	})

	t.Run("nil context -> error", func(t *testing.T) {
		_, err := NewClient(httpCtx, nil)
		require.Error(t, err)
	})

	t.Run("missing apiKey -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		_, err := NewClient(httpCtx, integrationCtx)
		require.Error(t, err)
	})
}

func Test__Client__Verify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data": []}`)),
				},
			},
		}

		client := &Client{
			APIKey:  "test-key",
			BaseURL: defaultBaseURL,
			http:    httpCtx,
		}

		err := client.Verify()

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/models")
		assert.Equal(t, "Bearer test-key", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("custom base URL is used in request", func(t *testing.T) {
		customURL := "https://my-provider.example.com/v1"
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data": []}`)),
				},
			},
		}

		client := &Client{
			APIKey:  "test-key",
			BaseURL: customURL,
			http:    httpCtx,
		}

		err := client.Verify()

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, customURL+"/models", httpCtx.Requests[0].URL.String())
	})

	t.Run("unauthorized -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": "invalid api key"}`)),
				},
			},
		}

		client := &Client{
			APIKey:  "bad-key",
			BaseURL: defaultBaseURL,
			http:    httpCtx,
		}

		err := client.Verify()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("server error -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`Internal Server Error`)),
				},
			},
		}

		client := &Client{
			APIKey:  "test-key",
			BaseURL: defaultBaseURL,
			http:    httpCtx,
		}

		err := client.Verify()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func Test__Client__ListModels(t *testing.T) {
	t.Run("returns models", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": [
							{"id": "gpt-4o"},
							{"id": "gpt-4o-mini"}
						]
					}`)),
				},
			},
		}

		client := &Client{
			APIKey:  "test-key",
			BaseURL: defaultBaseURL,
			http:    httpCtx,
		}

		models, err := client.ListModels()

		require.NoError(t, err)
		require.Len(t, models, 2)
		assert.Equal(t, "gpt-4o", models[0].ID)
		assert.Equal(t, "gpt-4o-mini", models[1].ID)
	})

	t.Run("empty list", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data": []}`)),
				},
			},
		}

		client := &Client{
			APIKey:  "test-key",
			BaseURL: defaultBaseURL,
			http:    httpCtx,
		}

		models, err := client.ListModels()

		require.NoError(t, err)
		assert.Empty(t, models)
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(strings.NewReader(`{"error": "rate limited"}`)),
				},
			},
		}

		client := &Client{
			APIKey:  "test-key",
			BaseURL: defaultBaseURL,
			http:    httpCtx,
		}

		_, err := client.ListModels()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "429")
	})
}

func Test__Client__CreateResponse(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "resp_123",
						"model": "gpt-4o",
						"output_text": "Hello there",
						"output": [
							{
								"type": "message",
								"role": "assistant",
								"content": [{"type": "output_text", "text": "Hello there"}]
							}
						],
						"usage": {"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}
					}`)),
				},
			},
		}

		client := &Client{
			APIKey:  "my-secret-key",
			BaseURL: defaultBaseURL,
			http:    httpCtx,
		}

		resp, err := client.CreateResponse("gpt-4o", "Hi")

		require.NoError(t, err)
		assert.Equal(t, "resp_123", resp.ID)
		assert.Equal(t, "gpt-4o", resp.Model)
		assert.Equal(t, "Hello there", resp.OutputText)
		require.Len(t, resp.Output, 1)
		assert.Equal(t, "assistant", resp.Output[0].Role)
		require.NotNil(t, resp.Usage)
		assert.Equal(t, 15, resp.Usage.TotalTokens)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/responses")
		assert.Equal(t, "Bearer my-secret-key", httpCtx.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, "application/json", httpCtx.Requests[0].Header.Get("Content-Type"))

		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		assert.Contains(t, string(body), `"model":"gpt-4o"`)
		assert.Contains(t, string(body), `"input":"Hi"`)
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error": "invalid model"}`)),
				},
			},
		}

		client := &Client{
			APIKey:  "test-key",
			BaseURL: defaultBaseURL,
			http:    httpCtx,
		}

		_, err := client.CreateResponse("invalid-model", "Hi")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}
