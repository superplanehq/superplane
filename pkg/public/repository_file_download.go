package public

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/services/files"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/gorm"
)

func (s *Server) handleRepositoryFileDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return
	}

	id := mux.Vars(r)["canvas_id"]
	if id == "" {
		http.Error(w, "canvas_id is required", http.StatusBadRequest)
		return
	}

	canvasID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Invalid canvas_id", http.StatusBadRequest)
		return
	}

	allowed, err := s.checkRepositoryReadPermission(ctx, user)
	if err != nil {
		log.Errorf("Failed to check permission: %v", err)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if !allowed {
		log.Warnf("User %s is not authorized to read files in canvas %s", user.ID.String(), canvasID.String())
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	db := database.DB(ctx)
	canvas, err := s.findRepositoryCanvas(ctx, db, user, canvasID)
	if err != nil {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}

	appFileReader := files.NewAppFileReader(db, canvas, user.ID)
	version := strings.TrimSpace(r.URL.Query().Get("version_id"))
	stage := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("stage")), "true")

	switch {

	//
	// If version_id is provided, we read the file for a specific version.
	//
	case version != "":
		reader, err := appFileReader.ReadFromVersion(ctx, path, version)
		if err != nil {
			http.Error(w, "Failed to read file from version", http.StatusInternalServerError)
			return
		}

		defer reader.Close()
		setInlineFileHeaders(w, path, "application/octet-stream")
		io.Copy(w, reader)
		return

	//
	// If stage=true is provided, we read the file from the user's staging area,
	// if it's there, or from the live version if it's not.
	// NOTE: still not sure about this behavior.
	//
	case stage:
		reader, err := appFileReader.Read(ctx, path)
		if err != nil {
			if errors.Is(err, files.ErrFileNotFound) {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}

			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}

		defer reader.Close()
		setInlineFileHeaders(w, path, "application/octet-stream")
		io.Copy(w, reader)
		return

	//
	// Otherwise, we read from the live version.
	//
	default:
		version, err := models.FindCanvasVersionInTransaction(db, canvas.ID, *canvas.LiveVersionID)
		if err != nil {
			http.Error(w, "Version not found", http.StatusNotFound)
			return
		}

		reader, err := appFileReader.ReadFromVersion(ctx, path, version.ID.String())
		if err != nil {
			http.Error(w, "Failed to read file from version", http.StatusInternalServerError)
			return
		}

		defer reader.Close()
		setInlineFileHeaders(w, path, "application/octet-stream")
		io.Copy(w, reader)
		return
	}
}

func (s *Server) checkRepositoryReadPermission(ctx context.Context, user *models.User) (allowed bool, err error) {
	ctx, done := telemetry.Span(ctx, "repository.check_permission")
	defer done(&err)

	return s.authService.CheckOrganizationPermission(ctx,
		user.ID.String(),
		user.OrganizationID.String(),
		"canvases",
		"read",
	)
}

func (s *Server) findRepositoryCanvas(ctx context.Context, db *gorm.DB, user *models.User, canvasID uuid.UUID) (canvas *models.Canvas, err error) {
	ctx, done := telemetry.Span(ctx, "repository.find_canvas")
	defer done(&err)

	return models.FindCanvasInTransaction(db, user.OrganizationID, canvasID)
}

func setInlineFileHeaders(w http.ResponseWriter, path, contentType string) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": filepath.Base(path),
	}))
}
