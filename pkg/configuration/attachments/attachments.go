// Package attachments reads repository files selected on an integration
// component and prepares them for upload to a provider Files API (Anthropic,
// OpenAI). It is a shared helper for integrations rather than an integration
// itself: the user picks files from the canvas repository, and the backend reads
// the raw bytes, detects the MIME type, and enforces size/type limits. This
// mirrors the repo-file + Files-API pattern used by claude.runAgent.
package attachments

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
	// The provider Files APIs accept far larger uploads (hundreds of MB), but
	// attachments are buffered fully in memory here, so the caps bound memory
	// use and request size rather than what the providers allow.
	MaxFileSize = 25 * 1024 * 1024 // 25 MB
	// MaxTotalSize caps the combined size of all attachments on one execution.
	MaxTotalSize = 50 * 1024 * 1024 // 50 MB
	// MaxFiles caps how many files one execution may attach; each attachment
	// costs a separate Files API upload call.
	MaxFiles = 20
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

// UploadMIME returns the MIME type to send when uploading to the Files API.
// Images and PDFs keep their detected type; every other (text) type is
// normalized to text/plain, since a document content block accepts only PDF and
// plaintext (Anthropic rejects e.g. text/markdown) and any text file is just
// plaintext to the model anyway.
func (a Attachment) UploadMIME() string {
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

func supported(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") ||
		mimeType == "application/pdf" ||
		strings.HasPrefix(mimeType, "text/") ||
		applicationTextTypes[mimeType]
}

// Read loads the given repository file paths via the execution's file context,
// detecting MIME type and enforcing size and type limits. It returns nil when no
// paths are given. Supported types are images, PDF, and text.
func Read(files core.RepositoryFilesContext, paths []string) ([]Attachment, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if files == nil {
		return nil, fmt.Errorf("files configured but file access is not available")
	}
	if len(paths) > MaxFiles {
		return nil, fmt.Errorf("too many files: %d configured, maximum is %d", len(paths), MaxFiles)
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
// to content sniffing. The extension mapping is trusted only when it yields a
// supported type: system MIME tables misclassify some plaintext source files
// (e.g. .ts as video/mp2t, .tsx as application/x-tiled-tsx), and sniffing the
// bytes identifies those as text.
func detectMIME(path string, data []byte) string {
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			if base := stripParams(t); supported(base) {
				return base
			}
		}
	}
	return stripParams(http.DetectContentType(data))
}

func stripParams(t string) string {
	if base, _, found := strings.Cut(t, ";"); found {
		return strings.TrimSpace(base)
	}
	return t
}
