package runner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentManifestJSONIncludesPolicyAndDest(t *testing.T) {
	t.Parallel()

	body := AttachmentManifestJSON([]TaskAttachment{{
		ID:          "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		URL:         "https://files.example/clip.mp4?sp_file=1",
		Filename:    "clip.mp4",
		ContentType: "video/mp4",
		SizeBytes:   12,
		Checksum:    "abc",
	}})
	assert.Contains(t, body, `"version": 1`)
	assert.Contains(t, body, `"max_duration_seconds": 900`)
	assert.Contains(t, body, `"max_frames": 24`)
	assert.Contains(t, body, `"dest": "01-clip.mp4"`)
	assert.Contains(t, body, `"kind": "video"`)
	assert.Contains(t, body, `"checksum": "abc"`)
}

func TestHasVideoAttachmentUsesTypeAndName(t *testing.T) {
	t.Parallel()

	assert.False(t, HasVideoAttachment([]TaskAttachment{{Filename: "shot.png", ContentType: "image/png"}}))
	assert.True(t, HasVideoAttachment([]TaskAttachment{{Filename: "clip.mov", ContentType: "video/quicktime"}}))
	assert.True(t, HasVideoAttachment([]TaskAttachment{{Filename: "walkthrough.mkv"}}))
	assert.True(t, HasVideoAttachment([]TaskAttachment{{Filename: "note.mp3", ContentType: "audio/mpeg"}}))
	assert.True(t, HasVideoAttachment([]TaskAttachment{{Filename: "voice.m4a"}}))
	assert.Equal(t, "audio", attachmentKind(TaskAttachment{Filename: "clip.ogg", ContentType: "audio/ogg"}))
	assert.Equal(t, "audio", attachmentKind(TaskAttachment{Filename: "clip.ogg"}))
	assert.Equal(t, "video", attachmentKind(TaskAttachment{Filename: "clip.ogv"}))
	assert.Equal(t, "audio", attachmentKind(TaskAttachment{Filename: "clip.webm", ContentType: "audio/webm"}))
	assert.Equal(t, "video", attachmentKind(TaskAttachment{Filename: "clip.webm"}))
}

func TestApplyAttachmentInstructionsPrependsFragment(t *testing.T) {
	t.Parallel()

	files := []TaskAttachment{{Filename: "clip.mp4", ContentType: "video/mp4"}}
	assert.Equal(t, "do the work", ApplyAttachmentInstructions("do the work", nil))
	assert.Equal(t, AttachmentAgentInstructions, ApplyAttachmentInstructions("", files))
	assert.True(t, strings.HasPrefix(ApplyAttachmentInstructions("do the work", files), AttachmentAgentInstructions))
	assert.NotContains(t, AttachmentAgentInstructions, "inspect_attachment")
}

func TestRewritePromptLocalAttachmentPathsNamesInspectAttachment(t *testing.T) {
	t.Parallel()

	signed := "https://app.example/api/v1/public/files/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa?expires=1&sig=abc&sp_file=1"
	prompt := RewritePromptLocalAttachmentPaths("See ![bug]("+signed+")", []TaskAttachment{{
		URL:         signed,
		Filename:    "bug.png",
		ContentType: "image/png",
	}})
	assert.Contains(t, prompt, "$SUPERPLANE_TASK_DIR/attachments/01-bug.png")
	assert.Contains(t, prompt, "inspect_attachment")
	assert.NotContains(t, prompt, signed)
}

func TestFormatAgentPromptKeepsLineImageURLs(t *testing.T) {
	t.Parallel()

	signed := "https://app.example/api/v1/public/files/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa?expires=1&sig=abc&sp_file=1"
	files := []TaskAttachment{{
		URL:         signed,
		Filename:    "bug.png",
		ContentType: "image/png",
	}}
	line := FormatAgentPrompt("See ![bug]("+signed+")", "", files, false)
	assert.Contains(t, line, signed)
	assert.Contains(t, line, AttachmentAgentInstructions)
	assert.NotContains(t, line, "inspect_attachment")

	planning := FormatAgentPrompt("See ![bug]("+signed+")", "", files, true)
	assert.Contains(t, planning, "$SUPERPLANE_TASK_DIR/attachments/01-bug.png")
	assert.Contains(t, planning, "inspect_attachment")
	assert.NotContains(t, planning, signed)
}

func TestAttachmentSetupProcessesImages(t *testing.T) {
	t.Parallel()

	files, commands := AttachmentSetup([]TaskAttachment{{
		URL:         "https://files.example/shot.png?sp_file=1",
		Filename:    "shot.png",
		ContentType: "image/png",
	}})
	require.NotEmpty(t, files)
	require.Len(t, commands, 2)
	assert.Equal(t, "Fetch task attachments", commands[0].Name)
	assert.Equal(t, "Process task attachments", commands[1].Name)
	assert.Equal(t, AttachmentManifestPath, files[len(files)-1].Path)
	assert.Contains(t, files[len(files)-1].Content, `"kind": "image"`)
}
