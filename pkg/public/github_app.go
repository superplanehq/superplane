package public

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"

	gh "github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
)

// HandleGitHubAppSetup finishes a public SuperPlane GitHub App install.
// GitHub sends every install to this one Setup URL. The CSRF state finds
// the pending SuperPlane connection.
func (s *Server) HandleGitHubAppSetup(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	integration, err := models.FindGitHubIntegrationByAppState(database.DB(r.Context()), state)
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	s.dispatchIntegrationRequest(w, r, integration)
}

// HandleGitHubAppWebhook receives installation events for the public
// SuperPlane GitHub App. One GitHub install can map to more than one
// SuperPlane connection.
func (s *Server) HandleGitHubAppWebhook(w http.ResponseWriter, r *http.Request) {
	app, ok := common.HostedAppFromEnv()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	payload, err := gh.ValidatePayload(r, []byte(app.WebhookSecret))
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	eventType := gh.WebHookType(r)
	event, err := gh.ParseWebHook(eventType, payload)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	installationID, ok := githubInstallationID(event)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	integrations, err := models.ListGitHubIntegrationsByInstallationID(database.DB(r.Context()), installationID)
	if err != nil {
		log.WithError(err).Error("failed to list GitHub App integrations")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	for i := range integrations {
		cloned, err := cloneRequestWithBody(r, payload)
		if err != nil {
			log.WithError(err).Error("failed to clone GitHub App webhook request")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		s.dispatchIntegrationRequest(httptest.NewRecorder(), cloned, &integrations[i])
	}

	w.WriteHeader(http.StatusOK)
}

func githubInstallationID(event any) (string, bool) {
	switch event := event.(type) {
	case *gh.InstallationEvent:
		if event.GetInstallation() == nil {
			return "", false
		}
		return strconv.FormatInt(event.GetInstallation().GetID(), 10), true
	case *gh.InstallationRepositoriesEvent:
		if event.GetInstallation() == nil {
			return "", false
		}
		return strconv.FormatInt(event.GetInstallation().GetID(), 10), true
	default:
		return "", false
	}
}

func cloneRequestWithBody(r *http.Request, body []byte) (*http.Request, error) {
	cloned := r.Clone(r.Context())
	cloned.Body = io.NopCloser(bytes.NewReader(body))
	cloned.ContentLength = int64(len(body))
	return cloned, nil
}
