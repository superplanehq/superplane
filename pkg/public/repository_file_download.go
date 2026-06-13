package public

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

func (s *Server) handleRepositoryFileDownload(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["canvas_id"]
	path := r.URL.Query().Get("path")
	versionID := strings.TrimSpace(r.URL.Query().Get("version_id"))
	stage := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("stage")), "true")

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

	if canvases.IsRepositorySpecFilePath(path) {
		ctx := authentication.SetUserIdInMetadata(r.Context(), user.ID.String())
		readSpecFile := canvases.ReadRepositorySpecFile
		if stage {
			readSpecFile = canvases.ReadRepositorySpecFileStaged
		}
		content, readErr := readSpecFile(
			ctx,
			user.OrganizationID.String(),
			canvas.ID.String(),
			versionID,
			path,
		)
		if readErr != nil {
			log.Errorf("Failed to read repository spec file %s in canvas %s: %v", path, canvasID.String(), readErr)
			http.Error(w, "Failed to get file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
			"filename": filepath.Base(path),
		}))
		_, err = io.WriteString(w, content)
		if err != nil {
			log.Errorf("Failed to write repository spec file %s in canvas %s: %v", path, canvasID.String(), err)
		}
		return
	}

	// For arbitrary repository files on a draft, staged edits (stored in
	// workflow_staged_files) take precedence over the committed git content when the
	// caller opts in with ?stage=true.
	if stage && versionID != "" {
		ctx := authentication.SetUserIdInMetadata(r.Context(), user.ID.String())
		content, found, deleted, stagedErr := canvases.ReadStagedRepositoryFile(
			ctx,
			user.OrganizationID.String(),
			canvas.ID.String(),
			versionID,
			path,
		)
		if stagedErr != nil {
			log.Errorf("Failed to read staged repository file %s in canvas %s: %v", path, canvasID.String(), stagedErr)
			http.Error(w, "Failed to get file", http.StatusInternalServerError)
			return
		}
		if deleted {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		if found {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
				"filename": filepath.Base(path),
			}))
			if _, writeErr := io.WriteString(w, content); writeErr != nil {
				log.Errorf("Failed to write staged repository file %s in canvas %s: %v", path, canvasID.String(), writeErr)
			}
			return
		}
	}

	repository, err := models.FindRepository(user.OrganizationID, canvas.ID)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	reader, err := s.gitProvider.GetFile(r.Context(), repository.RepoID, path, "")
	if err != nil {
		log.Errorf("Failed to get file %s in canvas %s: %v", path, canvasID.String(), err)
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}

	defer reader.Close()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/octet-stream")
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
