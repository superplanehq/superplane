package public

import (
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

type githubAppManifestResponse struct {
	URL    string            `json:"url"`
	Method string            `json:"method"`
	Form   map[string]string `json:"form"`
}

func (s *Server) HandleGitHubAppManifest(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.GetAccountFromContext(r.Context()); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if config.LoadGitHubHostedAppConfig().Enabled() {
		http.Error(w, "GitHub App is already configured", http.StatusConflict)
		return
	}
	if githubapp.Enabled(r.Context(), database.DB(r.Context()), s.encryptor) {
		http.Error(w, "GitHub App is already configured", http.StatusConflict)
		return
	}

	baseURL := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	webhooksBaseURL := strings.TrimRight(strings.TrimSpace(s.WebhooksBaseURL), "/")
	if webhooksBaseURL == "" {
		webhooksBaseURL = baseURL
	}
	manifest, err := githubapp.PublicManifestJSON(baseURL, webhooksBaseURL)
	if err != nil {
		http.Error(w, "GitHub App setup is not available", http.StatusInternalServerError)
		return
	}
	state, err := githubapp.SignCreateState(s.jwt.Secret, r.URL.Query().Get("return_to"))
	if err != nil {
		http.Error(w, "failed to start GitHub App setup", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(githubAppManifestResponse{
		URL:    githubapp.CreateURL(),
		Method: http.MethodPost,
		Form: map[string]string{
			"manifest": manifest,
			"state":    state,
		},
	})
}

func (s *Server) HandleGitHubAppCreated(w http.ResponseWriter, r *http.Request) {
	returnPath, err := githubapp.VerifyCreateState(s.jwt.Secret, r.URL.Query().Get("state"))
	if err != nil {
		http.Error(w, "invalid GitHub App setup state", http.StatusBadRequest)
		return
	}
	if config.LoadGitHubHostedAppConfig().Enabled() || githubapp.Enabled(r.Context(), database.DB(r.Context()), s.encryptor) {
		http.Redirect(w, r, returnPath, http.StatusFound)
		return
	}

	cfg, err := githubapp.ConvertManifest(s.registry.HTTPContext(), r.URL.Query().Get("code"))
	if err != nil {
		log.WithError(err).Error("failed to convert GitHub App manifest")
		http.Error(w, "failed to create GitHub App", http.StatusBadGateway)
		return
	}
	if err := githubapp.Save(r.Context(), database.DB(r.Context()), s.encryptor, cfg); err != nil {
		log.WithError(err).Error("failed to store GitHub App credentials")
		http.Error(w, "failed to save GitHub App", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, returnPath, http.StatusFound)
}
