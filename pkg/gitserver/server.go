// Package gitserver implements a Git HTTP smart protocol server.
// Each canvas gets its own bare repo. Shells out to the git binary
// for protocol handling (simpler and more compatible than go-git).
package gitserver

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Server manages bare git repositories for canvases.
type Server struct {
	reposDir string
	mu       sync.RWMutex

	// AuthFunc validates a token and returns error if invalid.
	AuthFunc func(token string) error

	// OnPush is called after a successful push to main.
	OnPush func(slug string, repoPath string) error
}

// NewServer creates a git server with repos stored under reposDir.
func NewServer(reposDir string) (*Server, error) {
	if err := os.MkdirAll(reposDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repos dir: %w", err)
	}

	// Verify git binary is available
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git binary not found in PATH: %w", err)
	}

	return &Server{reposDir: reposDir}, nil
}

// RepoPath returns the filesystem path for a canvas slug's bare repo.
func (s *Server) RepoPath(slug string) string {
	return filepath.Join(s.reposDir, slug+".git")
}

// InitRepo creates a bare repo for a canvas if it doesn't exist.
// Returns the repo path.
func (s *Server) InitRepo(slug string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repoPath := s.RepoPath(slug)
	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); err == nil {
		return repoPath, nil // Already exists
	}

	cmd := exec.Command("git", "init", "--bare", "--initial-branch=main", repoPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback for older git versions that don't support --initial-branch
		cmd2 := exec.Command("git", "init", "--bare", repoPath)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("git init --bare failed: %w: %s", err2, out2)
		}
		// Set HEAD to main
		exec.Command("git", "-C", repoPath, "symbolic-ref", "HEAD", "refs/heads/main").Run()
		_ = out
	}

	log.Infof("gitserver: initialized bare repo for %q at %s", slug, repoPath)
	return repoPath, nil
}

// RegisterRoutes adds git HTTP smart protocol routes to the router.
func (s *Server) RegisterRoutes(router *mux.Router) {
	// Match both /git/slug and /git/slug.git
	sub := router.PathPrefix("/git/").Subrouter()
	sub.HandleFunc("/{slug}/info/refs", s.handleInfoRefs).Methods("GET")
	sub.HandleFunc("/{slug}.git/info/refs", s.handleInfoRefs).Methods("GET")
	sub.HandleFunc("/{slug}/git-upload-pack", s.handleServiceRPC).Methods("POST")
	sub.HandleFunc("/{slug}.git/git-upload-pack", s.handleServiceRPC).Methods("POST")
	sub.HandleFunc("/{slug}/git-receive-pack", s.handleServiceRPC).Methods("POST")
	sub.HandleFunc("/{slug}.git/git-receive-pack", s.handleServiceRPC).Methods("POST")
}

// authenticate extracts Basic auth password and validates it.
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) bool {
	_, password, ok := r.BasicAuth()
	if !ok || password == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="SuperPlane Git"`)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return false
	}

	if s.AuthFunc != nil {
		if err := s.AuthFunc(password); err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="SuperPlane Git"`)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return false
		}
	}

	return true
}

func (s *Server) getSlug(r *http.Request) string {
	slug := mux.Vars(r)["slug"]
	slug = strings.TrimSuffix(slug, ".git")
	return slug
}

func (s *Server) getRepoPath(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	slug := s.getSlug(r)
	repoPath := s.RepoPath(slug)

	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); os.IsNotExist(err) {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return "", "", false
	}

	return slug, repoPath, true
}

// handleInfoRefs handles GET /info/refs?service=git-upload-pack|git-receive-pack
func (s *Server) handleInfoRefs(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(w, r) {
		return
	}

	_, repoPath, ok := s.getRepoPath(w, r)
	if !ok {
		return
	}

	service := r.URL.Query().Get("service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "Invalid service", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.Header().Set("Cache-Control", "no-cache")

	// Pkt-line service announcement
	serverAdvert := fmt.Sprintf("# service=%s\n", service)
	fmt.Fprintf(w, "%04x%s0000", len(serverAdvert)+4, serverAdvert)

	// Run git <service> --advertise-refs
	serviceName := strings.TrimPrefix(service, "git-")
	cmd := exec.Command("git", serviceName, "--stateless-rpc", "--advertise-refs", repoPath)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Errorf("gitserver: info/refs error: %v", err)
	}
}

// handleServiceRPC handles POST /git-upload-pack and /git-receive-pack
func (s *Server) handleServiceRPC(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(w, r) {
		return
	}

	slug, repoPath, ok := s.getRepoPath(w, r)
	if !ok {
		return
	}

	// Determine service from URL path
	var service string
	if strings.HasSuffix(r.URL.Path, "git-upload-pack") {
		service = "upload-pack"
	} else if strings.HasSuffix(r.URL.Path, "git-receive-pack") {
		service = "receive-pack"
	} else {
		http.Error(w, "Invalid service", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))
	w.Header().Set("Cache-Control", "no-cache")

	// Handle gzip-encoded request body
	var body io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Failed to decompress request", http.StatusBadRequest)
			return
		}
		defer gz.Close()
		body = gz
	}

	cmd := exec.Command("git", service, "--stateless-rpc", repoPath)
	cmd.Stdin = body
	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Errorf("gitserver: %s error for %s: %v", service, slug, err)
		return
	}

	// Fire post-receive hook for pushes
	if service == "receive-pack" && s.OnPush != nil {
		go func() {
			if err := s.OnPush(slug, repoPath); err != nil {
				log.Errorf("gitserver: post-push sync failed for %s: %v", slug, err)
			}
		}()
	}
}
