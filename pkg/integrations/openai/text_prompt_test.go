package openai

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateResponse__Configuration(t *testing.T) {
	c := &CreateResponse{}
	fields := c.Configuration()

	found := map[string]string{}
	for _, f := range fields {
		found[f.Name] = f.Type
	}
	if found["outputSchema"] != "object" {
		t.Errorf("expected outputSchema field of type object, got %q", found["outputSchema"])
	}
}

func Test__CreateResponse__Setup__schemaValidation(t *testing.T) {
	c := &CreateResponse{}

	tests := []struct {
		name        string
		schema      any
		expectError bool
	}{
		{
			name: "valid strict schema",
			schema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"name": map[string]any{"type": "string"}},
				"required":             []any{"name"},
				"additionalProperties": false,
			},
			expectError: false,
		},
		{
			name:        "non-object root rejected",
			schema:      map[string]any{"type": "string"},
			expectError: true,
		},
		{
			name: "missing additionalProperties rejected",
			schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"name": map[string]any{"type": "string"}},
				"required":   []any{"name"},
			},
			expectError: true,
		},
		{
			name: "optional property not in required rejected",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"age":  map[string]any{"type": "number"},
				},
				"required":             []any{"name"},
				"additionalProperties": false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := core.SetupContext{
				Configuration: map[string]any{
					"model":        "gpt-5.2",
					"input":        "hi",
					"outputSchema": tt.schema,
				},
			}
			err := c.Setup(ctx)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test__CreateResponse__Execute__structuredOutput(t *testing.T) {
	c := &CreateResponse{}
	schema := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"city": map[string]any{"type": "string"}},
		"required":             []any{"city"},
		"additionalProperties": false,
	}

	run := func(t *testing.T, responseBody string) (ResponsePayload, []byte) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-test"},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(responseBody))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"model":        "gpt-5.2",
				"input":        "where?",
				"outputSchema": schema,
			},
			ExecutionState: execState,
			HTTP:           httpCtx,
			Integration:    integrationCtx,
		}
		if err := c.Execute(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wrapped, ok := execState.Payloads[0].(map[string]any)
		if !ok {
			t.Fatal("emitted payload wrapper is not map[string]any")
		}
		payload, ok := wrapped["data"].(ResponsePayload)
		if !ok {
			t.Fatal("emitted payload data is not ResponsePayload")
		}
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		return payload, body
	}

	t.Run("parses output and sends text.format", func(t *testing.T) {
		payload, body := run(t, `{
			"id": "resp_1",
			"model": "gpt-5.2",
			"output_text": "{\"city\":\"Paris\"}",
			"output": [
				{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "{\"city\":\"Paris\"}"}]}
			],
			"usage": {"input_tokens": 1, "output_tokens": 1, "total_tokens": 2}
		}`)

		if !bytes.Contains(body, []byte(`"format"`)) || !bytes.Contains(body, []byte(`"json_schema"`)) || !bytes.Contains(body, []byte(`"strict":true`)) {
			t.Fatalf("expected text.format json_schema strict in request body, got %s", string(body))
		}
		parsed, ok := payload.Parsed.(map[string]any)
		if !ok {
			t.Fatalf("expected Parsed object, got %T", payload.Parsed)
		}
		if parsed["city"] != "Paris" {
			t.Errorf("expected city=Paris, got %v", parsed["city"])
		}
	})

	t.Run("refusal surfaces as text and leaves Parsed nil", func(t *testing.T) {
		payload, _ := run(t, `{
			"id": "resp_2",
			"model": "gpt-5.2",
			"output_text": "",
			"output": [
				{"type": "message", "role": "assistant", "content": [{"type": "refusal", "refusal": "I won't do that."}]}
			],
			"usage": {"input_tokens": 1, "output_tokens": 1, "total_tokens": 2}
		}`)
		if payload.Parsed != nil {
			t.Errorf("expected Parsed nil on refusal, got %v", payload.Parsed)
		}
		if payload.Text != "I won't do that." {
			t.Errorf("expected refusal surfaced as text, got %q", payload.Text)
		}
	})
}

func Test__CreateResponse__NodeMetadata(t *testing.T) {
	c := &CreateResponse{}
	md := &contexts.MetadataContext{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"model": "gpt-5.2",
			"input": "hi",
		},
		Metadata: md,
	}

	if err := c.Setup(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta, ok := md.Metadata.(ResponseNodeMetadata)
	if !ok {
		t.Fatalf("expected ResponseNodeMetadata, got %T", md.Metadata)
	}
	if meta.Model != "gpt-5.2" {
		t.Errorf("expected model gpt-5.2, got %q", meta.Model)
	}
	if meta.StructuredOutput {
		t.Error("expected structuredOutput false (no schema)")
	}
}
