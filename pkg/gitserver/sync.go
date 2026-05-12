package gitserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// SyncHandler processes a push by reading changed files from the repo
// and calling the appropriate update functions.
type SyncHandler struct {
	// UpdateCanvas is called with the canvas YAML content when canvas.yaml changes.
	UpdateCanvas func(slug string, yamlContent []byte) error

	// UpdateReadme is called with the readme content when README.md changes.
	UpdateReadme func(slug string, content string) error

	// UpdateApps is called with panel files when apps/ changes.
	// panels maps filename -> content, layout is the _layout.json content.
	UpdateApps func(slug string, panels map[string]string, layout []byte) error
}

// HandlePush is called after a push to the repo.
// It checks out the latest main, diffs against previous HEAD, and syncs changed artifacts.
func (h *SyncHandler) HandlePush(slug string, repoPath string) error {
	log.Infof("gitserver/sync: processing push for %s", slug)

	// Create a temporary worktree to read files
	workDir, err := os.MkdirTemp("", "sp-git-sync-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Clone the bare repo into the temp dir (shallow, just HEAD)
	cmd := exec.Command("git", "clone", "--depth=1", "--single-branch", repoPath, workDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone for sync: %w: %s", err, out)
	}

	// For POC: sync all known artifacts on every push (no diffing yet)
	// This is simpler and avoids issues with initial pushes having no previous HEAD.

	// 1. Canvas YAML
	if h.UpdateCanvas != nil {
		canvasPath := filepath.Join(workDir, "canvas.yaml")
		if content, err := os.ReadFile(canvasPath); err == nil {
			log.Infof("gitserver/sync: updating canvas for %s", slug)
			if err := h.UpdateCanvas(slug, content); err != nil {
				log.Errorf("gitserver/sync: canvas update failed for %s: %v", slug, err)
			}
		}
	}

	// 2. README
	if h.UpdateReadme != nil {
		readmePath := filepath.Join(workDir, "README.md")
		if content, err := os.ReadFile(readmePath); err == nil {
			log.Infof("gitserver/sync: updating readme for %s", slug)
			if err := h.UpdateReadme(slug, string(content)); err != nil {
				log.Errorf("gitserver/sync: readme update failed for %s: %v", slug, err)
			}
		}
	}

	// 3. Apps panels
	if h.UpdateApps != nil {
		appsDir := filepath.Join(workDir, "apps")
		if info, err := os.Stat(appsDir); err == nil && info.IsDir() {
			panels := make(map[string]string)
			var layoutContent []byte

			entries, _ := os.ReadDir(appsDir)
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				name := entry.Name()
				content, err := os.ReadFile(filepath.Join(appsDir, name))
				if err != nil {
					continue
				}

				if name == "_layout.json" {
					layoutContent = content
				} else if strings.HasSuffix(name, ".md") {
					panelID := strings.TrimSuffix(name, ".md")
					panels[panelID] = string(content)
				}
			}

			if len(panels) > 0 || layoutContent != nil {
				log.Infof("gitserver/sync: updating apps for %s (%d panels)", slug, len(panels))
				if err := h.UpdateApps(slug, panels, layoutContent); err != nil {
					log.Errorf("gitserver/sync: apps update failed for %s: %v", slug, err)
				}
			}
		}
	}

	log.Infof("gitserver/sync: push sync complete for %s", slug)
	return nil
}
