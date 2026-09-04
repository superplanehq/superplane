package opencodego

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
	"id": "chatcmpl-1",
	"object": "chat.completion",
	"created": 1787263920,
	"model": "glm-5.2",
	"choices": [
		{
			"index": 0,
			"finish_reason": "stop",
			"message": {"role": "assistant", "content": "Hello, world!"}
		}
	],
	"usage": {"prompt_tokens": 14, "completion_tokens": 9, "total_tokens": 23}
}`

func execContext(config map[string]any, httpContext *contexts.HTTPContext, state *contexts.ExecutionStateContext) core.ExecutionContext {
	return core.ExecutionContext{
		Logger:         logrus.NewEntry(logrus.New()),
		Configuration:  config,
		HTTP:           httpContext,
		Integration:    connectedIntegration(map[string]any{"apiKey": "oc-test-key"}),
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
			"model":  "glm-5.2",
			"prompt": "Say hello",
		}, httpContext, state))

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		request := httpContext.Requests[0]
		assert.Contains(t, request.URL.String(), baseURL+"/chat/completions")
		assert.Equal(t, "Bearer oc-test-key", request.Header.Get("Authorization"))
		assert.Empty(t, request.Header.Get("x-api-key"), "the key must not leak to other endpoints")
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))

		body := requestBody(t, request)
		assert.Equal(t, "glm-5.2", body["model"])
		assert.Equal(t, []any{
			map[string]any{"role": "user", "content": "Say hello"},
		}, body["messages"])

		require.Len(t, state.Payloads, 1)
		assert.Equal(t, ChatCompletionPayloadType, state.Type)
		assert.Equal(t, core.DefaultOutputChannel.Name, state.Channel)

		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, "chatcmpl-1", payload.ID)
		assert.Equal(t, "glm-5.2", payload.Model)
		assert.Equal(t, "Hello, world!", payload.Text)
		assert.Equal(t, "stop", payload.FinishReason)
		require.NotNil(t, payload.Usage)
		assert.Equal(t, 23, payload.Usage.TotalTokens)
		require.NotNil(t, payload.Response)
	})

	t.Run("system prompt is sent ahead of the prompt", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		err := c.Execute(execContext(map[string]any{
			"model":        "glm-5.2",
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

	t.Run("routes Messages models to the Messages endpoint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, `{"id":"msg-1","model":"qwen3.7-max","stop_reason":"end_turn","content":[{"type":"text","text":"Hello"}],"usage":{"input_tokens":4,"output_tokens":2}}`)}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{"model": "qwen3.7-max", "prompt": "Say hello"}, httpContext, state))

		require.NoError(t, err)
		request := httpContext.Requests[0]
		assert.Contains(t, request.URL.String(), "/messages")
		assert.Equal(t, "oc-test-key", request.Header.Get("x-api-key"), "the gateway reads the key from x-api-key here")
		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, "Hello", payload.Text)
		assert.Equal(t, 6, payload.Usage.TotalTokens)
	})

	t.Run("routes Responses models to the Responses endpoint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, `{"id":"resp-1","model":"grok-4.5","output_text":"Hello","usage":{"input_tokens":3,"output_tokens":1,"total_tokens":4}}`)}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{"model": "grok-4.5", "prompt": "Say hello"}, httpContext, state))

		require.NoError(t, err)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/responses")
		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, "Hello", payload.Text)
		assert.Equal(t, 4, payload.Usage.TotalTokens)
	})

	// Node configuration arrives as JSON, so the numeric fields reach Execute as
	// float64 or json.Number rather than as Go ints.
	t.Run("numeric fields survive the JSON round trip", func(t *testing.T) {
		for name, tokens := range map[string]any{
			"float64":     float64(512),
			"json.Number": json.Number("512"),
			"int":         512,
		} {
			t.Run(name, func(t *testing.T) {
				httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

				err := c.Execute(execContext(map[string]any{
					"model":       "glm-5.2",
					"prompt":      "Say hello",
					"maxTokens":   tokens,
					"temperature": float64(0.7),
				}, httpContext, &contexts.ExecutionStateContext{}))

				require.NoError(t, err)
				body := requestBody(t, httpContext.Requests[0])
				assert.Equal(t, float64(512), body["max_tokens"])
				assert.Equal(t, float64(0.7), body["temperature"])
			})
		}
	})

	t.Run("numeric fields are omitted when unset", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		err := c.Execute(execContext(map[string]any{
			"model":  "glm-5.2",
			"prompt": "Say hello",
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])
		assert.NotContains(t, body, "max_tokens")
		assert.NotContains(t, body, "temperature")
	})

	t.Run("null content becomes an empty text field", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"chatcmpl-2","model":"m","choices":[
				{"index":0,"finish_reason":"length","message":{"role":"assistant","content":null}}
			]}`),
		}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{"model": "glm-5.2", "prompt": "hi"}, httpContext, state))

		require.NoError(t, err)
		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Empty(t, payload.Text)
		assert.Equal(t, "length", payload.FinishReason)
	})

	t.Run("pdfs are inlined as base64 file parts", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}
		ctx := execContext(map[string]any{
			"model":  "glm-5.2",
			"prompt": "Summarize",
			"files":  []any{"doc.pdf"},
		}, httpContext, &contexts.ExecutionStateContext{})
		ctx.Files = &fakeFiles{data: map[string][]byte{"doc.pdf": []byte("%PDF-1.4")}}

		require.NoError(t, c.Execute(ctx))

		// No upload call: OpenCode Go has no Files API.
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
			"model":  "glm-5.2",
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

	// A file part would need document parsing on the provider side; text needs
	// none and costs prompt tokens only.
	t.Run("text files become prompt text rather than file parts", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}
		ctx := execContext(map[string]any{
			"model":  "glm-5.2",
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

	t.Run("structured output sends a strict json schema and parses the reply", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"chatcmpl-s","model":"m","choices":[{"index":0,"finish_reason":"stop",
				"message":{"role":"assistant","content":"{\"city\":\"Nairobi\"}"}}]}`),
		}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{
			"model":        "glm-5.2",
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

		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, map[string]any{"city": "Nairobi"}, payload.Parsed)
	})

	t.Run("no response_format is sent without a schema", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}

		err := c.Execute(execContext(map[string]any{
			"model":  "glm-5.2",
			"prompt": "Say hello",
		}, httpContext, &contexts.ExecutionStateContext{}))

		require.NoError(t, err)
		body := requestBody(t, httpContext.Requests[0])
		assert.NotContains(t, body, "response_format")
	})

	// The schema is a request rather than a guarantee: prose that is not JSON
	// leaves parsed unset instead of failing the run.
	t.Run("prose replies leave parsed unset", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, completionBody)}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(execContext(map[string]any{
			"model":        "glm-5.2",
			"prompt":       "Where?",
			"outputSchema": `{"type":"object","properties":{"city":{"type":"string"}}}`,
		}, httpContext, state))

		require.NoError(t, err)
		payload := state.Payloads[0].(map[string]any)["data"].(*ChatCompletionPayload)
		assert.Equal(t, "Hello, world!", payload.Text)
		assert.Nil(t, payload.Parsed)
	})
}

func Test__ChatCompletion__ExecuteErrors(t *testing.T) {
	c := &ChatCompletion{}

	t.Run("no choices", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"id":"chatcmpl-3","model":"m","choices":[]}`),
		}}

		err := c.Execute(execContext(map[string]any{"model": "glm-5.2", "prompt": "hi"}, httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no choices")
	})

	t.Run("unauthorized with an anthropic-style error body", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusUnauthorized, `{"type":"error","error":{"type":"AuthError","message":"Missing API key."}}`),
		}}

		err := c.Execute(execContext(map[string]any{"model": "glm-5.2", "prompt": "hi"}, httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "Missing API key.")
	})

	t.Run("an out of range temperature is rejected", func(t *testing.T) {
		err := c.Execute(execContext(map[string]any{
			"model":       "glm-5.2",
			"prompt":      "hi",
			"temperature": 2.5,
		}, &contexts.HTTPContext{}, &contexts.ExecutionStateContext{}))

		require.ErrorContains(t, err, "temperature must be between 0 and 2")
	})

	t.Run("files without file access fail", func(t *testing.T) {
		err := c.Execute(execContext(map[string]any{
			"model":  "glm-5.2",
			"prompt": "hi",
			"files":  []any{"doc.pdf"},
		}, &contexts.HTTPContext{}, &contexts.ExecutionStateContext{}))

		require.ErrorContains(t, err, "files configured but file access is not available")
	})

	t.Run("a schema that is not valid JSON fails", func(t *testing.T) {
		err := c.Execute(execContext(map[string]any{
			"model":        "glm-5.2",
			"prompt":       "hi",
			"outputSchema": `{"type":"object",`,
		}, &contexts.HTTPContext{}, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not valid JSON")
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

	t.Run("records the model in node metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Logger: logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"model":  "glm-5.2",
				"prompt": "hi",
			},
			Metadata: metadata,
		})

		require.NoError(t, err)
		assert.Equal(t, ChatCompletionNodeMetadata{Model: "glm-5.2"}, metadata.Metadata)
	})

	t.Run("model is required", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"prompt": "hi"}), "model is required")
	})

	t.Run("prompt is required", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"model": "glm-5.2"}), "prompt is required")
	})

	t.Run("accepts boundary temperatures", func(t *testing.T) {
		for _, temperature := range []float64{0, 2} {
			require.NoError(t, setup(map[string]any{
				"model":       "glm-5.2",
				"prompt":      "hi",
				"temperature": temperature,
			}))
		}
	})

	t.Run("rejects a negative temperature", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{
			"model":       "glm-5.2",
			"prompt":      "hi",
			"temperature": -0.5,
		}), "temperature must be between 0 and 2")
	})

	t.Run("rejects a temperature above 2", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{
			"model":       "glm-5.2",
			"prompt":      "hi",
			"temperature": 3,
		}), "temperature must be between 0 and 2")
	})

	t.Run("records structured output in node metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Logger: logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"model":        "glm-5.2",
				"prompt":       "hi",
				"outputSchema": `{"type":"object","properties":{"a":{"type":"string"}}}`,
			},
			Metadata: metadata,
		})

		require.NoError(t, err)
		assert.Equal(t, ChatCompletionNodeMetadata{
			Model:            "glm-5.2",
			StructuredOutput: true,
		}, metadata.Metadata)
	})

	t.Run("accepts a valid schema", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{
			"model":        "glm-5.2",
			"prompt":       "hi",
			"outputSchema": `{"type":"object","properties":{"a":{"type":"string"}}}`,
		}))
	})

	t.Run("defers validating a schema that still holds an expression", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{
			"model":        "glm-5.2",
			"prompt":       "hi",
			"outputSchema": "{{ inputs.schema }}",
		}))
	})

	t.Run("rejects a malformed schema", func(t *testing.T) {
		require.Error(t, setup(map[string]any{
			"model":        "glm-5.2",
			"prompt":       "hi",
			"outputSchema": `{"type":"array"}`,
		}))
	})

	t.Run("rejects a file that is not in the repository", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Logger: logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"model":  "glm-5.2",
				"prompt": "hi",
				"files":  []any{"missing.pdf"},
			},
			Metadata: &contexts.MetadataContext{},
			Files:    &fakeFiles{data: map[string][]byte{"doc.pdf": []byte("%PDF-1.4")}},
		})
		require.ErrorContains(t, err, "not found in app repository")
	})
}
