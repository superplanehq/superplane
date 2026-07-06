package public

import (
	"context"
	"errors"
	"fmt"
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

	reader, err := s.findFileReader(ctx, db, r, user, canvas, path)
	if err != nil {
		if errors.Is(err, files.ErrFileNotFound) || errors.Is(err, files.ErrFileDeleted) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	defer reader.Close()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": filepath.Base(path),
	}))

	if _, err := io.Copy(w, reader); err != nil {
		log.Errorf("Failed to copy file: %v", err)
		http.Error(w, "Failed to copy file", http.StatusInternalServerError)
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

func (s *Server) findFileReader(ctx context.Context, db *gorm.DB, r *http.Request, user *models.User, canvas *models.Canvas, path string) (reader io.ReadCloser, err error) {
	ctx, done := telemetry.Span(ctx, "repository.find_reader")
	defer done(&err)

	version := strings.TrimSpace(r.URL.Query().Get("version_id"))
	stage := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("stage")), "true")
	appFileReader := files.NewAppFileReader(db, s.gitProvider, canvas, user.ID)

	switch {

	//
	// If version_id is provided, we read the file for a specific version.
	//
	case version != "":
		versionID, err := uuid.Parse(version)
		if err != nil {
			return nil, files.ErrFileNotFound
		}

		return appFileReader.ReadFromVersion(ctx, path, versionID)

	//
	// If stage=true is provided, we read the file from the user's staging area,
	// if it's there, or from the live version if it's not.
	//
	case stage:
		return appFileReader.Read(ctx, path)

	//
	// Otherwise, we read from the live version.
	//
	default:
		version, err := models.FindCanvasVersionInTransaction(db, canvas.ID, *canvas.LiveVersionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, files.ErrFileNotFound
			}

			return nil, fmt.Errorf("failed to find canvas version: %w", err)
		}

		return appFileReader.ReadFromVersion(ctx, path, version.ID)
	}
}

func (s *Server) findRepositoryCanvas(ctx context.Context, db *gorm.DB, user *models.User, canvasID uuid.UUID) (canvas *models.Canvas, err error) {
	ctx, done := telemetry.Span(ctx, "repository.find_canvas")
	defer done(&err)

	return models.FindCanvasInTransaction(db, user.OrganizationID, canvasID)
}
