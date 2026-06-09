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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) handleRepositoryFileDownload(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["canvas_id"]
	path := r.URL.Query().Get("path")
	versionID := strings.TrimSpace(r.URL.Query().Get("version_id"))

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
		content, readErr := canvases.ReadRepositorySpecFile(
			ctx,
			user.OrganizationID.String(),
			canvas.ID.String(),
			versionID,
			path,
		)
		if readErr != nil {
			writeRepositorySpecFileError(w, readErr, path, canvasID.String())
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

	repository, err := models.FindRepository(user.OrganizationID, canvas.ID)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	reader, err := s.gitProvider.GetFile(r.Context(), repository.RepoID, path)
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

// writeRepositorySpecFileError maps a gRPC error returned by
// ReadRepositorySpecFile to an HTTP response. Client-caused errors (NotFound,
// PermissionDenied, InvalidArgument, Unauthenticated) become 4xx so they are
// not reported as server errors to Sentry, and only true server errors are
// logged at error level.
func writeRepositorySpecFileError(w http.ResponseWriter, err error, path, canvasID string) {
	code := status.Code(err)
	httpStatus, message := repositorySpecFileHTTPStatus(code, err)

	if httpStatus >= http.StatusInternalServerError {
		log.Errorf("Failed to read repository spec file %s in canvas %s: %v", path, canvasID, err)
	} else {
		log.Debugf("Repository spec file %s in canvas %s not served: %v", path, canvasID, err)
	}

	http.Error(w, message, httpStatus)
}

func repositorySpecFileHTTPStatus(code codes.Code, err error) (int, string) {
	switch code {
	case codes.NotFound:
		return http.StatusNotFound, statusMessageOrDefault(err, "File not found")
	case codes.PermissionDenied:
		return http.StatusForbidden, statusMessageOrDefault(err, "Forbidden")
	case codes.Unauthenticated:
		return http.StatusUnauthorized, statusMessageOrDefault(err, "Unauthenticated")
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest, statusMessageOrDefault(err, "Invalid request")
	default:
		return http.StatusInternalServerError, "Failed to get file"
	}
}

func statusMessageOrDefault(err error, fallback string) string {
	message := strings.TrimSpace(status.Convert(err).Message())
	if message == "" {
		return fallback
	}
	return message
}
