package public

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

func (s *Server) registerGitRepositoryRoutes(r *mux.Router) {
	gitRoute := r.PathPrefix("/git").Subrouter()
	gitRoute.Use(middleware.OrganizationAuthMiddleware(s.jwt))
	gitRoute.PathPrefix("/").HandlerFunc(s.handleGitRepositoryProxy).Methods(http.MethodGet, http.MethodPost)
}

func (s *Server) handleGitRepositoryProxy(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	canvasID, gitSuffix, ok := parseGitRepositoryPath(r.URL.Path)
	if !ok {
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	repository, err := models.FindRepository(user.OrganizationID, uuid.MustParse(canvasID))
	if err != nil {
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	targetBase := supergitGitBaseURL()
	if targetBase == "" {
		http.Error(w, "git storage is not configured", http.StatusServiceUnavailable)
		return
	}

	target, err := url.Parse(targetBase)
	if err != nil {
		http.Error(w, "git storage is not configured", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		path := "/git/" + repository.RepoID + ".git"
		if gitSuffix != "" {
			path += "/" + gitSuffix
		}

		req.URL.Path = path
		req.URL.RawPath = ""
		req.Host = target.Host
	}

	proxy.ServeHTTP(w, r)
}

func parseGitRepositoryPath(path string) (canvasID string, gitSuffix string, ok bool) {
	path = strings.TrimPrefix(path, "/git/")
	path = strings.Trim(path, "/")
	if path == "" {
		return "", "", false
	}

	dotGit := strings.Index(path, ".git")
	if dotGit < 0 {
		return "", "", false
	}

	canvasID = strings.Trim(path[:dotGit], "/")
	if canvasID == "" {
		return "", "", false
	}

	if _, err := uuid.Parse(canvasID); err != nil {
		return "", "", false
	}

	gitSuffix = strings.Trim(strings.TrimPrefix(path[dotGit+len(".git"):], "/"), "/")
	return canvasID, gitSuffix, true
}

func supergitGitBaseURL() string {
	if base := strings.TrimSpace(os.Getenv("GIT_STORAGE_SUPERGIT_GIT_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}

	apiBase := strings.TrimSpace(os.Getenv("GIT_STORAGE_SUPERGIT_BASE_URL"))
	if apiBase == "" {
		return ""
	}

	apiBase = strings.TrimRight(apiBase, "/")
	return strings.TrimSuffix(apiBase, "/api") + "/git"
}
