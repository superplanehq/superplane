package authentication

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/markbates/goth"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	authConnectStatePrefix = "connect:"
	authConnectIntent      = "connect"
	authConnectTTL         = 10 * time.Minute
	authConnectResultParam = "linked_account"
	authErrorConnectInUse  = "linked_account_in_use"
)

// connectableProviders lists the services that hold activity SuperPlane can
// attribute to a member. A linked account is not a sign-in method, so a
// provider that only proves identity does not belong here.
var connectableProviders = []string{models.ProviderGitHub}

type connectState struct {
	AccountID string
	Provider  string
	Redirect  string
}

func isConnectIntent(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("intent")), authConnectIntent)
}

func isConnectableProvider(provider string) bool {
	return slices.ContainsFunc(connectableProviders, func(candidate string) bool {
		return strings.EqualFold(candidate, provider)
	})
}

func (a *Handler) signConnectState(accountID, provider, redirectURL string) (string, error) {
	token, err := a.jwtSigner.GenerateWithClaims(authConnectTTL, map[string]string{
		"sub":      accountID,
		"intent":   authConnectIntent,
		"provider": provider,
		"redirect": redirectURL,
		"jti":      uuid.NewString(),
	})
	if err != nil {
		return "", err
	}
	return authConnectStatePrefix + token, nil
}

func (a *Handler) parseConnectState(state string) (*connectState, error) {
	if !strings.HasPrefix(state, authConnectStatePrefix) {
		return nil, errors.New("not a connect state")
	}

	claims, err := a.jwtSigner.ValidateAndGetClaims(strings.TrimPrefix(state, authConnectStatePrefix))
	if err != nil {
		return nil, err
	}

	intent, _ := claims["intent"].(string)
	if intent != authConnectIntent {
		return nil, errors.New("invalid connect intent")
	}

	accountID, _ := claims["sub"].(string)
	provider, _ := claims["provider"].(string)
	redirect, _ := claims["redirect"].(string)
	nonce, _ := claims["jti"].(string)
	if accountID == "" || provider == "" || nonce == "" {
		return nil, errors.New("invalid connect state")
	}
	if _, err := uuid.Parse(accountID); err != nil {
		return nil, err
	}

	return &connectState{AccountID: accountID, Provider: provider, Redirect: redirect}, nil
}

// completeAccountConnection stores the external identity as a linked account.
// It issues no session and writes no sign-in credential, so a linked account
// never becomes a way to sign in.
func (a *Handler) completeAccountConnection(w http.ResponseWriter, r *http.Request, gothUser goth.User, state *connectState) {
	if !strings.EqualFold(gothUser.Provider, state.Provider) {
		http.Error(w, "Linked account does not match", http.StatusBadRequest)
		return
	}
	if !isConnectableProvider(gothUser.Provider) {
		http.Error(w, "Provider does not support linked accounts", http.StatusBadRequest)
		return
	}

	sessionAccount, err := a.sessionAccountFromCookie(r)
	if err != nil {
		http.Redirect(w, r, "/login?redirect="+url.QueryEscape(state.Redirect), http.StatusSeeOther)
		return
	}
	if sessionAccount.ID.String() != state.AccountID {
		http.Error(w, "Linked account cannot change for another account", http.StatusForbidden)
		return
	}

	username := strings.TrimSpace(gothUser.NickName)
	if username == "" {
		http.Error(w, "Provider returned no username", http.StatusBadGateway)
		return
	}

	linked := models.NewAccountLinkedAccount(
		sessionAccount.ID,
		strings.ToLower(gothUser.Provider),
		gothUser.UserID,
		username,
		gothUser.Name,
		gothUser.AvatarURL,
	)

	err = models.SaveAccountLinkedAccount(database.DB(r.Context()), linked)
	if errors.Is(err, models.ErrLinkedAccountInUse) {
		http.Redirect(w, r, connectErrorRedirectURL(state.Redirect, authErrorConnectInUse, gothUser.Provider), http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Errorf("Error linking %s account for %s: %v", gothUser.Provider, sessionAccount.ID, err)
		http.Error(w, "Failed to link account", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, connectSuccessRedirectURL(state.Redirect, gothUser.Provider), http.StatusSeeOther)
}

// finishAccountConnection handles a connect request that already carries a
// completed provider session. It signs a fresh state from the current session,
// so a caller cannot link an identity to an account it does not hold.
func (a *Handler) finishAccountConnection(w http.ResponseWriter, r *http.Request, gothUser goth.User) {
	account, err := a.sessionAccountFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	signed, err := a.signConnectState(account.ID.String(), gothUser.Provider, getRedirectURL(r))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	state, err := a.parseConnectState(signed)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	a.completeAccountConnection(w, r, gothUser, state)
}

func connectSuccessRedirectURL(redirect, provider string) string {
	return appendAuthQuery(redirect, authConnectResultParam, "linked", provider)
}

func connectErrorRedirectURL(redirect, code, provider string) string {
	return appendAuthQuery(redirect, authErrorParam, code, provider)
}
