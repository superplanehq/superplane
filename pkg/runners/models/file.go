package models

// TaskFile is a file the runner materializes before task execution.
// Path is relative to SUPERPLANE_TASK_DIR (no absolute paths or "..").
type TaskFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	// Mode is an optional unix file mode in octal (e.g. "0644", "0755").
	// When empty, the runner uses 0644.
	Mode string `json:"mode,omitempty"`
}
