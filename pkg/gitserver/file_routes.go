package gitserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// FileInfo represents a file in the repo.
type FileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Dir  bool   `json:"dir,omitempty"`
}

// CommitInfo represents the latest commit.
type CommitInfo struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Date    string `json:"date"`
	Author  string `json:"author"`
}

// FileContent represents a file's content.
type FileContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	SHA      string `json:"sha"`
	Editable bool   `json:"editable"`
}

// RegisterFileRoutes adds file browsing endpoints.
func (s *Server) RegisterFileRoutes(router *mux.Router) {
	sub := router.PathPrefix("/git/").Subrouter()
	sub.HandleFunc("/{slug}/files", s.handleListFiles).Methods("GET")
	sub.HandleFunc("/{slug}/files/{path:.*}", s.handleGetFile).Methods("GET")
	sub.HandleFunc("/{slug}/files/{path:.*}", s.handleUpdateFile).Methods("PUT")
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(w, r) {
		return
	}

	slug := s.getSlug(r)
	repoPath := s.RepoPath(slug)

	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); os.IsNotExist(err) {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// List files via git ls-tree
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
		// Format: <mode> <type> <sha> <size> <path>
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		size := int64(0)
		fmt.Sscanf(parts[3], "%d", &size)
		path := strings.Join(parts[4:], " ")
		files = append(files, FileInfo{Path: path, Size: size})
	}

	// Get latest commit info
	commit := getLatestCommit(repoPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files":  files,
		"commit": commit,
	})
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(w, r) {
		return
	}

	slug := s.getSlug(r)
	repoPath := s.RepoPath(slug)
	filePath := mux.Vars(r)["path"]

	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); os.IsNotExist(err) {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// Read file via git show
	cmd := exec.Command("git", "--git-dir", repoPath, "show", "HEAD:"+filePath)
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Get file SHA
	shaCmd := exec.Command("git", "--git-dir", repoPath, "rev-parse", "HEAD")
	shaOut, _ := shaCmd.Output()
	sha := strings.TrimSpace(string(shaOut))

	editable := strings.HasSuffix(filePath, ".md")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FileContent{
		Path:     filePath,
		Content:  string(out),
		SHA:      sha,
		Editable: editable,
	})
}

func (s *Server) handleUpdateFile(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(w, r) {
		return
	}

	slug := s.getSlug(r)
	repoPath := s.RepoPath(slug)
	filePath := mux.Vars(r)["path"]

	// Only allow editing .md files
	if !strings.HasSuffix(filePath, ".md") {
		http.Error(w, "Only markdown files can be edited", http.StatusForbidden)
		return
	}

	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); os.IsNotExist(err) {
		http.Error(w, "Repository not found", http.StatusNotFound)
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

	// Get author from Basic auth token
	_, token, _ := r.BasicAuth()
	authorName := "SuperPlane"
	if s.AuthFunc != nil && token != "" {
		// Try to resolve user name from token
		authorName = resolveUserName(token)
	}

	// Clone, write, commit, push
	workDir, err := os.MkdirTemp("", "sp-git-edit-*")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(workDir)

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME="+authorName,
			"GIT_AUTHOR_EMAIL=ui@superplane.com",
			"GIT_COMMITTER_NAME=SuperPlane",
			"GIT_COMMITTER_EMAIL=system@superplane.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s: %w: %s", args[0], err, out)
		}
		return nil
	}

	if err := run("clone", repoPath, "."); err != nil {
		log.Errorf("git-file-update: clone failed: %v", err)
		http.Error(w, "Failed to update file", http.StatusInternalServerError)
		return
	}

	// Ensure parent directory exists
	fullPath := filepath.Join(workDir, filePath)
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	if err := os.WriteFile(fullPath, []byte(body.Content), 0644); err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	if err := run("add", filePath); err != nil {
		http.Error(w, "Failed to stage file", http.StatusInternalServerError)
		return
	}

	if err := run("commit", "-m", body.Message); err != nil {
		// Might be "nothing to commit"
		http.Error(w, "No changes to commit", http.StatusBadRequest)
		return
	}

	if err := run("push", "origin", "main"); err != nil {
		http.Error(w, "Failed to push", http.StatusInternalServerError)
		return
	}

	// Get new SHA
	shaCmd := exec.Command("git", "--git-dir", repoPath, "rev-parse", "HEAD")
	shaOut, _ := shaCmd.Output()
	sha := strings.TrimSpace(string(shaOut))

	// Forward sync will fire automatically from the push

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sha": sha,
		"commit": CommitInfo{
			SHA:     sha,
			Message: body.Message,
			Date:    time.Now().UTC().Format(time.RFC3339),
			Author:  authorName,
		},
	})
}

func getLatestCommit(repoPath string) *CommitInfo {
	cmd := exec.Command("git", "--git-dir", repoPath, "log", "-1", "--format=%H|%s|%aI|%an")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 4)
	if len(parts) < 4 {
		return nil
	}

	return &CommitInfo{
		SHA:     parts[0],
		Message: parts[1],
		Date:    parts[2],
		Author:  parts[3],
	}
}

func resolveUserName(token string) string {
	hashedToken := hashToken(token)
	// Try to find user by token hash
	user, err := findUserByTokenHash(hashedToken)
	if err != nil || user == "" {
		return "SuperPlane"
	}
	return user
}
