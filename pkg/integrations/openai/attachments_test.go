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

func TestReadAttachments_ClassifiesTypes(t *testing.T) {
	files := &fakeFiles{data: map[string][]byte{
		"img.png":   []byte("\x89PNG\r\n\x1a\n"),
		"doc.pdf":   []byte("%PDF-1.4"),
		"notes.txt": []byte("hello"),
	}}
	atts, err := readAttachments(files, []string{"img.png", "doc.pdf", "notes.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atts) != 3 {
		t.Fatalf("expected 3 attachments, got %d", len(atts))
	}
	byName := map[string]attachment{}
	for _, a := range atts {
		byName[a.Name] = a
	}
	if !byName["img.png"].IsImage() {
		t.Errorf("png should be image, got %q", byName["img.png"].Mime)
	}
	if !byName["doc.pdf"].IsPDF() {
		t.Errorf("pdf should be PDF, got %q", byName["doc.pdf"].Mime)
	}
	if byName["notes.txt"].IsImage() || byName["notes.txt"].IsPDF() {
		t.Errorf("txt misclassified, got %q", byName["notes.txt"].Mime)
	}
}

func TestUploadMIME(t *testing.T) {
	cases := map[string]string{
		"image/png":       "image/png",
		"image/jpeg":      "image/jpeg",
		"application/pdf": "application/pdf",
		"text/markdown":   "text/plain",
		"text/csv":        "text/plain",
		"text/plain":      "text/plain",
	}
	for detected, want := range cases {
		if got := (attachment{Mime: detected}).UploadMIME(); got != want {
			t.Errorf("UploadMIME(%q) = %q, want %q", detected, got, want)
		}
	}
}

func TestReadAttachments_EmptyPaths(t *testing.T) {
	atts, err := readAttachments(nil, nil)
	if err != nil || atts != nil {
		t.Errorf("empty paths should return nil,nil; got %v, %v", atts, err)
	}
}

func TestReadAttachments_NilContext(t *testing.T) {
	if _, err := readAttachments(nil, []string{"a.png"}); err == nil {
		t.Error("expected error when file context is nil")
	}
}

func TestReadAttachments_UnsupportedType(t *testing.T) {
	files := &fakeFiles{data: map[string][]byte{"app.bin": {0x00, 0x01, 0x02, 0xff}}}
	if _, err := readAttachments(files, []string{"app.bin"}); err == nil {
		t.Error("expected error for unsupported file type")
	}
}

func TestReadAttachments_EmptyFile(t *testing.T) {
	files := &fakeFiles{data: map[string][]byte{"README.md": {}}}
	if _, err := readAttachments(files, []string{"README.md"}); err == nil {
		t.Error("expected error for empty file")
	}
}

func TestReadAttachments_SizeLimit(t *testing.T) {
	files := &fakeFiles{data: map[string][]byte{"big.txt": make([]byte, maxAttachmentSize+1)}}
	if _, err := readAttachments(files, []string{"big.txt"}); err == nil {
		t.Error("expected error for oversized file")
	}
}
