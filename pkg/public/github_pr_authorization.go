package public

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

const (
	githubPRAuthorizationIntent       = "github_pr_authorization"
	githubPRAuthorizationStatePrefix  = "github-pr:"
	githubPRAuthorizationTTL          = 10 * time.Minute
	githubPRAuthorizationCookie       = "github_pr_authorization_verifier"
	githubPRAuthorizationCookiePath   = "/account/github-authorization"
	githubPRAuthorizationResultParam  = "github_pr_authorization"
	githubPRAuthorizationErrorParam   = "auth_error"
	githubPRAuthorizationErrorDenied  = "github_pr_authorization_denied"
	githubPRAuthorizationErrorFailed  = "github_pr_authorization_failed"
	githubPRAuthorizationErrorInUse   = "linked_account_in_use"
	githubPRAuthorizationProviderName = "provider"
)

// AccountGitHubPRAuthorizationResponse tells the profile whether the member
// opens pull requests with their own GitHub account.
type AccountGitHubPRAuthorizationResponse struct {
	Available            bool   `json:"available"`
	Enabled              bool   `json:"enabled"`
	Login                string `json:"login,omitempty"`
	NeedsReauthorization bool   `json:"needs_reauthorization"`
}

type githubPRAuthorizationState struct {
	AccountID uuid.UUID
	Redirect  string
	Challenge string
}

func (s *Server) accountGitHubPRAuthorization(r *http.Request, accountID uuid.UUID) (AccountGitHubPRAuthorizationResponse, error) {
	cfg, err := githubapp.Resolve(r.Context(), database.DB(r.Context()), s.encryptor)
	if err != nil {
		return AccountGitHubPRAuthorizationResponse{}, err
	}
	response := AccountGitHubPRAuthorizationResponse{Available: cfg.UserAuthorizationEnabled()}

	authorization, err := models.FindAccountGitHubAuthorization(database.DB(r.Context()), accountID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response, nil
	}
	if err != nil {
		return AccountGitHubPRAuthorizationResponse{}, err
	}

	response.Login = authorization.GitHubLogin
	if response.Available && authorization.UsableFor(cfg.ID) {
		response.Enabled = true
		return response, nil
	}
	response.NeedsReauthorization = true
	return response, nil
}

// startGitHubPRAuthorization sends the member to GitHub to authorize the
// public SuperPlane GitHub App for pull requests.
func (s *Server) startGitHubPRAuthorization(w http.ResponseWriter, r *http.Request) {
	if rejectInSupportSession(w, r) {
		return
	}
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cfg, err := githubapp.Resolve(r.Context(), database.DB(r.Context()), s.encryptor)
	if err != nil || !cfg.UserAuthorizationEnabled() {
		http.Error(w, "GitHub authorization for pull requests is not configured", http.StatusServiceUnavailable)
		return
	}

	verifier, err := githubapp.NewPKCEVerifier()
	if err != nil {
		http.Error(w, "Failed to start GitHub authorization", http.StatusInternalServerError)
		return
	}
	redirect := githubapp.SafeReturnPath(r.URL.Query().Get("redirect"))
	state, err := s.signGitHubPRAuthorizationState(account.ID, redirect, githubapp.PKCEChallenge(verifier))
	if err != nil {
		http.Error(w, "Failed to start GitHub authorization", http.StatusInternalServerError)
		return
	}
	authorizeURL, err := githubapp.UserAuthorizationURL(cfg, s.githubPRAuthorizationCallbackURL(), state, verifier)
	if err != nil {
		http.Error(w, "Failed to start GitHub authorization", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, s.githubPRAuthorizationCookie(verifier, int(githubPRAuthorizationTTL.Seconds())))
	http.Redirect(w, r, authorizeURL, http.StatusFound)
}

// completeGitHubPRAuthorization stores the user token and links the GitHub
// identity. The signed state, the session account, and the PKCE cookie must
// all match.
func (s *Server) completeGitHubPRAuthorization(w http.ResponseWriter, r *http.Request) {
	if rejectInSupportSession(w, r) {
		return
	}
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	state, err := s.parseGitHubPRAuthorizationState(r.URL.Query().Get("state"))
	if err != nil {
		http.Error(w, "Invalid GitHub authorization state", http.StatusBadRequest)
		return
	}
	if state.AccountID != account.ID {
		http.Error(w, "GitHub authorization cannot change another account", http.StatusForbidden)
		return
	}

	verifierCookie, err := r.Cookie(githubPRAuthorizationCookie)
	http.SetCookie(w, s.githubPRAuthorizationCookie("", -1))
	if err != nil || githubapp.PKCEChallenge(verifierCookie.Value) != state.Challenge {
		http.Error(w, "Invalid GitHub authorization state", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("error") != "" {
		http.Redirect(w, r, githubPRAuthorizationRedirect(state.Redirect, githubPRAuthorizationErrorParam, githubPRAuthorizationErrorDenied), http.StatusSeeOther)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Error(w, "GitHub returned no authorization code", http.StatusBadRequest)
		return
	}

	err = s.saveGitHubPRAuthorization(r, account.ID, code, verifierCookie.Value)
	if errors.Is(err, models.ErrLinkedAccountInUse) {
		http.Redirect(w, r, githubPRAuthorizationRedirect(state.Redirect, githubPRAuthorizationErrorParam, githubPRAuthorizationErrorInUse), http.StatusSeeOther)
		return
	}
	if err != nil {
		log.WithError(err).Errorf("Error saving GitHub pull request authorization for %s", account.ID)
		http.Redirect(w, r, githubPRAuthorizationRedirect(state.Redirect, githubPRAuthorizationErrorParam, githubPRAuthorizationErrorFailed), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, githubPRAuthorizationRedirect(state.Redirect, githubPRAuthorizationResultParam, "enabled"), http.StatusSeeOther)
}

// disableGitHubPRAuthorization removes the grant. SuperPlane then opens pull
// requests for this member as the app.
func (s *Server) disableGitHubPRAuthorization(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if hasActiveSupportSession(r) {
		http.Error(w, "GitHub authorization cannot change in a read-only support session", http.StatusForbidden)
		return
	}

	cfg, err := githubapp.Resolve(r.Context(), database.DB(r.Context()), s.encryptor)
	if err != nil {
		cfg = config.GitHubHostedAppConfig{}
	}

	revokeErr, err := githubapp.DeleteUserAuthorization(
		r.Context(),
		database.DB(r.Context()),
		s.encryptor,
		s.registry.HTTPContext(),
		cfg,
		account.ID,
	)
	if err != nil {
		log.WithError(err).Errorf("Error removing GitHub pull request authorization for %s", account.ID)
		http.Error(w, "Failed to turn off GitHub authorization", http.StatusInternalServerError)
		return
	}
	if revokeErr != nil {
		log.WithError(revokeErr).Warnf("Could not revoke GitHub user token for %s", account.ID)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) saveGitHubPRAuthorization(r *http.Request, accountID uuid.UUID, code, verifier string) error {
	ctx := r.Context()
	cfg, err := githubapp.Resolve(ctx, database.DB(ctx), s.encryptor)
	if err != nil {
		return err
	}
	httpCtx := s.registry.HTTPContext()

	tokens, err := githubapp.ExchangeUserCode(httpCtx, cfg, code, verifier, s.githubPRAuthorizationCallbackURL())
	if err != nil {
		return err
	}
	user, err := githubapp.FetchGitHubUser(httpCtx, tokens.AccessToken)
	if err != nil {
		return err
	}

	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		linked := models.NewAccountLinkedAccount(accountID, models.ProviderGitHub, user.UserIDString(), user.Login, user.Name, user.AvatarURL)
		if err := models.SaveAccountLinkedAccount(tx, linked); err != nil {
			return err
		}
		return githubapp.SaveUserAuthorization(ctx, tx, s.encryptor, accountID, cfg.ID, user, tokens)
	})
}

func (s *Server) signGitHubPRAuthorizationState(accountID uuid.UUID, redirect, challenge string) (string, error) {
	token, err := s.jwt.GenerateWithClaims(githubPRAuthorizationTTL, map[string]string{
		"sub":       accountID.String(),
		"intent":    githubPRAuthorizationIntent,
		"redirect":  redirect,
		"challenge": challenge,
		"jti":       uuid.NewString(),
	})
	if err != nil {
		return "", err
	}
	return githubPRAuthorizationStatePrefix + token, nil
}

func (s *Server) parseGitHubPRAuthorizationState(state string) (*githubPRAuthorizationState, error) {
	token, ok := strings.CutPrefix(strings.TrimSpace(state), githubPRAuthorizationStatePrefix)
	if !ok {
		return nil, errors.New("not a GitHub authorization state")
	}
	claims, err := s.jwt.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}

	intent, _ := claims["intent"].(string)
	subject, _ := claims["sub"].(string)
	redirect, _ := claims["redirect"].(string)
	challenge, _ := claims["challenge"].(string)
	nonce, _ := claims["jti"].(string)
	if intent != githubPRAuthorizationIntent || challenge == "" || nonce == "" {
		return nil, errors.New("invalid GitHub authorization state")
	}
	accountID, err := uuid.Parse(subject)
	if err != nil || accountID == uuid.Nil {
		return nil, errors.New("invalid account in GitHub authorization state")
	}
	return &githubPRAuthorizationState{
		AccountID: accountID,
		Redirect:  githubapp.SafeReturnPath(redirect),
		Challenge: challenge,
	}, nil
}

func (s *Server) githubPRAuthorizationCallbackURL() string {
	return githubapp.UserAuthorizationCallbackURL(s.BaseURL)
}

func (s *Server) githubPRAuthorizationCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     githubPRAuthorizationCookie,
		Value:    value,
		Path:     githubPRAuthorizationCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   strings.HasPrefix(strings.TrimSpace(s.BaseURL), "https://"),
		SameSite: http.SameSiteLaxMode,
	}
}

func githubPRAuthorizationRedirect(redirect, key, value string) string {
	target, err := url.Parse(githubapp.SafeReturnPath(redirect))
	if err != nil {
		return "/"
	}
	query := target.Query()
	query.Set(key, value)
	query.Set(githubPRAuthorizationProviderName, models.ProviderGitHub)
	target.RawQuery = query.Encode()
	return target.String()
}
