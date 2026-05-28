package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/supergit/internal/api/ndjson"
	"github.com/superplanehq/superplane/supergit/internal/config"
	"github.com/superplanehq/superplane/supergit/internal/storage"
)

type Server struct {
	store  *storage.Store
	config config.Config
}

func NewServer(store *storage.Store, cfg config.Config) *Server {
	return &Server{store: store, config: cfg}
}

func (s *Server) Router() http.Handler {
	router := mux.NewRouter()

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/repos", s.listRepos).Methods(http.MethodGet)
	api.HandleFunc("/repos", s.createRepo).Methods(http.MethodPost)
	api.HandleFunc("/repos/{id:.+}/files", s.files).Methods(http.MethodGet)
	api.HandleFunc("/repos/{id:.+}/commits", s.listCommits).Methods(http.MethodGet)
	api.HandleFunc("/repos/{id:.+}/commits", s.createCommit).Methods(http.MethodPost)
	api.HandleFunc("/repos/{id:.+}/commit", s.getCommit).Methods(http.MethodGet)
	api.HandleFunc("/repos/{id:.+}", s.getRepo).Methods(http.MethodGet)
	api.HandleFunc("/repos/{id:.+}", s.deleteRepo).Methods(http.MethodDelete)

	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return router
}

func (s *Server) listRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := s.store.ListRepositories(r.Context())
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"repos":       repos,
		"next_cursor": "",
		"has_more":    false,
	})
}

type createRepoRequest struct {
	ID            string `json:"id"`
	DefaultBranch string `json:"default_branch"`
}

func (s *Server) createRepo(w http.ResponseWriter, r *http.Request) {
	var req createRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	repo, err := s.store.CreateRepository(r.Context(), storage.RepositorySpec{
		ID:            req.ID,
		DefaultBranch: req.DefaultBranch,
	})
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, repo)
}

func (s *Server) getRepo(w http.ResponseWriter, r *http.Request) {
	repoID, err := repoIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	repo, err := s.store.GetRepository(r.Context(), repoID)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, repo)
}

func (s *Server) deleteRepo(w http.ResponseWriter, r *http.Request) {
	repoID, err := repoIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = s.store.DeleteRepository(r.Context(), storage.RepositoryRef{ID: repoID})
	if err != nil {
		writeStorageError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) files(w http.ResponseWriter, r *http.Request) {
	repoID, err := repoIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	filePath := strings.TrimSpace(r.URL.Query().Get("path"))
	ref := strings.TrimSpace(r.URL.Query().Get("ref"))

	repo, err := s.store.GetRepository(r.Context(), repoID)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	repositoryRef := storage.RepositoryRef{
		ID:            repo.ID,
		DefaultBranch: repo.DefaultBranch,
	}

	if filePath != "" {
		reader, err := s.store.GetFile(r.Context(), repositoryRef, storage.GetFileOptions{
			Path: filePath,
			Ref:  ref,
		})
		if err != nil {
			writeStorageError(w, err)
			return
		}
		defer reader.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, reader)
		return
	}

	result, err := s.store.ListFiles(r.Context(), repositoryRef, storage.ListFilesOptions{Ref: ref})
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listCommits(w http.ResponseWriter, r *http.Request) {
	repoID, err := repoIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	repo, err := s.store.GetRepository(r.Context(), repoID)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	commits, err := s.store.ListCommits(r.Context(), storage.RepositoryRef{
		ID:            repo.ID,
		DefaultBranch: repo.DefaultBranch,
	}, branch, limit)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"commits":     commits,
		"next_cursor": "",
		"has_more":    false,
	})
}

func (s *Server) createCommit(w http.ResponseWriter, r *http.Request) {
	repoID, err := repoIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	repo, err := s.store.GetRepository(r.Context(), repoID)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	options, err := ndjson.ParseCommitBody(r.Body, s.storeLimits())
	if err != nil {
		writeStorageError(w, err)
		return
	}

	result, err := s.store.Commit(r.Context(), storage.RepositoryRef{
		ID:            repo.ID,
		DefaultBranch: repo.DefaultBranch,
	}, options)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"commit": map[string]any{
			"commit_sha": result.CommitSHA,
		},
		"result": map[string]any{
			"branch":  refOrDefault(options.Branch, repo.DefaultBranch),
			"new_sha": result.CommitSHA,
			"old_sha": result.OldSHA,
			"success": true,
		},
	})
}

func (s *Server) getCommit(w http.ResponseWriter, r *http.Request) {
	repoID, err := repoIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sha := strings.TrimSpace(r.URL.Query().Get("sha"))
	if sha == "" {
		writeError(w, http.StatusBadRequest, "sha query parameter is required")
		return
	}

	repo, err := s.store.GetRepository(r.Context(), repoID)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	commit, err := s.store.GetCommit(r.Context(), storage.RepositoryRef{
		ID:            repo.ID,
		DefaultBranch: repo.DefaultBranch,
	}, sha)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, commit)
}

func (s *Server) storeLimits() storage.Limits {
	return storage.Limits{
		MaxFileBytes:   s.config.MaxFileBytes,
		MaxCommitBytes: s.config.MaxCommitBytes,
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeStorageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrInvalidPath),
		errors.Is(err, storage.ErrInvalidRepositoryID),
		errors.Is(err, storage.ErrReservedPath),
		errors.Is(err, storage.ErrInvalidCommit):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, storage.ErrFileTooLarge),
		errors.Is(err, storage.ErrCommitTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, storage.ErrExpectedHeadMismatch):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, storage.ErrRepositoryNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func refOrDefault(ref, branch string) string {
	ref = strings.TrimSpace(ref)
	if ref != "" {
		return ref
	}
	return branch
}

func repoIDFromRequest(r *http.Request) (string, error) {
	repoID := strings.TrimSpace(mux.Vars(r)["id"])
	if repoID == "" {
		return "", errors.New("repository id is required")
	}

	decoded, err := url.PathUnescape(repoID)
	if err != nil {
		return "", errors.New("invalid repository id encoding")
	}

	return decoded, nil
}
