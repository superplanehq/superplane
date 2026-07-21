package api

import (
	"path"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
)

const (
	// MaxTaskFiles is the maximum number of files accepted on create-task.
	MaxTaskFiles = 64
	// MaxTaskFileBytes is the maximum size of a single file's content.
	MaxTaskFileBytes = 1 << 20 // 1 MiB
	// MaxTaskFilesTotalBytes is the maximum combined content size.
	MaxTaskFilesTotalBytes = 4 << 20 // 4 MiB
	// DefaultTaskFileMode is used when TaskFile.Mode is empty.
	DefaultTaskFileMode = 0o644
)

// TaskFile is one file attached to a create-task / claim payload.
type TaskFile = models.TaskFile

// ValidateFiles returns a non-empty error message when task files are invalid.
func ValidateFiles(files []TaskFile) string {
	if len(files) == 0 {
		return ""
	}
	if len(files) > MaxTaskFiles {
		return "files exceeds maximum of " + strconv.Itoa(MaxTaskFiles)
	}

	seen := make(map[string]struct{}, len(files))
	total := 0
	for i, file := range files {
		rel, msg := normalizeTaskFilePath(file.Path)
		if msg != "" {
			return "files[" + strconv.Itoa(i) + "]: " + msg
		}
		if _, ok := seen[rel]; ok {
			return "files[" + strconv.Itoa(i) + "]: duplicate path"
		}
		seen[rel] = struct{}{}

		if strings.ContainsRune(file.Content, '\x00') {
			return "files[" + strconv.Itoa(i) + "]: content cannot contain NUL bytes"
		}
		if len(file.Content) > MaxTaskFileBytes {
			return "files[" + strconv.Itoa(i) + "]: content exceeds maximum of " + strconv.Itoa(MaxTaskFileBytes) + " bytes"
		}
		total += len(file.Content)
		if total > MaxTaskFilesTotalBytes {
			return "files total content exceeds maximum of " + strconv.Itoa(MaxTaskFilesTotalBytes) + " bytes"
		}
		if _, msg := ParseTaskFileMode(file.Mode); msg != "" {
			return "files[" + strconv.Itoa(i) + "]: " + msg
		}
	}
	return ""
}

// NormalizeFiles returns a copy with cleaned relative paths; drops empty path entries.
func NormalizeFiles(files []TaskFile) []TaskFile {
	if len(files) == 0 {
		return nil
	}
	out := make([]TaskFile, 0, len(files))
	for _, file := range files {
		rel, msg := normalizeTaskFilePath(file.Path)
		if msg != "" {
			continue
		}
		out = append(out, TaskFile{
			Path:    rel,
			Content: file.Content,
			Mode:    strings.TrimSpace(file.Mode),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// CloneFiles returns a detached copy of task files.
func CloneFiles(files []TaskFile) []TaskFile {
	if len(files) == 0 {
		return nil
	}
	out := make([]TaskFile, len(files))
	copy(out, files)
	return out
}

func normalizeTaskFilePath(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "path is required"
	}
	if strings.ContainsRune(raw, '\x00') {
		return "", "path cannot contain NUL bytes"
	}
	if path.IsAbs(raw) || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, `\`) {
		return "", "path must be relative"
	}
	// Reject Windows drive paths (C:...) even though runners are typically Unix.
	if len(raw) >= 2 && raw[1] == ':' {
		return "", "path must be relative"
	}

	cleaned := path.Clean(strings.ReplaceAll(raw, `\`, "/"))
	cleaned = strings.TrimPrefix(cleaned, "./")
	if cleaned == "." || cleaned == "" {
		return "", "path must be a relative file path"
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", "path cannot contain '..'"
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." {
			return "", "path cannot contain '..'"
		}
		if part == "" {
			return "", "path is invalid"
		}
	}
	return cleaned, ""
}

// ParseTaskFileMode parses an optional octal unix file mode.
// Empty input yields DefaultTaskFileMode.
func ParseTaskFileMode(raw string) (uint32, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultTaskFileMode, ""
	}
	if strings.HasPrefix(raw, "0o") || strings.HasPrefix(raw, "0O") {
		raw = raw[2:]
	}
	v, err := strconv.ParseUint(raw, 8, 32)
	if err != nil {
		return 0, "mode must be an octal unix file mode"
	}
	if v > 0o7777 {
		return 0, "mode is out of range"
	}
	return uint32(v), ""
}
