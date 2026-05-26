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
