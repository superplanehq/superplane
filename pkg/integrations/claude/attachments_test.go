package claude

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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

// uploadPartContentType parses the multipart upload body and returns the
// Content-Type header of the "file" part.
func uploadPartContentType(t *testing.T, req *http.Request) string {
	t.Helper()
	_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse upload Content-Type: %v", err)
	}
	mr := multipart.NewReader(req.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read multipart part: %v", err)
		}
		if part.FormName() == "file" {
			return part.Header.Get("Content-Type")
		}
	}
	return ""
}

func TestTextPrompt_Attachments(t *testing.T) {
	c := &TextPrompt{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"id":"file_abc"}`))},                                                                                                                        // upload
		{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"id":"msg_1","model":"m","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))}, // message
		{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{}`))},                                                                                                                                       // delete (cleanup)
	}}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":  "m",
			"prompt": "describe",
			"files":  []any{"img.png"},
		},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		Files:          &fakeFiles{data: map[string][]byte{"img.png": []byte("pngdata")}},
	}

	if err := c.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(httpCtx.Requests) != 3 {
		t.Fatalf("expected 3 requests (upload, message, delete), got %d", len(httpCtx.Requests))
	}

	// 1. upload to the Files API with the files beta header
	up := httpCtx.Requests[0]
	if !strings.Contains(up.URL.String(), "/files") {
		t.Errorf("request 0 should be a /files upload, got %s", up.URL)
	}
	if up.Header.Get("anthropic-beta") != "files-api-2025-04-14" {
		t.Errorf("upload missing files beta header")
	}
	// The multipart file part must carry the detected MIME type, otherwise the
	// provider stores the file as application/octet-stream and rejects it.
	if ct := uploadPartContentType(t, up); ct != "image/png" {
		t.Errorf("upload file part Content-Type = %q, want image/png", ct)
	}

	// 2. message references the uploaded file via an image block + file_id
	msg := httpCtx.Requests[1]
	if !strings.Contains(msg.URL.String(), "/messages") {
		t.Errorf("request 1 should be /messages, got %s", msg.URL)
	}
	if msg.Header.Get("anthropic-beta") != "files-api-2025-04-14" {
		t.Errorf("message missing files beta header")
	}
	body, _ := io.ReadAll(msg.Body)
	bodyStr := string(body)
	for _, want := range []string{`"type":"image"`, `"type":"file"`, `"file_id":"file_abc"`} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("message body missing %s: %s", want, bodyStr)
		}
	}

	// 3. uploaded file is cleaned up
	del := httpCtx.Requests[2]
	if del.Method != http.MethodDelete || !strings.Contains(del.URL.String(), "/files/file_abc") {
		t.Errorf("request 2 should be DELETE /files/file_abc, got %s %s", del.Method, del.URL)
	}
}
