package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/superplane/runner/shared/api"
)

const (
	envSuperplaneTaskDir = "SUPERPLANE_TASK_DIR"
	dockerTaskDirMount   = "/superplane-task"
)

// materializeTaskFiles writes task.Files under root and returns the absolute root path.
// When there are no files, it returns ("", nil).
func materializeTaskFiles(root string, files []api.TaskFile) (string, error) {
	normalized := api.NormalizeFiles(files)
	if len(normalized) == 0 {
		return "", nil
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("task file root is required")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	for _, file := range normalized {
		mode, err := taskFileMode(file.Mode)
		if err != nil {
			return "", fmt.Errorf("%s: %w", file.Path, err)
		}
		dest := filepath.Join(absRoot, filepath.FromSlash(file.Path))
		if !isPathInsideRoot(absRoot, dest) {
			return "", fmt.Errorf("%s: path escapes task directory", file.Path)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return "", err
		}
		if err := os.WriteFile(dest, []byte(file.Content), mode); err != nil {
			return "", err
		}
	}
	return absRoot, nil
}

func hostTaskFilesRoot(workDir, taskID string) string {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		workDir = "."
	}
	id := strings.TrimSpace(taskID)
	if id == "" {
		id = "unknown"
	}
	return filepath.Join(workDir, ".superplane", "tasks", id)
}

func withTaskDirEnv(env []string, taskDir string) []string {
	taskDir = strings.TrimSpace(taskDir)
	if taskDir == "" {
		return env
	}
	pair := envSuperplaneTaskDir + "=" + taskDir
	if env == nil {
		return append(os.Environ(), pair)
	}
	return append(env, pair)
}

func taskFileMode(raw string) (os.FileMode, error) {
	mode, msg := api.ParseTaskFileMode(raw)
	if msg != "" {
		return 0, fmt.Errorf("%s", msg)
	}
	return os.FileMode(mode), nil
}

func isPathInsideRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
