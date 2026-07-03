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
