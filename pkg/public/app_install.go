package public

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/installation"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type installAppRequest struct {
	Repo           string                                     `json:"repo"`
	OrganizationID string                                     `json:"organizationId"`
	Name           string                                     `json:"name"`
	InstallParams  map[string]string                          `json:"installParams,omitempty"`
	Integrations   map[string]installation.IntegrationMapping `json:"integrations,omitempty"`
}

type installAppResponse struct {
	CanvasID       string `json:"canvasId"`
	OrganizationID string `json:"organizationId"`
}

func (s *Server) installationService() *installation.Service {
	return &installation.Service{
		Registry:        s.registry,
		Encryptor:       s.encryptor,
		AuthService:     s.authService,
		WebhooksBaseURL: s.WebhooksBaseURL,
		GitProvider:     s.gitProvider,
		UsageService:    s.usageService,
	}
}

func (s *Server) appInstallPreview(w http.ResponseWriter, r *http.Request) {
	preview, err := s.installationService().Preview(r.URL.Query().Get("repo"))
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var req installAppRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	organizationID, err := uuid.Parse(strings.TrimSpace(req.OrganizationID))
	if err != nil {
		http.Error(w, "organizationId is invalid", http.StatusBadRequest)
		return
	}

	result, err := s.installationService().Install(r.Context(), installation.InstallRequest{
		Repo:           req.Repo,
		Name:           req.Name,
		OrganizationID: organizationID,
		AccountID:      account.ID,
		InstallParams:  req.InstallParams,
		Integrations:   req.Integrations,
	})
	if err != nil {
		writeAppInstallError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(installAppResponse{
		CanvasID:       result.CanvasID,
		OrganizationID: result.OrganizationID,
	}); err != nil {
		log.Errorf("failed to encode app install response: %v", err)
	}
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
