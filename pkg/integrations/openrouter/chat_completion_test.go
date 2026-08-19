package openrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type fakeFiles struct{ data map[string][]byte }

func (f *fakeFiles) List() ([]string, error) {
	out := make([]string, 0, len(f.data))
	for k := range f.data {
		out = append(out, k)
	}
	return out, nil
}

func (f *fakeFiles) Read(path string) (io.ReadCloser, error) {
	b, ok := f.data[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

const completionBody = `{
	"id": "gen-1",
	"model": "openai/gpt-4o-mini",
	"provider": "OpenAI",
	"choices": [
		{
			"index": 0,
			"finish_reason": "stop",
			"native_finish_reason": "stop",
			"message": {"role": "assistant", "content": "Hello, world!"}
		}
	],
	"usage": {"prompt_tokens": 14, "completion_tokens": 9, "total_tokens": 23, "cost": 0.0000075}
}`

// connectedIntegration is an integration that finished the OAuth flow, so the
// inference key lives in secrets rather than configuration.
func connectedIntegration(config map[string]any) *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: config,
		CurrentSecrets: map[string]core.IntegrationSecret{
			SecretAPIKey: {Name: SecretAPIKey, Value: []byte("sk-or-v1-test")},
		},
	}
}

func execContext(config map[string]any, httpContext *contexts.HTTPContext, state *contexts.ExecutionStateContext) core.ExecutionContext {
	return core.ExecutionContext{
		Logger:         logrus.NewEntry(logrus.New()),
		Configuration:  config,
		HTTP:           httpContext,
		Integration:    connectedIntegration(map[string]any{}),
		ExecutionState: state,
	}
}

func requestBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	body := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

func Test__ChatCompletion__Execute(t *testing.T) {
	c := &ChatCompletion{}

	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "Say hello",
		}, httpContext, state))

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/chat/completions")

		body := requestBody(t, httpContext.Requests[0])
		assert.Equal(t, "openai/gpt-4o-mini", body["model"])
		assert.NotContains(t, body, "models")
		assert.NotContains(t, body, "provider")
		assert.Equal(t, []any{
			map[string]any{"role": "user", "content": "Say hello"},
		}, body["messages"])

		require.Len(t, state.Payloads, 1)
		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, "gen-1", payload.ID)
		assert.Equal(t, "Hello, world!", payload.Text)
		assert.Equal(t, "OpenAI", payload.Provider)
		assert.Equal(t, "stop", payload.FinishReason)
		assert.Equal(t, ChatCompletionPayloadType, state.Type)
	})

	t.Run("system prompt is sent ahead of the prompt", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		err := c.Execute(execContext(map[string]any{
			"model":        "openai/gpt-4o-mini",
			"prompt":       "Say hello",
			"systemPrompt": "Be terse",
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])
		assert.Equal(t, []any{
			map[string]any{"role": "system", "content": "Be terse"},
			map[string]any{"role": "user", "content": "Say hello"},
		}, body["messages"])
	})

	t.Run("provider routing is sent as the provider object", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		err := c.Execute(execContext(map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "Say hello",
			"provider": map[string]any{
				"sort":              "price",
				"only":              []any{"azure", "openai"},
				"ignore":            []any{"together"},
				"allowFallbacks":    false,
				"requireParameters": true,
				"dataCollection":    "deny",
			},
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])
		assert.Equal(t, map[string]any{
			"sort":               "price",
			"only":               []any{"azure", "openai"},
			"ignore":             []any{"together"},
			"allow_fallbacks":    false,
			"require_parameters": true,
			"data_collection":    "deny",
		}, body["provider"])
	})

	// Node configuration arrives as JSON, so the numeric fields reach Execute as
	// float64 or json.Number rather than as Go ints.
	t.Run("numeric fields survive the JSON round trip", func(t *testing.T) {
		for name, value := range map[string]any{
			"float64":     float64(1024),
			"json.Number": json.Number("1024"),
			"int":         1024,
		} {
			t.Run(name, func(t *testing.T) {
				httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

				err := c.Execute(execContext(map[string]any{
					"model":       "openai/gpt-4o-mini",
					"prompt":      "Say hello",
					"maxTokens":   value,
					"temperature": value,
				}, httpContext, &contexts.ExecutionStateContext{}))

				require.NoError(t, err)
				body := requestBody(t, httpContext.Requests[0])
				assert.Equal(t, float64(1024), body["max_tokens"])
				assert.Equal(t, float64(1024), body["temperature"])
			})
		}
	})

	t.Run("numeric fields are omitted when unset", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		err := c.Execute(execContext(map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "Say hello",
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])
		assert.NotContains(t, body, "max_tokens")
		assert.NotContains(t, body, "temperature")
	})

	t.Run("web search enables the web plugin and flattens citations", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"gen-w","model":"m","provider":"P","choices":[{"index":0,"finish_reason":"stop","message":{
				"role":"assistant","content":"Rates rose.","annotations":[
					{"type":"url_citation","url_citation":{"url":"https://example.com/a","title":"Report A","content":"..."}},
					{"type":"other","url_citation":{"url":"https://example.com/ignored","title":"Ignored"}}
				]}}]}`),
		}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{
			"model":         "openai/gpt-4o-mini",
			"prompt":        "What happened?",
			"webSearch":     true,
			"webMaxResults": 3,
		}, httpContext, state))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])
		assert.Equal(t, []any{map[string]any{"id": "web", "max_results": float64(3)}}, body["plugins"])

		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		require.Len(t, payload.Citations, 1)
		assert.Equal(t, "https://example.com/a", payload.Citations[0].URL)
		assert.Equal(t, "Report A", payload.Citations[0].Title)
	})

	t.Run("web search is omitted when off", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		require.NoError(t, c.Execute(execContext(map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "hi",
		}, httpContext, &contexts.ExecutionStateContext{})))

		assert.NotContains(t, requestBody(t, httpContext.Requests[0]), "plugins")
	})

	t.Run("structured output sends a strict json schema and parses the reply", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"gen-s","model":"m","provider":"P","choices":[{"index":0,"finish_reason":"stop",
				"message":{"role":"assistant","content":"{\"city\":\"Nairobi\"}"}}]}`),
		}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{
			"model":        "openai/gpt-4o-mini",
			"prompt":       "Where?",
			"outputSchema": `{"type":"object","properties":{"city":{"type":"string"}}}`,
		}, httpContext, state))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])

		format := body["response_format"].(map[string]any)
		assert.Equal(t, "json_schema", format["type"])
		schema := format["json_schema"].(map[string]any)
		assert.Equal(t, "structured_output", schema["name"])
		assert.Equal(t, true, schema["strict"])
		assert.NotNil(t, schema["schema"])

		// A provider that ignores response_format would return prose, so routing
		// is pinned to providers that honour it.
		assert.Equal(t, map[string]any{"require_parameters": true}, body["provider"])

		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, map[string]any{"city": "Nairobi"}, payload.Parsed)
	})

	t.Run("structured output keeps routing the user configured", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		require.NoError(t, c.Execute(execContext(map[string]any{
			"model":        "openai/gpt-4o-mini",
			"prompt":       "Where?",
			"outputSchema": `{"type":"object","properties":{"city":{"type":"string"}}}`,
			"provider":     map[string]any{"sort": "price", "requireParameters": false},
		}, httpContext, &contexts.ExecutionStateContext{})))

		provider := requestBody(t, httpContext.Requests[0])["provider"].(map[string]any)
		assert.Equal(t, "price", provider["sort"])
		// An explicit opt-out is respected rather than overridden.
		assert.Equal(t, false, provider["require_parameters"])
	})

	t.Run("a refusal is not parsed as structured output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"gen-r","model":"m","choices":[{"index":0,"finish_reason":"stop",
				"message":{"role":"assistant","content":null,"refusal":"I cannot help with that."}}]}`),
		}}
		state := &contexts.ExecutionStateContext{}

		require.NoError(t, c.Execute(execContext(map[string]any{
			"model":        "m",
			"prompt":       "no",
			"outputSchema": `{"type":"object","properties":{"a":{"type":"string"}}}`,
		}, httpContext, state)))

		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, "I cannot help with that.", payload.Text)
		assert.Nil(t, payload.Parsed)
	})

	t.Run("the balanced sort is omitted rather than sent", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		err := c.Execute(execContext(map[string]any{
			"model":    "openai/gpt-4o-mini",
			"prompt":   "Say hello",
			"provider": map[string]any{"sort": sortAuto, "dataCollection": "deny"},
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])
		assert.Equal(t, map[string]any{"data_collection": "deny"}, body["provider"])
	})

	t.Run("fallback models replace the single model field", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		err := c.Execute(execContext(map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "Say hello",
			// The primary repeated as a fallback must not be retried as its own fallback.
			"models": []any{"anthropic/claude-sonnet-4.5", "openai/gpt-4o-mini", ""},
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])
		assert.NotContains(t, body, "model")
		assert.Equal(t, []any{"openai/gpt-4o-mini", "anthropic/claude-sonnet-4.5"}, body["models"])
	})

	t.Run("pdfs are inlined as base64 file parts", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}
		ctx := execContext(map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "Summarize",
			"files":  []any{"doc.pdf"},
		}, httpContext, &contexts.ExecutionStateContext{})
		ctx.Files = &fakeFiles{data: map[string][]byte{"doc.pdf": []byte("%PDF-1.4")}}

		require.NoError(t, c.Execute(ctx))

		// No upload call: OpenRouter has no Files API.
		require.Len(t, httpContext.Requests, 1)

		body := requestBody(t, httpContext.Requests[0])
		parts := body["messages"].([]any)[0].(map[string]any)["content"].([]any)
		require.Len(t, parts, 2)
		assert.Equal(t, "text", parts[0].(map[string]any)["type"])

		file := parts[1].(map[string]any)
		assert.Equal(t, "file", file["type"])
		assert.Equal(t, "doc.pdf", file["file"].(map[string]any)["filename"])
		assert.Equal(t, "data:application/pdf;base64,JVBERi0xLjQ=", file["file"].(map[string]any)["file_data"])
	})

	t.Run("images are inlined as image parts", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}
		ctx := execContext(map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "Describe",
			"files":  []any{"shot.png"},
		}, httpContext, &contexts.ExecutionStateContext{})
		ctx.Files = &fakeFiles{data: map[string][]byte{"shot.png": []byte("\x89PNG\r\n\x1a\n")}}

		require.NoError(t, c.Execute(ctx))

		body := requestBody(t, httpContext.Requests[0])
		parts := body["messages"].([]any)[0].(map[string]any)["content"].([]any)
		require.Len(t, parts, 2)

		image := parts[1].(map[string]any)
		assert.Equal(t, "image_url", image["type"])
		assert.Contains(t, image["image_url"].(map[string]any)["url"], "data:image/png;base64,")
	})

	// A file part would route plain text through OpenRouter's document parser,
	// which is a paid feature that rejects the request below a minimum balance.
	t.Run("text files become prompt text rather than file parts", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}
		ctx := execContext(map[string]any{
			"model":  "openai/gpt-4o-mini",
			"prompt": "Summarize",
			"files":  []any{"notes.md"},
		}, httpContext, &contexts.ExecutionStateContext{})
		ctx.Files = &fakeFiles{data: map[string][]byte{"notes.md": []byte("the codeword is BANANA")}}

		require.NoError(t, c.Execute(ctx))

		body := requestBody(t, httpContext.Requests[0])
		parts := body["messages"].([]any)[0].(map[string]any)["content"].([]any)
		require.Len(t, parts, 2)

		attached := parts[1].(map[string]any)
		assert.Equal(t, "text", attached["type"])
		assert.NotContains(t, attached, "file")
		assert.Equal(t, "--- notes.md ---\nthe codeword is BANANA", attached["text"])
	})

	t.Run("reasoning stands in when content is null", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"gen-2","model":"m","provider":"P","choices":[
				{"index":0,"finish_reason":"length","message":{"role":"assistant","content":null,"reasoning":"partial thinking"}}
			]}`),
		}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{
			"model":  "m",
			"prompt": "Think",
		}, httpContext, state))

		require.NoError(t, err)
		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, "partial thinking", payload.Text)
		assert.Equal(t, "partial thinking", payload.Reasoning)
	})

	t.Run("null content with no reasoning fails instead of emitting a blank result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"gen-3","model":"m","choices":[
				{"index":0,"finish_reason":"length","message":{"role":"assistant","content":null}}
			]}`),
		}}

		err := c.Execute(execContext(map[string]any{
			"model":  "m",
			"prompt": "Think",
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "token budget was exhausted")
	})

	t.Run("no choices", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"gen-4","model":"m","choices":[]}`),
		}}

		err := c.Execute(execContext(map[string]any{"model": "m", "prompt": "hi"}, httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no choices")
	})
}

func Test__ChatCompletion__Errors(t *testing.T) {
	c := &ChatCompletion{}

	t.Run("unauthorized", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusUnauthorized, `{"error":{"message":"User not found.","code":401}}`),
		}}

		err := c.Execute(execContext(map[string]any{"model": "m", "prompt": "hi"}, httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "User not found.")
	})

	t.Run("invalid model id", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusBadRequest, `{"error":{"message":"does/not-exist is not a valid model ID","code":400}}`),
		}}

		err := c.Execute(execContext(map[string]any{"model": "does/not-exist", "prompt": "hi"}, httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a valid model ID")
	})

	t.Run("rate limited carries the backoff hint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusTooManyRequests, `{"error":{"message":"Rate limit exceeded","code":429,"metadata":{"retry_after_seconds":30}}}`),
		}}

		err := c.Execute(execContext(map[string]any{"model": "m", "prompt": "hi"}, httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "429")
		assert.Contains(t, err.Error(), "retry after 30 seconds")
	})

	t.Run("no allowed provider serves the model", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusNotFound, `{"error":{"message":"No allowed providers are available for the selected model.","code":404,"metadata":{"available_providers":["nvidia"]}}}`),
		}}

		err := c.Execute(execContext(map[string]any{
			"model":    "m",
			"prompt":   "hi",
			"provider": map[string]any{"only": []any{"nonexistent"}, "allowFallbacks": false},
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
		assert.Contains(t, err.Error(), "available providers: [nvidia]")
	})
}

func Test__ChatCompletion__Setup(t *testing.T) {
	c := &ChatCompletion{}

	setup := func(config map[string]any) error {
		return c.Setup(core.SetupContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: config,
			Metadata:      &contexts.MetadataContext{},
		})
	}

	t.Run("records the model and whether routing is configured", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Logger: logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"model":    "openai/gpt-4o-mini",
				"prompt":   "hi",
				"provider": map[string]any{"sort": "price"},
			},
			Metadata: metadata,
		})

		require.NoError(t, err)
		assert.Equal(t, ChatCompletionNodeMetadata{Model: "openai/gpt-4o-mini", ProviderRouting: true}, metadata.Metadata)
	})

	t.Run("model is required", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"prompt": "hi"}), "model is required")
	})

	t.Run("prompt is required", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"model": "m"}), "prompt is required")
	})

	t.Run("rejects an unknown sort", func(t *testing.T) {
		err := setup(map[string]any{
			"model":    "m",
			"prompt":   "hi",
			"provider": map[string]any{"sort": "cheapest"},
		})
		require.ErrorContains(t, err, "invalid sort")
	})

	t.Run("rejects an unknown data collection policy", func(t *testing.T) {
		err := setup(map[string]any{
			"model":    "m",
			"prompt":   "hi",
			"provider": map[string]any{"dataCollection": "maybe"},
		})
		require.ErrorContains(t, err, "invalid data collection policy")
	})

	t.Run("rejects a provider that is both allowed and excluded", func(t *testing.T) {
		err := setup(map[string]any{
			"model":  "m",
			"prompt": "hi",
			"provider": map[string]any{
				"only":   []any{"azure"},
				"ignore": []any{"azure"},
			},
		})
		require.ErrorContains(t, err, "both allowed and excluded")
	})

	t.Run("rejects an invalid output schema", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{
			"model":        "m",
			"prompt":       "hi",
			"outputSchema": "{not json",
		}), "schema")
	})

	t.Run("defers validating a schema that still holds an expression", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{
			"model":        "m",
			"prompt":       "hi",
			"outputSchema": "{{ inputs.schema }}",
		}))
	})

	t.Run("records structured output and web search", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Logger: logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"model":        "openai/gpt-4o-mini",
				"prompt":       "hi",
				"webSearch":    true,
				"outputSchema": `{"type":"object","properties":{"a":{"type":"string"}}}`,
			},
			Metadata: metadata,
		})

		require.NoError(t, err)
		assert.Equal(t, ChatCompletionNodeMetadata{
			Model:            "openai/gpt-4o-mini",
			StructuredOutput: true,
			WebSearch:        true,
		}, metadata.Metadata)
	})

	t.Run("rejects a file that is not in the repository", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Logger: logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"model":  "m",
				"prompt": "hi",
				"files":  []any{"missing.pdf"},
			},
			Metadata: &contexts.MetadataContext{},
			Files:    &fakeFiles{data: map[string][]byte{"doc.pdf": []byte("%PDF-1.4")}},
		})
		require.ErrorContains(t, err, "not found in app repository")
	})
}
