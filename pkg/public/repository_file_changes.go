package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

type repositoryFileChangesResponse struct {
	HasUnpublishedFileChanges bool     `json:"hasUnpublishedFileChanges"`
	ChangedPaths              []string `json:"changedPaths"`
}

// handleRepositoryFileChanges reports whether a draft version has committed changes
// to arbitrary repository files relative to live. The spec-based graph/console
// diffs do not cover files such as README.md, so the UI needs this signal to enable
// Publish and light the Files tab's unpublished-changes indicator.
func (s *Server) handleRepositoryFileChanges(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["canvas_id"]
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

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return
	}

	allowed, err := s.authService.CheckOrganizationPermission(
		r.Context(),
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
		log.Warnf("User %s is not authorized to read canvas %s", user.ID.String(), canvasID.String())
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
		// No repository means there are no files to diff against live.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeRepositoryFileChanges(w, nil)
			return
		}
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	ctx := authentication.SetUserIdInMetadata(r.Context(), user.ID.String())
	changedPaths, err := canvases.ListUnpublishedRepositoryFileChanges(
		ctx,
		s.gitProvider,
		repository.RepoID,
		user.OrganizationID.String(),
		canvas.ID.String(),
		versionID,
	)
	if err != nil {
		log.Errorf("Failed to compute repository file changes for canvas %s: %v", canvasID.String(), err)
		http.Error(w, "Failed to compute repository file changes", http.StatusInternalServerError)
		return
	}

	writeRepositoryFileChanges(w, changedPaths)
}

func writeRepositoryFileChanges(w http.ResponseWriter, changedPaths []string) {
	w.Header().Set("Content-Type", "application/json")
	response := repositoryFileChangesResponse{
		HasUnpublishedFileChanges: len(changedPaths) > 0,
		ChangedPaths:              changedPaths,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Failed to encode repository file changes response: %v", err)
	}
}
