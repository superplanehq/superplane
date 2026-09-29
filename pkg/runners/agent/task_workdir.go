package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveTaskWorkDir returns the directory host-mode tasks start in: the runner
// process user's home directory, or "." when HOME is unavailable.
func ResolveTaskWorkDir() (string, error) {
	dir := "."
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		dir = home
	}
	dir = filepath.Clean(dir)
	st, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("task work dir %q: %w", dir, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("task work dir %q is not a directory", dir)
	}
	return dir, nil
}

func isolatedTaskHome(parent, taskID string) string {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		parent = "."
	}
	id := strings.TrimSpace(taskID)
	if id == "" {
		id = "unknown"
	}
	return filepath.Join(parent, ".superplane", "homes", id)
}

func createIsolatedTaskHome(parent, taskID string) (string, error) {
	dir := isolatedTaskHome(parent, taskID)
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("reset task home %q: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create task home %q: %w", dir, err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("task home %q: %w", dir, err)
	}
	return abs, nil
}
