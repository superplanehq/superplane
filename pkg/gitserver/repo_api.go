package gitserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// RegisterRepoAPIRoutes adds repo file browsing endpoints on the standard API path.
// These use cookie-based auth (session), not Basic auth.
func (s *Server) RegisterRepoAPIRoutes(router *mux.Router) {
	// Register directly on the main router — no subrouter middleware
	// (auth is handled by the existing session cookie via the org-auth path)
	router.HandleFunc("/api/repo/{canvasId}/archive", s.handleRepoArchive).Methods("GET")
	router.HandleFunc("/api/repo/{canvasId}/files", s.handleRepoListFiles).Methods("GET")
	router.HandleFunc("/api/repo/{canvasId}/files/{path:.*}", s.handleRepoGetFile).Methods("GET")
	router.HandleFunc("/api/repo/{canvasId}/files/{path:.*}", s.handleRepoUpdateFile).Methods("PUT")
}

func (s *Server) resolveSlugFromCanvasID(canvasID string, registry *Registry) string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	for slug, m := range registry.mappings {
		if m.CanvasID == canvasID {
			return slug
		}
	}
	return ""
}

func (s *Server) handleRepoListFiles(w http.ResponseWriter, r *http.Request) {
	canvasID := mux.Vars(r)["canvasId"]
	slug := s.resolveSlugFromCanvasID(canvasID, s.Registry)
	if slug == "" {
		http.Error(w, "No repository for this canvas", http.StatusNotFound)
		return
	}

	repoPath := s.RepoPath(slug)
	cmd := exec.Command("git", "--git-dir", repoPath, "ls-tree", "-r", "--long", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		size := int64(0)
		fmt.Sscanf(parts[3], "%d", &size)
		path := strings.Join(parts[4:], " ")
		files = append(files, FileInfo{Path: path, Size: size})
	}

	commit := getLatestCommit(repoPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files":  files,
		"commit": commit,
		"slug":   slug,
	})
}

func (s *Server) handleRepoGetFile(w http.ResponseWriter, r *http.Request) {
	canvasID := mux.Vars(r)["canvasId"]
	filePath := mux.Vars(r)["path"]
	slug := s.resolveSlugFromCanvasID(canvasID, s.Registry)
	if slug == "" {
		http.Error(w, "No repository for this canvas", http.StatusNotFound)
		return
	}

	repoPath := s.RepoPath(slug)
	cmd := exec.Command("git", "--git-dir", repoPath, "show", "HEAD:"+filePath)
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	shaCmd := exec.Command("git", "--git-dir", repoPath, "rev-parse", "HEAD")
	shaOut, _ := shaCmd.Output()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FileContent{
		Path:     filePath,
		Content:  string(out),
		SHA:      strings.TrimSpace(string(shaOut)),
		Editable: strings.HasSuffix(filePath, ".md"),
	})
}

func (s *Server) handleRepoUpdateFile(w http.ResponseWriter, r *http.Request) {
	canvasID := mux.Vars(r)["canvasId"]
	filePath := mux.Vars(r)["path"]
	slug := s.resolveSlugFromCanvasID(canvasID, s.Registry)
	if slug == "" {
		http.Error(w, "No repository for this canvas", http.StatusNotFound)
		return
	}

	if !strings.HasSuffix(filePath, ".md") {
		http.Error(w, "Only markdown files can be edited", http.StatusForbidden)
		return
	}

	var body struct {
		Content string `json:"content"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Message == "" {
		body.Message = fmt.Sprintf("Update %s", filePath)
	}

	repoPath := s.RepoPath(slug)
	workDir, err := createTempClone(repoPath)
	if err != nil {
		log.Errorf("repo-api: clone failed: %v", err)
		http.Error(w, "Failed to update file", http.StatusInternalServerError)
		return
	}
	defer removeTempDir(workDir)

	// Write the file
	fullPath := filepath.Join(workDir, filePath)
	if err := writeFileWithDirs(fullPath, []byte(body.Content)); err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	// Commit and push
	authorName := "SuperPlane UI" // TODO: resolve from session
	if err := commitAndPush(workDir, filePath, body.Message, authorName); err != nil {
		if strings.Contains(err.Error(), "nothing to commit") {
			http.Error(w, "No changes to commit", http.StatusBadRequest)
			return
		}
		log.Errorf("repo-api: commit failed: %v", err)
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}

	// Fire forward sync (same as git push via HTTP)
	if s.OnPush != nil {
		go func() {
			if err := s.OnPush(slug, repoPath); err != nil {
				log.Errorf("repo-api: post-save sync failed for %s: %v", slug, err)
			}
		}()
	}

	shaCmd := exec.Command("git", "--git-dir", repoPath, "rev-parse", "HEAD")
	shaOut, _ := shaCmd.Output()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"sha": strings.TrimSpace(string(shaOut)),
	})
}

func (s *Server) handleRepoArchive(w http.ResponseWriter, r *http.Request) {
	canvasID := mux.Vars(r)["canvasId"]
	slug := s.resolveSlugFromCanvasID(canvasID, s.Registry)
	if slug == "" {
		http.Error(w, "No repository for this canvas", http.StatusNotFound)
		return
	}

	repoPath := s.RepoPath(slug)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, slug))

	cmd := exec.Command("git", "--git-dir", repoPath, "archive", "--format=zip", "HEAD")
	cmd.Stdout = w
	if err := cmd.Run(); err != nil {
		log.Errorf("repo-api: archive failed for %s: %v", slug, err)
	}
}
