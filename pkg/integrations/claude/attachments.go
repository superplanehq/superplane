package claude

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

// Attachment reading for the text prompt component: the user picks files from
// the canvas repository, and the backend reads the raw bytes, detects the MIME
// type, and enforces size/type limits before uploading them to the Files API.
// This mirrors the repo-file + Files-API pattern used by claude.runAgent.

const (
	// maxAttachmentSize is the per-file cap when reading an attachment from the repo.
	maxAttachmentSize = 25 * 1024 * 1024 // 25 MB
	// maxAttachmentTotal caps the combined size of all attachments on one execution.
	maxAttachmentTotal = 50 * 1024 * 1024 // 50 MB
)

// attachment is a repository file read and classified, ready to upload.
type attachment struct {
	Name string // normalized repo path
	Mime string // detected MIME type (e.g. image/png, application/pdf, text/plain)
	Data []byte // raw file bytes
}

// IsImage reports whether the attachment is an image type.
func (a attachment) IsImage() bool {
	return strings.HasPrefix(a.Mime, "image/")
}

// IsPDF reports whether the attachment is a PDF.
func (a attachment) IsPDF() bool {
	return a.Mime == "application/pdf"
}

// UploadMIME returns the MIME type to send when uploading to the Files API.
// Images and PDFs keep their detected type; every other (text) type is
// normalized to text/plain, since a document content block accepts only PDF and
// plaintext (Anthropic rejects e.g. text/markdown) and any text file is just
// plaintext to the model anyway.
func (a attachment) UploadMIME() string {
	if a.IsImage() || a.IsPDF() {
		return a.Mime
	}
	return "text/plain"
}

// applicationTextTypes lists application/* MIME types that are really plaintext
// (config and source formats) and so are valid text attachments. mime.TypeByExtension
// maps common extensions like .json/.yaml/.sql/.sh here rather than under text/*,
// and UploadMIME normalizes them all to text/plain on the wire anyway.
var applicationTextTypes = map[string]bool{
	"application/json":          true,
	"application/yaml":          true,
	"application/x-yaml":        true,
	"application/sql":           true,
	"application/toml":          true,
	"application/xml":           true,
	"application/x-sh":          true,
	"application/x-shellscript": true,
	"application/x-ruby":        true,
	"application/javascript":    true,
	"application/typescript":    true,
}

func attachmentSupported(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") ||
		mimeType == "application/pdf" ||
		strings.HasPrefix(mimeType, "text/") ||
		applicationTextTypes[mimeType]
}

// readAttachments loads the given repository file paths via the execution's file
// context, detecting MIME type and enforcing size and type limits. It returns
// nil when no paths are given. Supported types are images, PDF, and text.
func readAttachments(files core.RepositoryFilesContext, paths []string) ([]attachment, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if files == nil {
		return nil, fmt.Errorf("files configured but file access is not available")
	}

	out := make([]attachment, 0, len(paths))
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
		data, err := io.ReadAll(io.LimitReader(reader, maxAttachmentSize+1))
		reader.Close()
		if err != nil {
			return nil, fmt.Errorf("read file %q: %w", path, err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("file %q is empty and cannot be attached", path)
		}
		if len(data) > maxAttachmentSize {
			return nil, fmt.Errorf("file %q exceeds the maximum attachment size of %d bytes", path, maxAttachmentSize)
		}
		total += len(data)
		if total > maxAttachmentTotal {
			return nil, fmt.Errorf("total attachment size exceeds the maximum of %d bytes", maxAttachmentTotal)
		}

		mimeType := detectAttachmentMIME(normalized, data)
		if !attachmentSupported(mimeType) {
			return nil, fmt.Errorf("file %q has unsupported type %q (supported: images, PDF, text)", path, mimeType)
		}

		out = append(out, attachment{Name: normalized, Mime: mimeType, Data: data})
	}
	return out, nil
}

// detectAttachmentMIME prefers the file extension (accurate for text/code) and
// falls back to content sniffing.
func detectAttachmentMIME(path string, data []byte) string {
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			return stripMIMEParams(t)
		}
	}
	return stripMIMEParams(http.DetectContentType(data))
}

func stripMIMEParams(t string) string {
	if base, _, found := strings.Cut(t, ";"); found {
		return strings.TrimSpace(base)
	}
	return t
}
