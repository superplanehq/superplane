// Package llmattach reads repository files selected on an LLM component and
// prepares them for upload to a provider Files API (Anthropic, OpenAI). It
// mirrors the repo-file + Files-API pattern already used by claude.runAgent:
// the user picks files from the canvas repository, and the backend reads the
// raw bytes, detects the MIME type, and enforces size/type limits.
package llmattach

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
)

const (
	// MaxFileSize is the per-file cap when reading an attachment from the repo.
	MaxFileSize = 25 * 1024 * 1024 // 25 MB
	// MaxTotalSize caps the combined size of all attachments on one execution.
	MaxTotalSize = 50 * 1024 * 1024 // 50 MB
)

// Attachment is a repository file read and classified, ready to upload.
type Attachment struct {
	Name string // normalized repo path
	Mime string // detected MIME type (e.g. image/png, application/pdf, text/plain)
	Data []byte // raw file bytes
}

// IsImage reports whether the attachment is an image type.
func (a Attachment) IsImage() bool {
	return strings.HasPrefix(a.Mime, "image/")
}

// IsPDF reports whether the attachment is a PDF.
func (a Attachment) IsPDF() bool {
	return a.Mime == "application/pdf"
}

func supported(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") ||
		mimeType == "application/pdf" ||
		strings.HasPrefix(mimeType, "text/")
}

// Read loads the given repository file paths via the execution's file context,
// detecting MIME type and enforcing size and type limits. It returns nil when
// no paths are given. Supported types are images, PDF, and text.
func Read(files core.RepositoryFilesContext, paths []string) ([]Attachment, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if files == nil {
		return nil, fmt.Errorf("files configured but file access is not available")
	}

	out := make([]Attachment, 0, len(paths))
	total := 0
	for _, path := range paths {
		normalized, err := gitprovider.ValidateUserPath(path)
		if err != nil {
			return nil, fmt.Errorf("invalid file path %q: %w", path, err)
		}

		reader, err := files.Read(normalized)
		if err != nil {
			return nil, fmt.Errorf("read file %q: %w", path, err)
		}
		data, err := io.ReadAll(io.LimitReader(reader, MaxFileSize+1))
		reader.Close()
		if err != nil {
			return nil, fmt.Errorf("read file %q: %w", path, err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("file %q is empty and cannot be attached", path)
		}
		if len(data) > MaxFileSize {
			return nil, fmt.Errorf("file %q exceeds the maximum attachment size of %d bytes", path, MaxFileSize)
		}
		total += len(data)
		if total > MaxTotalSize {
			return nil, fmt.Errorf("total attachment size exceeds the maximum of %d bytes", MaxTotalSize)
		}

		mimeType := detectMIME(normalized, data)
		if !supported(mimeType) {
			return nil, fmt.Errorf("file %q has unsupported type %q (supported: images, PDF, text)", path, mimeType)
		}

		out = append(out, Attachment{Name: normalized, Mime: mimeType, Data: data})
	}
	return out, nil
}

// detectMIME prefers the file extension (accurate for text/code) and falls back
// to content sniffing.
func detectMIME(path string, data []byte) string {
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			return stripParams(t)
		}
	}
	return stripParams(http.DetectContentType(data))
}

func stripParams(t string) string {
	if i := strings.IndexByte(t, ';'); i >= 0 {
		return strings.TrimSpace(t[:i])
	}
	return t
}
