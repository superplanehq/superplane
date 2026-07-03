package attachments

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"testing"
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

func TestRead_ClassifiesTypes(t *testing.T) {
	files := &fakeFiles{data: map[string][]byte{
		"img.png":   []byte("\x89PNG\r\n\x1a\n"),
		"doc.pdf":   []byte("%PDF-1.4"),
		"notes.txt": []byte("hello"),
	}}
	atts, err := Read(files, []string{"img.png", "doc.pdf", "notes.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atts) != 3 {
		t.Fatalf("expected 3 attachments, got %d", len(atts))
	}
	byName := map[string]Attachment{}
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
		if got := (Attachment{Mime: detected}).UploadMIME(); got != want {
			t.Errorf("UploadMIME(%q) = %q, want %q", detected, got, want)
		}
	}
}

func TestRead_EmptyPaths(t *testing.T) {
	atts, err := Read(nil, nil)
	if err != nil || atts != nil {
		t.Errorf("empty paths should return nil,nil; got %v, %v", atts, err)
	}
}

func TestRead_NilContext(t *testing.T) {
	if _, err := Read(nil, []string{"a.png"}); err == nil {
		t.Error("expected error when file context is nil")
	}
}

func TestRead_SourceFileWithMisleadingExtensionMapping(t *testing.T) {
	// System MIME tables map some plaintext source extensions to non-text
	// types (Debian maps .ts to video/mp2t). Pin that mapping so the test is
	// deterministic across environments, and check that content sniffing
	// rescues the file as text.
	if err := mime.AddExtensionType(".ts", "video/mp2t"); err != nil {
		t.Fatalf("register extension: %v", err)
	}
	files := &fakeFiles{data: map[string][]byte{"main.ts": []byte("export const x = 1\n")}}
	atts, err := Read(files, []string{"main.ts"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atts[0].UploadMIME(); got != "text/plain" {
		t.Errorf("expected text/plain upload type, got %q (detected %q)", got, atts[0].Mime)
	}
}

func TestRead_TooManyFiles(t *testing.T) {
	data := map[string][]byte{}
	paths := make([]string, 0, MaxFiles+1)
	for i := 0; i <= MaxFiles; i++ {
		name := fmt.Sprintf("f%d.txt", i)
		data[name] = []byte("x")
		paths = append(paths, name)
	}
	if _, err := Read(&fakeFiles{data: data}, paths); err == nil {
		t.Error("expected error when exceeding the file count limit")
	}
}

func TestRead_UnsupportedType(t *testing.T) {
	files := &fakeFiles{data: map[string][]byte{"app.bin": {0x00, 0x01, 0x02, 0xff}}}
	if _, err := Read(files, []string{"app.bin"}); err == nil {
		t.Error("expected error for unsupported file type")
	}
}

func TestRead_EmptyFile(t *testing.T) {
	files := &fakeFiles{data: map[string][]byte{"README.md": {}}}
	if _, err := Read(files, []string{"README.md"}); err == nil {
		t.Error("expected error for empty file")
	}
}

func TestRead_SizeLimit(t *testing.T) {
	files := &fakeFiles{data: map[string][]byte{"big.txt": make([]byte, MaxFileSize+1)}}
	if _, err := Read(files, []string{"big.txt"}); err == nil {
		t.Error("expected error for oversized file")
	}
}
