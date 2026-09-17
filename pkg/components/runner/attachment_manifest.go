package runner

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
)

type AttachmentManifest struct {
	Version int                      `json:"version"`
	Policy  AttachmentPolicy         `json:"policy"`
	Files   []AttachmentManifestFile `json:"files"`
}

type AttachmentManifestFile struct {
	ID          string `json:"id,omitempty"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	Checksum    string `json:"checksum,omitempty"`
	URL         string `json:"url"`
	Dest        string `json:"dest"`
	Kind        string `json:"kind,omitempty"`
	Status      string `json:"status,omitempty"`
}

func NewAttachmentManifest(attachments []TaskAttachment) AttachmentManifest {
	used := map[string]int{}
	files := make([]AttachmentManifestFile, 0, len(attachments))
	for i, attachment := range attachments {
		dest := attachmentDestName(attachment, i+1, used)
		files = append(files, AttachmentManifestFile{
			ID:          attachment.ID,
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			SizeBytes:   attachment.SizeBytes,
			Checksum:    attachment.Checksum,
			URL:         attachment.URL,
			Dest:        dest,
			Kind:        attachmentKind(attachment),
			Status:      "pending",
		})
	}
	return AttachmentManifest{
		Version: AttachmentManifestVersion,
		Policy:  DefaultAttachmentPolicy(),
		Files:   files,
	}
}

func AttachmentManifestJSON(attachments []TaskAttachment) string {
	body, err := json.MarshalIndent(NewAttachmentManifest(attachments), "", "  ")
	if err != nil {
		panic(err)
	}
	return string(body) + "\n"
}

func TaskAttachmentsFromDispatch(files []storedfiles.DispatchFile) []TaskAttachment {
	attachments := make([]TaskAttachment, 0, len(files))
	seen := map[string]struct{}{}
	for _, file := range files {
		id := file.ID.String()
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		attachments = append(attachments, TaskAttachment{
			ID:          id,
			URL:         file.URL,
			Filename:    file.Filename,
			ContentType: file.ContentType,
			SizeBytes:   file.SizeBytes,
			Checksum:    file.Checksum,
		})
	}
	return attachments
}

func ResolveTaskAttachments(input AgentBrokerTaskInput) []TaskAttachment {
	if len(input.Attachments) > 0 {
		return uniqueTaskAttachments(input.Attachments)
	}
	return CollectTaskAttachmentsFromSteps(AgentStepsForDispatch(input.Steps, input.DispatchedSteps))
}

func HasVideoAttachment(attachments []TaskAttachment) bool {
	for _, attachment := range attachments {
		if attachmentKind(attachment) == "video" {
			return true
		}
	}
	return false
}

func uniqueTaskAttachments(attachments []TaskAttachment) []TaskAttachment {
	seen := map[string]struct{}{}
	out := make([]TaskAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		key := attachment.ID
		if key == "" {
			key = attachment.URL
		}
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, attachment)
	}
	return out
}

func attachmentKind(attachment TaskAttachment) string {
	if models.IsInlineVideoContentType(attachment.ContentType) || looksLikeVideoName(attachment.Filename) {
		return "video"
	}
	if models.IsInlineImageContentType(attachment.ContentType) {
		return "image"
	}
	return "file"
}

func looksLikeVideoName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp4", ".webm", ".mov", ".ogv", ".ogg", ".m4v", ".mkv":
		return true
	default:
		return false
	}
}

func attachmentDestName(attachment TaskAttachment, index int, used map[string]int) string {
	base := sanitizeAttachmentName(attachment.Filename)
	if base == "" {
		base = sanitizeAttachmentName(path.Base(attachment.URL))
	}
	if base == "" || base == "." {
		base = "file"
	}
	name := fmt.Sprintf("%02d-%s", index, base)
	if count := used[name]; count > 0 {
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		name = fmt.Sprintf("%s-%d%s", stem, count+1, ext)
	}
	used[name]++
	used[fmt.Sprintf("%02d-%s", index, base)]++
	return name
}

func sanitizeAttachmentName(raw string) string {
	var cleaned strings.Builder
	for _, r := range filepath.Base(raw) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			cleaned.WriteRune(r)
		}
	}
	name := cleaned.String()
	if name == "" || name == "." {
		return "file"
	}
	return name
}
