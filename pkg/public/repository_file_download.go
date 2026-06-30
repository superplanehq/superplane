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
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	errRepositoryFileNotFound   = errors.New("repository file not found")
	errRepositoryNotFound       = errors.New("repository not found")
	errRepositoryFileReadFailed = errors.New("repository file read failed")
)

func (s *Server) handleRepositoryFileDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return
	}

	allowed, err := s.checkRepositoryReadPermission(ctx, user)
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

	canvas, err := s.findRepositoryCanvas(ctx, user, canvasID)
	if err != nil {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}

	err = s.readRepositoryFile(ctx, w, user, canvas, canvasID, path, versionID, stage)
	if errors.Is(err, errRepositoryFileNotFound) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, errRepositoryNotFound) {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}
}

func (s *Server) writeRepositorySpecFile(
	ctx context.Context,
	w http.ResponseWriter,
	user *models.User,
	canvas *models.Canvas,
	canvasID uuid.UUID,
	path string,
	versionID string,
	stage bool,
) error {
	authCtx := authentication.SetUserIdInMetadata(ctx, user.ID.String())
	readSpecFile := canvases.ReadRepositorySpecFile
	if stage {
		readSpecFile = canvases.ReadRepositorySpecFileStaged
	}
	content, readErr := readSpecFile(
		authCtx,
		user.OrganizationID.String(),
		canvas.ID.String(),
		versionID,
		path,
	)
	if readErr != nil {
		log.Errorf("Failed to read repository spec file %s in canvas %s: %v", path, canvasID.String(), readErr)
		return errRepositoryFileReadFailed
	}

	setInlineFileHeaders(w, path, "text/yaml; charset=utf-8")
	if _, writeErr := io.WriteString(w, content); writeErr != nil {
		log.Errorf("Failed to write repository spec file %s in canvas %s: %v", path, canvasID.String(), writeErr)
		return writeErr
	}

	return nil
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

func (s *Server) findRepositoryCanvas(ctx context.Context, user *models.User, canvasID uuid.UUID) (canvas *models.Canvas, err error) {
	ctx, done := telemetry.Span(ctx, "repository.find_canvas")
	defer done(&err)

	return models.FindCanvasInTransaction(database.DB(ctx), user.OrganizationID, canvasID)
}

func (s *Server) readRepositoryFile(
	ctx context.Context,
	w http.ResponseWriter,
	user *models.User,
	canvas *models.Canvas,
	canvasID uuid.UUID,
	path string,
	versionID string,
	stage bool,
) (err error) {
	ctx, done := telemetry.Span(ctx, "repository.read_file")
	defer done(&err)

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.String("repository.file_path", path),
			attribute.Bool("repository.staged", stage),
		)
		if versionID != "" {
			span.SetAttributes(attribute.String("repository.version_id", versionID))
		}
	}

	if canvases.IsRepositorySpecFilePath(path) {
		return s.writeRepositorySpecFile(ctx, w, user, canvas, canvasID, path, versionID, stage)
	}

	if stage && versionID != "" {
		written, readErr := s.tryWriteStagedRepositoryFile(ctx, w, user, canvas, canvasID, path, versionID)
		if readErr != nil {
			return readErr
		}
		if written {
			return nil
		}
	}

	return s.writeRepositoryGitFile(ctx, w, user.OrganizationID, canvas, canvasID, path)
}

func (s *Server) tryWriteStagedRepositoryFile(
	ctx context.Context,
	w http.ResponseWriter,
	user *models.User,
	canvas *models.Canvas,
	canvasID uuid.UUID,
	path string,
	versionID string,
) (bool, error) {
	authCtx := authentication.SetUserIdInMetadata(ctx, user.ID.String())
	content, found, deleted, stagedErr := canvases.ReadStagedRepositoryFile(
		authCtx,
		user.OrganizationID.String(),
		canvas.ID.String(),
		versionID,
		"",
		path,
	)
	if stagedErr != nil {
		log.Errorf("Failed to read staged repository file %s in canvas %s: %v", path, canvasID.String(), stagedErr)
		return false, errRepositoryFileReadFailed
	}
	if deleted {
		return false, errRepositoryFileNotFound
	}
	if !found {
		return false, nil
	}

	setInlineFileHeaders(w, path, "application/octet-stream")
	if _, writeErr := io.WriteString(w, content); writeErr != nil {
		log.Errorf("Failed to write staged repository file %s in canvas %s: %v", path, canvasID.String(), writeErr)
		return false, writeErr
	}

	return true, nil
}

func (s *Server) writeRepositoryGitFile(
	ctx context.Context,
	w http.ResponseWriter,
	organizationID uuid.UUID,
	canvas *models.Canvas,
	canvasID uuid.UUID,
	path string,
) error {
	repository, repoErr := models.FindRepository(organizationID, canvas.ID)
	if repoErr != nil {
		return errRepositoryNotFound
	}

	reader, fileErr := s.gitProvider.GetFile(ctx, repository.RepoID, path, "")
	if fileErr != nil {
		log.Errorf("Failed to get file %s in canvas %s: %v", path, canvasID.String(), fileErr)
		return errRepositoryFileReadFailed
	}

	defer reader.Close()

	setInlineFileHeaders(w, path, "application/octet-stream")
	if _, copyErr := io.Copy(w, reader); copyErr != nil {
		log.Errorf("Failed to copy file %s in canvas %s: %v", path, canvasID.String(), copyErr)
		return copyErr
	}

	return nil
}

func setInlineFileHeaders(w http.ResponseWriter, path, contentType string) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": filepath.Base(path),
	}))
}
