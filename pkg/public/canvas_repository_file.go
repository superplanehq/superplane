package public

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"slices"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/git"
	canvasActions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const canvasRepositoryFileHTTPPath = "/api/v1/canvases/{canvas_id}/repository/file/{path=**}"

func (s *Server) handleCanvasRepositoryFile(w http.ResponseWriter, r *http.Request) {
	setOtelMetricRoute(r.Context(), canvasRepositoryFileHTTPPath)

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	canvasID := vars["canvas_id"]
	if canvasID == "" {
		http.Error(w, "canvas_id is required", http.StatusBadRequest)
		return
	}

	allowed, err := s.authService.CheckOrganizationPermission(
		user.ID.String(),
		user.OrganizationID.String(),
		"canvases",
		"read",
	)
	if err != nil {
		writeCanvasRepositoryFileError(w, err)
		return
	}

	if !allowed || !hasCanvasRepositoryFileScopedPermission(r, canvasID) {
		http.NotFound(w, r)
		return
	}

	file, err := canvasActions.OpenCanvasRepositoryFile(
		r.Context(),
		user.OrganizationID.String(),
		canvasID,
		vars["path"],
		r.URL.Query().Get("ref"),
		s.canvasStorage,
		s.canvasStorageOptions,
	)
	if err != nil {
		writeCanvasRepositoryFileError(w, err)
		return
	}
	defer file.Content.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": filepath.Base(file.Path),
	}))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if _, err := io.Copy(w, file.Content); err != nil && !errors.Is(err, git.ErrFileTooLarge) {
		return
	}
}

func hasCanvasRepositoryFileScopedPermission(r *http.Request, canvasID string) bool {
	claims, ok := middleware.GetScopedTokenClaimsFromContext(r.Context())
	if !ok {
		return true
	}

	for _, permission := range jwt.PermissionsFromScopes(claims.Scopes) {
		if permission.ResourceType != "canvases" || permission.Action != "read" {
			continue
		}

		if len(permission.Resources) == 0 || slices.Contains(permission.Resources, canvasID) {
			return true
		}
	}

	return false
}

func writeCanvasRepositoryFileError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Error(w, st.Message(), canvasRepositoryFileHTTPStatus(st.Code()))
}

func canvasRepositoryFileHTTPStatus(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.ResourceExhausted:
		return http.StatusRequestEntityTooLarge
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
