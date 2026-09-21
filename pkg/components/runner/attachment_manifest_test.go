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
}

func TestApplyAttachmentInstructionsPrependsFragment(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AttachmentAgentInstructions, ApplyAttachmentInstructions(""))
	assert.True(t, strings.HasPrefix(ApplyAttachmentInstructions("do the work"), AttachmentAgentInstructions))
}

func TestAttachmentSetupSkipsProcessWhenNoVideo(t *testing.T) {
	t.Parallel()

	files, commands := AttachmentSetup([]TaskAttachment{{
		URL:      "https://files.example/shot.png?sp_file=1",
		Filename: "shot.png",
	}})
	require.NotEmpty(t, files)
	require.Len(t, commands, 1)
	assert.Equal(t, "Fetch task attachments", commands[0].Name)
	assert.Equal(t, AttachmentManifestPath, files[len(files)-1].Path)
}
