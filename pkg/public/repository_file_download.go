package public

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

func (s *Server) handleRepositoryFileDownload(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["canvas_id"]
	path := r.URL.Query().Get("path")
	branch := r.URL.Query().Get("branch")

	if id == "" {
		http.Error(w, "canvas_id is required", http.StatusBadRequest)
		return
	}

	canvasID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid canvas_id", http.StatusBadRequest)
		return
	}

	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return
	}

	allowed, err := s.authService.CheckOrganizationPermission(
		user.ID.String(),
		user.OrganizationID.String(),
		"canvases",
		"read",
	)

	if err != nil {
		log.Errorf("Failed to check permission: %v", err)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if !allowed {
		log.Warnf("User %s is not authorized to read file %s in canvas %s", user.ID.String(), path, canvasID.String())
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	canvas, err := models.FindCanvas(user.OrganizationID, canvasID)
	if err != nil {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}

	repository, err := models.FindRepository(user.OrganizationID, canvas.ID)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	reader, err := s.gitProvider.GetFile(r.Context(), repository.RepoID, path, branch)
	if err != nil {
		log.Errorf("Failed to get file %s in canvas %s: %v", path, canvasID.String(), err)
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}

	defer reader.Close()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/octet-stream")
	// Repository contents change on every commit while the URL (path + branch)
	// stays the same, so the browser must never serve a cached response.
	// Otherwise, after committing to a draft branch the UI keeps showing the
	// previously fetched (pre-commit) content, both immediately and after a full
	// page reload.
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": filepath.Base(path),
	}))

	_, err = io.Copy(w, reader)
	if err != nil {
		log.Errorf("Failed to copy file %s in canvas %s: %v", path, canvasID.String(), err)
		http.Error(w, "Failed to copy file", http.StatusInternalServerError)
		return
	}
}
