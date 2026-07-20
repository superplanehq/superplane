package openai

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

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

func Test__CreateResponse__Attachments(t *testing.T) {
	c := &CreateResponse{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk"}}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"file-img"}`))},                                // upload image (vision)
		{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"file-pdf"}`))},                                // upload pdf (user_data)
		{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","model":"gpt","output_text":"ok"}`))}, // response
		{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))},                                               // delete img
		{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))},                                               // delete pdf
	}}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"model": "gpt",
			"input": "analyze",
			"files": []any{"img.png", "doc.pdf"},
		},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		Files:          &fakeFiles{data: map[string][]byte{"img.png": []byte("png"), "doc.pdf": []byte("%PDF-1.4")}},
	}

	if err := c.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(httpCtx.Requests) != 5 {
		t.Fatalf("expected 5 requests (2 uploads, response, 2 deletes), got %d", len(httpCtx.Requests))
	}

	if !strings.Contains(httpCtx.Requests[0].URL.String(), "/files") ||
		!strings.Contains(httpCtx.Requests[1].URL.String(), "/files") {
		t.Errorf("requests 0 and 1 should be /files uploads")
	}

	respReq := httpCtx.Requests[2]
	if !strings.Contains(respReq.URL.String(), "/responses") {
		t.Errorf("request 2 should be /responses, got %s", respReq.URL)
	}
	body, _ := io.ReadAll(respReq.Body)
	bodyStr := string(body)
	for _, want := range []string{`"input_text"`, `"input_image"`, `"file_id":"file-img"`, `"input_file"`, `"file_id":"file-pdf"`} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("response body missing %s: %s", want, bodyStr)
		}
	}

	// uploaded files cleaned up
	if httpCtx.Requests[3].Method != http.MethodDelete || httpCtx.Requests[4].Method != http.MethodDelete {
		t.Errorf("requests 3 and 4 should be DELETE cleanups")
	}
}

func Test__CreateResponse__Configuration(t *testing.T) {
	c := &CreateResponse{}
	fields := c.Configuration()

	found := map[string]string{}
	for _, f := range fields {
		found[f.Name] = f.Type
	}
	if found["outputSchema"] != "text" {
		t.Errorf("expected outputSchema field of type text, got %q", found["outputSchema"])
	}
}

func Test__CreateResponse__Setup__fieldValidation(t *testing.T) {
	c := &CreateResponse{}

	tests := []struct {
		name        string
		schema      string
		expectError bool
	}{
		{
			name:        "valid schema",
			schema:      `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			schema:      `{"type":"object",}`,
			expectError: true,
		},
		{
			name:        "root not an object",
			schema:      `["a"]`,
			expectError: true,
		},
		{
			name:        "missing properties",
			schema:      `{"type":"object"}`,
			expectError: true,
		},
		{
			name:        "unresolved expression is allowed",
			schema:      `{"type":"object","properties":{"x":{"type":"string","description":"{{ inputs.desc }}"}}}`,
			expectError: false,
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
	outputSchema := `{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`

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
				"outputSchema": outputSchema,
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

func Test__CreateResponse__Execute__codeInterpreter(t *testing.T) {
	c := &CreateResponse{}

	run := func(t *testing.T, config map[string]any, responses []string) (ResponsePayload, *contexts.HTTPContext) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		httpResponses := make([]*http.Response, 0, len(responses))
		for _, body := range responses {
			httpResponses = append(httpResponses, &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(body))})
		}
		httpCtx := &contexts.HTTPContext{Responses: httpResponses}
		ctx := core.ExecutionContext{
			Configuration:  config,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
		}
		if err := c.Execute(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		payload := execState.Payloads[0].(map[string]any)["data"].(ResponsePayload)
		return payload, httpCtx
	}

	t.Run("enabled adds tool and extracts annotation artifacts with content", func(t *testing.T) {
		payload, httpCtx := run(t,
			map[string]any{"model": "gpt-5.2", "input": "plot it", "codeInterpreter": true},
			[]string{
				`{
					"id": "resp_1",
					"model": "gpt-5.2",
					"output": [
						{"type": "code_interpreter_call", "id": "ci_1", "container_id": "cntr_1"},
						{"type": "message", "role": "assistant", "content": [
							{"type": "output_text", "text": "Here is the chart", "annotations": [
								{"type": "container_file_citation", "container_id": "cntr_1", "file_id": "cfile_1", "filename": "plot.png", "start_index": 0, "end_index": 0}
							]}
						]}
					],
					"usage": {"input_tokens": 1, "output_tokens": 1, "total_tokens": 2}
				}`,
				`{"id": "cfile_1", "object": "container.file", "container_id": "cntr_1", "path": "/mnt/data/plot.png", "bytes": 8, "source": "assistant"}`,
				"\x89PNG\r\n\x1a\n",
				// The container sweep also runs so files the model generated
				// but did not cite are included; cited files are deduplicated.
				`{"object": "list", "data": [
					{"id": "cfile_1", "object": "container.file", "container_id": "cntr_1", "path": "/mnt/data/plot.png", "bytes": 8, "source": "assistant"},
					{"id": "cfile_2", "object": "container.file", "container_id": "cntr_1", "path": "/mnt/data/report.csv", "bytes": 8, "source": "assistant"}
				]}`,
				"a,b\n1,2\n",
			},
		)

		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		if !bytes.Contains(body, []byte(`"code_interpreter"`)) || !bytes.Contains(body, []byte(`"auto"`)) {
			t.Fatalf("expected code_interpreter auto tool in request body, got %s", string(body))
		}

		if len(payload.Artifacts) != 2 {
			t.Fatalf("expected 2 artifacts, got %d", len(payload.Artifacts))
		}
		artifact := payload.Artifacts[0]
		if artifact.FileID != "cfile_1" || artifact.ContainerID != "cntr_1" || artifact.Filename != "plot.png" {
			t.Errorf("unexpected artifact: %+v", artifact)
		}
		if artifact.DownloadURL != "https://api.openai.com/v1/containers/cntr_1/files/cfile_1/content" {
			t.Errorf("unexpected download URL: %s", artifact.DownloadURL)
		}
		if artifact.Bytes != 8 || artifact.Encoding != "base64" || artifact.Content == "" {
			t.Errorf("expected inlined base64 content, got %+v", artifact)
		}

		// The uncited assistant file from the container sweep.
		uncited := payload.Artifacts[1]
		if uncited.FileID != "cfile_2" || uncited.Filename != "report.csv" {
			t.Errorf("unexpected uncited artifact: %+v", uncited)
		}
		if uncited.Encoding != "text" || uncited.Content != "a,b\n1,2\n" {
			t.Errorf("expected inlined text content for uncited artifact, got %+v", uncited)
		}
	})

	t.Run("falls back to listing container files when annotations are missing", func(t *testing.T) {
		payload, httpCtx := run(t,
			map[string]any{"model": "gpt-5.2", "input": "plot it", "codeInterpreter": true},
			[]string{
				`{
					"id": "resp_1",
					"model": "gpt-5.2",
					"output": [
						{"type": "code_interpreter_call", "id": "ci_1", "container_id": "cntr_1"},
						{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "done"}]}
					],
					"usage": {"input_tokens": 1, "output_tokens": 1, "total_tokens": 2}
				}`,
				`{"object": "list", "data": [
					{"id": "cfile_gen", "object": "container.file", "container_id": "cntr_1", "path": "/mnt/data/report.csv", "bytes": 10, "source": "assistant"},
					{"id": "cfile_in", "object": "container.file", "container_id": "cntr_1", "path": "/mnt/data/input.csv", "bytes": 5, "source": "user"}
				]}`,
				"a,b\n1,2\n",
			},
		)

		// Only the assistant-generated file becomes an artifact, with content.
		if len(payload.Artifacts) != 1 {
			t.Fatalf("expected 1 artifact, got %d", len(payload.Artifacts))
		}
		artifact := payload.Artifacts[0]
		if artifact.FileID != "cfile_gen" || artifact.Filename != "report.csv" {
			t.Errorf("unexpected artifact: %+v", artifact)
		}
		if artifact.Encoding != "text" || artifact.Content != "a,b\n1,2\n" {
			t.Errorf("expected inlined text content, got %+v", artifact)
		}
		if len(httpCtx.Requests) != 3 {
			t.Fatalf("expected fallback listing and content download, got %d requests", len(httpCtx.Requests))
		}
		if !strings.Contains(httpCtx.Requests[1].URL.Path, "/containers/cntr_1/files") {
			t.Errorf("unexpected fallback request: %s", httpCtx.Requests[1].URL.String())
		}
	})

	t.Run("disabled sends no tools and skips fallback", func(t *testing.T) {
		payload, httpCtx := run(t,
			map[string]any{"model": "gpt-5.2", "input": "hi"},
			[]string{`{
				"id": "resp_1",
				"model": "gpt-5.2",
				"output_text": "hello",
				"output": [{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "hello"}]}],
				"usage": {"input_tokens": 1, "output_tokens": 1, "total_tokens": 2}
			}`},
		)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		if bytes.Contains(body, []byte(`"tools"`)) {
			t.Errorf("expected no tools in request body, got %s", string(body))
		}
		if payload.Artifacts != nil {
			t.Errorf("expected no artifacts, got %+v", payload.Artifacts)
		}
		if len(httpCtx.Requests) != 1 {
			t.Errorf("expected no fallback request, got %d requests", len(httpCtx.Requests))
		}
	})
}
