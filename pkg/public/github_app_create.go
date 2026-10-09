package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

type githubAppManifestResponse struct {
	URL    string            `json:"url"`
	Method string            `json:"method"`
	Form   map[string]string `json:"form"`
}

func (s *Server) HandleGitHubAppManifest(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !account.IsInstallationAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if s.githubAppCreateBlocked(r) {
		if githubapp.UserConnectReady(r.Context()) {
			http.Error(w, "GitHub App is already configured", http.StatusConflict)
			return
		}
		http.Error(w, "add the client id and client secret for the existing GitHub App", http.StatusConflict)
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
	state, err := githubapp.SignCreateState(s.jwt.Secret, r.URL.Query().Get("return_to"), account.ID)
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
	returnPath, accountID, err := githubapp.VerifyCreateState(s.jwt.Secret, r.URL.Query().Get("state"))
	if err != nil {
		http.Error(w, "invalid GitHub App setup state", http.StatusBadRequest)
		return
	}
	account, err := models.FindAccountByID(accountID.String())
	if err != nil || !account.IsInstallationAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if s.githubAppCreateBlocked(r) {
		s.redirectExistingGitHubApp(w, r, returnPath)
		return
	}

	cfg, err := githubapp.ConvertManifest(s.registry.HTTPContext(), r.URL.Query().Get("code"))
	if err != nil {
		log.WithError(err).Error("failed to convert GitHub App manifest")
		http.Error(w, "failed to create GitHub App", http.StatusBadGateway)
		return
	}
	if err := githubapp.Save(r.Context(), database.DB(r.Context()), s.encryptor, cfg); err != nil {
		if errors.Is(err, githubapp.ErrAlreadyConfigured) {
			s.redirectExistingGitHubApp(w, r, returnPath)
			return
		}
		log.WithError(err).Error("failed to store GitHub App credentials")
		http.Error(w, "failed to save GitHub App", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, githubAccountConnectURL(returnPath), http.StatusFound)
}

func (s *Server) githubAppCreateBlocked(r *http.Request) bool {
	if config.LoadGitHubHostedAppConfig().Enabled() {
		return true
	}
	cfg, err := githubapp.Resolve(r.Context(), database.DB(r.Context()), s.encryptor)
	return err == nil && cfg.Enabled()
}

func (s *Server) redirectExistingGitHubApp(w http.ResponseWriter, r *http.Request, returnPath string) {
	if githubapp.UserConnectReady(r.Context()) {
		http.Redirect(w, r, githubAccountConnectURL(returnPath), http.StatusFound)
		return
	}
	http.Redirect(w, r, githubLoginClientURL(returnPath), http.StatusFound)
}

type githubAppLoginResponse struct {
	State     string `json:"state"`
	Slug      string `json:"slug,omitempty"`
	CanUpdate bool   `json:"canUpdate,omitempty"`
}

type githubAppLoginRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func (s *Server) HandleGitHubAppLogin(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !account.IsInstallationAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		status, err := githubapp.LoginClientStatus(database.DB(r.Context()))
		if err != nil {
			log.WithError(err).Error("failed to read GitHub login client")
			http.Error(w, "failed to read GitHub login", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(githubAppLoginResponse{
			State:     status.State,
			Slug:      status.Slug,
			CanUpdate: status.CanUpdate,
		})
	case http.MethodPost:
		var body githubAppLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid GitHub login client", http.StatusBadRequest)
			return
		}
		err := githubapp.SaveLoginClient(r.Context(), database.DB(r.Context()), s.encryptor, body.ClientID, body.ClientSecret)
		if errors.Is(err, githubapp.ErrAlreadyConfigured) {
			http.Error(w, "GitHub App is already configured", http.StatusConflict)
			return
		}
		if errors.Is(err, githubapp.ErrGitHubAppMissing) {
			http.Error(w, "create the GitHub App before adding its login client", http.StatusConflict)
			return
		}
		if err != nil {
			log.WithError(err).Error("failed to store GitHub login client")
			http.Error(w, "failed to save GitHub login", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func githubLoginClientURL(returnPath string) string {
	target, err := url.Parse(returnPath)
	if err != nil || target.Path == "" || !strings.HasPrefix(target.Path, "/") {
		target = &url.URL{Path: "/"}
	}
	query := target.Query()
	query.Set("githubLogin", "needs_client")
	target.RawQuery = query.Encode()
	return target.String()
}

func githubAccountConnectURL(returnPath string) string {
	target, err := url.Parse(returnPath)
	if err != nil || target.Path == "" || !strings.HasPrefix(target.Path, "/") {
		target = &url.URL{Path: "/"}
	}
	query := target.Query()
	if query.Get("githubConnected") == "" {
		query.Set("githubConnected", "1")
	}
	target.RawQuery = query.Encode()
	values := url.Values{}
	values.Set("intent", "connect")
	values.Set("redirect", target.String())
	return "/auth/github?" + values.Encode()
}
