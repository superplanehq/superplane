package public

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
)

type adminFactoryTemplatesResponse struct {
	Templates []factoryactions.OnboardingFactoryTemplate `json:"templates"`
}

func (s *Server) factoryWebhookBaseURL() string {
	webhookURL := strings.TrimSpace(s.WebhooksBaseURL)
	if webhookURL == "" {
		webhookURL = strings.TrimSpace(s.BaseURL)
	}
	return webhookURL
}

func (s *Server) factoryIntakeDependencies() factoryactions.IntakeDependencies {
	return factoryactions.IntakeDependencies{
		Registry:       s.registry,
		Encryptor:      s.encryptor,
		AuthService:    s.authService,
		WebhookBaseURL: s.factoryWebhookBaseURL(),
	}
}

// adminListFactoryTemplates lists onboarding factory templates on this
// installation.
func (s *Server) adminListFactoryTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := factoryactions.ListOnboardingFactoryTemplates(r.Context())
	if err != nil {
		log.Errorf("admin: failed to list factory templates: %v", err)
		http.Error(w, "Failed to list factory templates", http.StatusInternalServerError)
		return
	}

	respondJSON(w, adminFactoryTemplatesResponse{Templates: templates})
}

// adminResetFactoryTemplate replaces every matching onboarding automation on
// this installation with the current SuperPlane defaults.
func (s *Server) adminResetFactoryTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := mux.Vars(r)["templateId"]
	result, err := factoryactions.ResetOnboardingFactoryTemplate(
		r.Context(),
		s.factoryIntakeDependencies(),
		templateID,
	)
	if err != nil {
		if errors.Is(err, factoryactions.ErrUnknownOnboardingFactoryTemplate) {
			http.Error(w, "Factory template not found", http.StatusNotFound)
			return
		}
		log.Errorf("admin: failed to reset factory template %s: %v", templateID, err)
		http.Error(w, "Failed to reset factory template", http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}
