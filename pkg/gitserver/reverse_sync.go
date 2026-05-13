package gitserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// ReverseSync auto-commits canvas state to git after UI publishes.
type ReverseSync struct {
	gitServer *Server
	registry  *Registry
	reader    *InternalReader

	// Debounce: avoid committing twice for rapid successive publishes
	mu         sync.Mutex
	lastCommit map[string]time.Time
}

func NewReverseSync(gitServer *Server, registry *Registry) *ReverseSync {
	return &ReverseSync{
		gitServer:  gitServer,
		registry:   registry,
		reader:     &InternalReader{},
		lastCommit: make(map[string]time.Time),
	}
}

// OnCanvasPublished is called after a canvas version is published via UI/API.
// It exports the new state and commits it to the git repo.
func (rs *ReverseSync) OnCanvasPublished(canvasID, orgID, userName string) {
	// Skip if this publish was triggered by a git push (prevent infinite loop)
	if IsSyncActive(canvasID) {
		log.Debugf("git-reverse-sync: skipping for %s (triggered by git push)", canvasID)
		return
	}

	rs.mu.Lock()
	if last, ok := rs.lastCommit[canvasID]; ok && time.Since(last) < 5*time.Second {
		rs.mu.Unlock()
		log.Debugf("git-reverse-sync: debounced for canvas %s", canvasID)
		return
	}
	rs.lastCommit[canvasID] = time.Now()
	rs.mu.Unlock()

	// Find the slug for this canvas
	slug := rs.findSlugByCanvasID(canvasID)
	if slug == "" {
		return // No git repo for this canvas
	}

	repoPath := rs.gitServer.RepoPath(slug)
	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); os.IsNotExist(err) {
		return // No repo
	}

	go func() {
		if err := rs.commitCurrentState(slug, canvasID, userName); err != nil {
			log.Errorf("git-reverse-sync: failed for %s: %v", slug, err)
		}
	}()
}

func (rs *ReverseSync) findSlugByCanvasID(canvasID string) string {
	rs.registry.mu.RLock()
	defer rs.registry.mu.RUnlock()

	for slug, mapping := range rs.registry.mappings {
		if mapping.CanvasID == canvasID {
			return slug
		}
	}
	return ""
}

func (rs *ReverseSync) commitCurrentState(slug, canvasID, userName string) error {
	repoPath := rs.gitServer.RepoPath(slug)

	// Clone to temp worktree
	workDir, err := os.MkdirTemp("", "sp-git-reverse-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir

		authorName := userName
		if authorName == "" {
			authorName = "SuperPlane UI"
		}

		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME="+authorName,
			"GIT_AUTHOR_EMAIL=ui@superplane.com",
			"GIT_COMMITTER_NAME=SuperPlane",
			"GIT_COMMITTER_EMAIL=system@superplane.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s failed: %w: %s", args[0], err, out)
		}
		return nil
	}

	// Clone the bare repo
	if err := run("clone", repoPath, "."); err != nil {
		return err
	}

	// Export current canvas state directly from DB (no API token needed)
	canvasYAML, err := rs.reader.ReadCanvasYAML(canvasID)
	if err != nil {
		return fmt.Errorf("failed to read canvas: %w", err)
	}
	os.WriteFile(filepath.Join(workDir, "canvas.yaml"), canvasYAML, 0644)

	readme, err := rs.reader.ReadReadme(canvasID)
	if err != nil {
		log.Warnf("git-reverse-sync: failed to read readme: %v", err)
	} else if readme != "" {
		os.WriteFile(filepath.Join(workDir, "README.md"), []byte(readme), 0644)
	}

	// Check if anything changed
	checkCmd := exec.Command("git", "status", "--porcelain")
	checkCmd.Dir = workDir
	statusOut, _ := checkCmd.Output()
	if len(statusOut) == 0 {
		log.Debugf("git-reverse-sync: no changes for %s, skipping commit", slug)
		return nil
	}

	// Commit and push
	if err := run("add", "-A"); err != nil {
		return err
	}
	if err := run("commit", "-m", fmt.Sprintf("UI publish by %s", userName)); err != nil {
		return err
	}
	if err := run("push", "origin", "main"); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	log.Infof("git-reverse-sync: committed UI changes for %s by %s", slug, userName)
	return nil
}

