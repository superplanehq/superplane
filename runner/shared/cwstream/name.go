// Package cwstream holds naming helpers for task log streams (e.g. Amazon CloudWatch Logs).
package cwstream

import "strings"

// TaskLogStream returns the log stream name for a task.
// Prefix is optional (e.g. "tasks"); when empty the stream is the task id alone.
func TaskLogStream(prefix, taskID string) string {
	p := strings.TrimSpace(prefix)
	t := strings.TrimSpace(taskID)
	if t == "" {
		return ""
	}
	if p == "" {
		return t
	}
	p = strings.TrimSuffix(p, "/")
	return p + "/" + t
}
