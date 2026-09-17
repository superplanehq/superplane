package runner

const (
	VideoMaxDurationSeconds     = 900
	VideoMaxFrames              = 24
	VideoMaxFrameWidth          = 1280
	VideoProcessTimeoutSeconds  = 120
	VideoDiskBudgetBytes        = 2 << 30
	AttachmentManifestVersion   = 1
	AttachmentManifestPath      = "attachments/manifest.json"
	AttachmentIndexPath         = "attachments/INDEX.md"
	AttachmentFetchScriptPath   = "fetch_task_attachments.sh"
	AttachmentProcessScriptPath = "process_video_attachments.sh"
)

// AttachmentAgentInstructions tell every agent CLI to read processed
// artifacts instead of original video bytes.
const AttachmentAgentInstructions = `If $SUPERPLANE_TASK_DIR/attachments/INDEX.md exists, read that file first. Use the listed frames and transcript for any video. Do not ingest original video bytes.`

type AttachmentPolicy struct {
	MaxDurationSeconds    int   `json:"max_duration_seconds"`
	MaxFrames             int   `json:"max_frames"`
	MaxFrameWidth         int   `json:"max_frame_width"`
	ProcessTimeoutSeconds int   `json:"process_timeout_seconds"`
	DiskBudgetBytes       int64 `json:"disk_budget_bytes"`
}

func DefaultAttachmentPolicy() AttachmentPolicy {
	return AttachmentPolicy{
		MaxDurationSeconds:    VideoMaxDurationSeconds,
		MaxFrames:             VideoMaxFrames,
		MaxFrameWidth:         VideoMaxFrameWidth,
		ProcessTimeoutSeconds: VideoProcessTimeoutSeconds,
		DiskBudgetBytes:       VideoDiskBudgetBytes,
	}
}

func ApplyAttachmentInstructions(prompt string) string {
	if prompt == "" {
		return AttachmentAgentInstructions
	}
	return AttachmentAgentInstructions + "\n\n" + prompt
}
