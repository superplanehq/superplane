package claude

import "testing"

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
