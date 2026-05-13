package gitserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"gopkg.in/yaml.v3"
)

// BootstrapFromAPI exports the current canvas state from the API and creates
// the initial git commit. This is for existing canvases that predate the git feature.
func (s *Server) BootstrapFromAPI(slug, canvasID, orgID, baseURL, token string) error {
	repoPath, err := s.InitRepo(slug)
	if err != nil {
		return err
	}

	// Check if repo already has commits
	cmd := exec.Command("git", "--git-dir", repoPath, "rev-parse", "HEAD")
	if err := cmd.Run(); err == nil {
		log.Infof("gitserver: repo %s already has commits, skipping bootstrap", slug)
		return nil
	}

	client := &APIClient{BaseURL: baseURL, Token: token}

	// Create temp worktree
	workDir, err := os.MkdirTemp("", "sp-git-bootstrap-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=SuperPlane",
			"GIT_AUTHOR_EMAIL=system@superplane.com",
			"GIT_COMMITTER_NAME=SuperPlane",
			"GIT_COMMITTER_EMAIL=system@superplane.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s failed: %w: %s", args[0], err, out)
		}
		return nil
	}

	if err := run("init", "--initial-branch=main"); err != nil {
		// Fallback for older git
		if err2 := run("init"); err2 != nil {
			return err2
		}
		run("checkout", "-b", "main")
	}

	// 1. Export canvas YAML via CLI-compatible endpoint
	canvasYAML, err := client.ExportCanvasYAML(canvasID)
	if err != nil {
		log.Warnf("gitserver: failed to export canvas YAML: %v", err)
	} else {
		os.WriteFile(filepath.Join(workDir, "canvas.yaml"), canvasYAML, 0644)
	}

	// 2. Export README
	readme, err := client.ExportReadme(canvasID)
	if err != nil {
		log.Warnf("gitserver: failed to export readme: %v", err)
	} else if readme != "" {
		os.MkdirAll(filepath.Join(workDir, "docs"), 0755)
		os.WriteFile(filepath.Join(workDir, "docs", "README.md"), []byte(readme), 0644)
	}

	// 3. Write .superplane.yaml
	spConfig := fmt.Sprintf("canvasId: %s\norgId: %s\nslug: %s\n", canvasID, orgID, slug)
	os.WriteFile(filepath.Join(workDir, ".superplane.yaml"), []byte(spConfig), 0644)

	// 4. Commit
	if err := run("add", "-A"); err != nil {
		return err
	}
	if err := run("commit", "-m", "Initial commit (bootstrapped from canvas)"); err != nil {
		return err
	}

	// 5. Push to bare repo
	if err := run("remote", "add", "origin", repoPath); err != nil {
		return err
	}
	if err := run("push", "origin", "main"); err != nil {
		return fmt.Errorf("push to bare repo failed: %w", err)
	}

	// 6. Register in the slug registry
	configDest := filepath.Join(repoPath, "superplane.yaml")
	os.WriteFile(configDest, []byte(spConfig), 0644)

	log.Infof("gitserver: bootstrapped repo for %s (canvas %s) with initial commit", slug, canvasID)
	return nil
}

// ExportCanvasYAML gets the canvas and serializes it as CLI-compatible YAML.
func (c *APIClient) ExportCanvasYAML(canvasID string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/canvases/%s", c.BaseURL, canvasID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("canvas GET returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON canvas response into the CLI model structure
	var apiResp struct {
		Canvas json.RawMessage `json:"canvas"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &apiResp)

	// Unmarshal into the CLI Canvas model
	var canvas models.Canvas
	canvas.APIVersion = "v1"
	canvas.Kind = "Canvas"
	json.Unmarshal(apiResp.Canvas, &canvas)

	// Marshal as YAML
	yamlBytes, err := yaml.Marshal(canvas)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canvas as YAML: %w", err)
	}

	return yamlBytes, nil
}

// ExportReadme gets the canvas readme content.
func (c *APIClient) ExportReadme(canvasID string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/canvases/%s/readme", c.BaseURL, canvasID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("readme GET returned %d", resp.StatusCode)
	}

	var result struct {
		Content string `json:"content"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Content, nil
}

// ToSlug converts a canvas name to a git-friendly slug.
func ToSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric chars except hyphens
	var clean []byte
	for _, c := range []byte(slug) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			clean = append(clean, c)
		}
	}
	return string(clean)
}

// BootstrapFromDB exports the current canvas state from the DB and creates
// the initial git commit. No API token needed.
func (s *Server) BootstrapFromDB(slug, canvasID, orgID string) error {
	reader := &InternalReader{}

	repoPath, err := s.InitRepo(slug)
	if err != nil {
		return err
	}

	// Check if repo already has commits
	cmd := exec.Command("git", "--git-dir", repoPath, "rev-parse", "HEAD")
	if err := cmd.Run(); err == nil {
		log.Infof("gitserver: repo %s already has commits, skipping bootstrap", slug)
		return nil
	}

	workDir, err := os.MkdirTemp("", "sp-git-bootstrap-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=SuperPlane",
			"GIT_AUTHOR_EMAIL=system@superplane.com",
			"GIT_COMMITTER_NAME=SuperPlane",
			"GIT_COMMITTER_EMAIL=system@superplane.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s failed: %w: %s", args[0], err, out)
		}
		return nil
	}

	if err := run("init", "--initial-branch=main"); err != nil {
		if err2 := run("init"); err2 != nil {
			return err2
		}
		run("checkout", "-b", "main")
	}

	// Export canvas YAML from DB
	canvasYAML, err := reader.ReadCanvasYAML(canvasID)
	if err != nil {
		log.Warnf("gitserver: failed to read canvas YAML: %v", err)
	} else {
		os.WriteFile(filepath.Join(workDir, "canvas.yaml"), canvasYAML, 0644)
	}

	// Export README from DB
	readme, err := reader.ReadReadme(canvasID)
	if err != nil {
		log.Warnf("gitserver: failed to read readme: %v", err)
	} else if readme != "" {
		os.MkdirAll(filepath.Join(workDir, "docs"), 0755)
		os.WriteFile(filepath.Join(workDir, "docs", "README.md"), []byte(readme), 0644)
	}

	// Export launchpad panels
	lp, err := reader.ReadLaunchpad(canvasID)
	if err != nil {
		log.Warnf("gitserver: failed to read launchpad: %v", err)
	} else if len(lp.Panels) > 0 {
		exportAppsToDir(filepath.Join(workDir, "apps"), lp)
	}

	// Write .superplane.yaml
	spConfig := fmt.Sprintf("canvasId: %s\norgId: %s\nslug: %s\n", canvasID, orgID, slug)
	os.WriteFile(filepath.Join(workDir, ".superplane.yaml"), []byte(spConfig), 0644)

	if err := run("add", "-A"); err != nil {
		return err
	}
	if err := run("commit", "-m", "Initial commit (bootstrapped from canvas)"); err != nil {
		return err
	}
	if err := run("remote", "add", "origin", repoPath); err != nil {
		return err
	}
	if err := run("push", "origin", "main"); err != nil {
		return fmt.Errorf("push to bare repo failed: %w", err)
	}

	configDest := filepath.Join(repoPath, "superplane.yaml")
	os.WriteFile(configDest, []byte(spConfig), 0644)

	log.Infof("gitserver: bootstrapped repo for %s (canvas %s) from DB", slug, canvasID)
	return nil
}
