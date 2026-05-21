package public

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapps"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type installAppRequest struct {
	Repo           string `json:"repo"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
}

type installAppResponse struct {
	CanvasID       string `json:"canvasId"`
	OrganizationID string `json:"organizationId"`
}

func (s *Server) appInstallPreview(w http.ResponseWriter, r *http.Request) {
	repo := strings.TrimSpace(r.URL.Query().Get("repo"))
	if repo == "" {
		http.Error(w, "repo query parameter is required", http.StatusBadRequest)
		return
	}

	preview, err := githubapps.BuildPreview(repo)
	if err != nil {
		writeAppInstallError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(preview); err != nil {
		log.Errorf("failed to encode app install preview: %v", err)
	}
}

func (s *Server) installApp(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetEffectiveAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var req installAppRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	repo, err := githubapps.ParseRepository(req.Repo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	organizationID, err := uuid.Parse(strings.TrimSpace(req.OrganizationID))
	if err != nil {
		http.Error(w, "organizationId is invalid", http.StatusBadRequest)
		return
	}

	user, err := findActiveUserForAccountInOrganization(account.ID, organizationID)
	if err != nil {
		writeAppInstallError(w, err)
		return
	}

	allowed, err := s.authService.CheckOrganizationPermission(
		user.ID.String(),
		organizationID.String(),
		"canvases",
		"create",
	)
	if err != nil {
		log.Errorf("failed to check canvas create permission: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "You do not have permission to create apps in this organization", http.StatusForbidden)
		return
	}

	canvas, _, err := githubapps.FetchCanvas(repo)
	if err != nil {
		writeAppInstallError(w, err)
		return
	}

	canvas.Metadata.Name = name

	ctx := authentication.SetUserIdInMetadata(r.Context(), user.ID.String())
	response, err := canvases.CreateCanvas(
		ctx,
		s.registry,
		s.encryptor,
		s.authService,
		s.WebhooksBaseURL,
		organizationID,
		canvas,
		nil,
		s.usageService,
	)
	if err != nil {
		writeAppInstallError(w, err)
		return
	}

	canvasID := ""
	if response != nil && response.Canvas != nil && response.Canvas.Metadata != nil {
		canvasID = response.Canvas.Metadata.Id
	}

	if canvasID == "" {
		http.Error(w, "Failed to install app", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(installAppResponse{
		CanvasID:       canvasID,
		OrganizationID: organizationID.String(),
	}); err != nil {
		log.Errorf("failed to encode app install response: %v", err)
	}
}

func findActiveUserForAccountInOrganization(accountID, organizationID uuid.UUID) (*models.User, error) {
	account, err := models.FindAccountByID(accountID.String())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "account not found")
	}

	user, err := models.FindMaybeDeletedUserByEmailInTransaction(database.Conn(), organizationID.String(), account.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.PermissionDenied, "you are not a member of this organization")
		}

		return nil, status.Error(codes.Internal, "failed to resolve organization membership")
	}

	if user.DeletedAt.Valid {
		return nil, status.Error(codes.PermissionDenied, "you are not a member of this organization")
	}

	return user, nil
}

func writeAppInstallError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.InvalidArgument:
			http.Error(w, st.Message(), http.StatusBadRequest)
		case codes.Unauthenticated:
			http.Error(w, st.Message(), http.StatusUnauthorized)
		case codes.PermissionDenied:
			http.Error(w, st.Message(), http.StatusForbidden)
		case codes.AlreadyExists:
			http.Error(w, st.Message(), http.StatusConflict)
		case codes.ResourceExhausted:
			http.Error(w, st.Message(), http.StatusTooManyRequests)
		default:
			http.Error(w, st.Message(), http.StatusInternalServerError)
		}

		return
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "required"),
		strings.Contains(message, "expected github.com"),
		strings.Contains(message, "parse canvas"),
		strings.Contains(message, "template"):
		http.Error(w, message, http.StatusBadRequest)
	case strings.Contains(message, "not found"):
		http.Error(w, message, http.StatusNotFound)
	default:
		log.Errorf("app install error: %v", err)
		http.Error(w, message, http.StatusBadRequest)
	}
}
