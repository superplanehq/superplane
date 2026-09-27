package runner

import (
	"fmt"
	"strings"
)

const (
	VideoMaxDurationSeconds     = 900
	VideoMaxFrames              = 24
	VideoMaxFrameWidth          = 1280
	VideoMaxFrameHeight         = 1280
	VideoMaxSourcePixels        = 16_777_216
	VideoProcessTimeoutSeconds  = 120
	VideoDiskBudgetBytes        = 2 << 30
	AttachmentManifestVersion   = 1
	AttachmentManifestPath      = "attachments/manifest.json"
	AttachmentIndexPath         = "attachments/INDEX.md"
	AttachmentFetchScriptPath   = "fetch_task_attachments.sh"
	AttachmentProcessScriptPath = "process_video_attachments.sh"
)

// AttachmentAgentInstructions tell every agent CLI to read processed
// artifacts instead of original video or audio bytes.
const AttachmentAgentInstructions = `If $SUPERPLANE_TASK_DIR/attachments/INDEX.md exists, read that file first. Use the listed frames and transcript for any video. Use the listed transcript for any audio. Do not ingest original video or audio bytes.`

type AttachmentPolicy struct {
	MaxDurationSeconds    int   `json:"max_duration_seconds"`
	MaxFrames             int   `json:"max_frames"`
	MaxFrameWidth         int   `json:"max_frame_width"`
	MaxFrameHeight        int   `json:"max_frame_height"`
	MaxSourcePixels       int   `json:"max_source_pixels"`
	ProcessTimeoutSeconds int   `json:"process_timeout_seconds"`
	DiskBudgetBytes       int64 `json:"disk_budget_bytes"`
}

func DefaultAttachmentPolicy() AttachmentPolicy {
	return AttachmentPolicy{
		MaxDurationSeconds:    VideoMaxDurationSeconds,
		MaxFrames:             VideoMaxFrames,
		MaxFrameWidth:         VideoMaxFrameWidth,
		MaxFrameHeight:        VideoMaxFrameHeight,
		MaxSourcePixels:       VideoMaxSourcePixels,
		ProcessTimeoutSeconds: VideoProcessTimeoutSeconds,
		DiskBudgetBytes:       VideoDiskBudgetBytes,
	}
}

func ApplyAttachmentInstructions(prompt string, attachments []TaskAttachment) string {
	if len(attachments) == 0 {
		return prompt
	}
	if prompt == "" {
		return AttachmentAgentInstructions
	}
	return AttachmentAgentInstructions + "\n\n" + prompt
}

func FormatAgentPrompt(prompt, usage string, attachments []TaskAttachment, inspectImages bool) string {
	next := ApplyIntegrationUsage(prompt, usage)
	if inspectImages {
		next = RewritePromptLocalAttachmentPaths(next, attachments)
	}
	return ApplyAttachmentInstructions(next, attachments)
}

func LocalAttachmentPath(dest string) string {
	return "$SUPERPLANE_TASK_DIR/attachments/" + dest
}

func RewritePromptLocalAttachmentPaths(prompt string, attachments []TaskAttachment) string {
	if prompt == "" || len(attachments) == 0 {
		return prompt
	}
	next := prompt
	var imagePaths []string
	for _, file := range NewAttachmentManifest(attachments).Files {
		if file.Kind != "image" || file.URL == "" || file.Dest == "" {
			continue
		}
		local := LocalAttachmentPath(file.Dest)
		next = strings.ReplaceAll(next, file.URL, local)
		imagePaths = append(imagePaths, local)
	}
	if len(imagePaths) == 0 {
		return next
	}
	return next + "\n\n" + FormatInspectAttachmentInstruction(imagePaths)
}

func FormatInspectAttachmentInstruction(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return fmt.Sprintf("Call inspect_attachment on %s and review the returned image.", strings.Join(paths, ", "))
}
