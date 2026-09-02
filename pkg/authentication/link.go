package authentication

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/markbates/goth"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	authLinkStatePrefix = "link:"
	authLinkIntent      = "link"
	authLinkTTL         = 10 * time.Minute
	authErrorLinkFailed = "signin_method_in_use"
)

type linkState struct {
	AccountID string
	Provider  string
	Redirect  string
}

func isLinkIntent(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("intent")), authLinkIntent)
}

func (a *Handler) sessionAccountFromCookie(r *http.Request) (*models.Account, error) {
	cookie, err := r.Cookie("account_token")
	if err != nil {
		return nil, err
	}

	claims, err := a.jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return nil, err
	}

	accountID, _ := claims["sub"].(string)
	if accountID == "" {
		return nil, errors.New("account ID missing from token")
	}

	account, err := models.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}
	if account.IsBlocked() {
		return nil, models.ErrAccountBlocked
	}

	iat, _ := claims["iat"].(float64)
	if !account.IsSessionFresh(int64(iat)) {
		return nil, errors.New("session invalidated by password change")
	}

	return account, nil
}

func (a *Handler) signLinkState(accountID, provider, redirectURL string) (string, error) {
	token, err := a.jwtSigner.GenerateWithClaims(authLinkTTL, map[string]string{
		"sub":      accountID,
		"intent":   authLinkIntent,
		"provider": provider,
		"redirect": redirectURL,
		"jti":      uuid.NewString(),
	})
	if err != nil {
		return "", err
	}
	return authLinkStatePrefix + token, nil
}

func (a *Handler) parseLinkState(state string) (*linkState, error) {
	if !strings.HasPrefix(state, authLinkStatePrefix) {
		return nil, errors.New("not a link state")
	}

	claims, err := a.jwtSigner.ValidateAndGetClaims(strings.TrimPrefix(state, authLinkStatePrefix))
	if err != nil {
		return nil, err
	}

	intent, _ := claims["intent"].(string)
	if intent != authLinkIntent {
		return nil, errors.New("invalid link intent")
	}

	accountID, _ := claims["sub"].(string)
	provider, _ := claims["provider"].(string)
	redirect, _ := claims["redirect"].(string)
	nonce, _ := claims["jti"].(string)
	if accountID == "" || provider == "" || nonce == "" {
		return nil, errors.New("invalid link state")
	}
	if _, err := uuid.Parse(accountID); err != nil {
		return nil, err
	}

	return &linkState{AccountID: accountID, Provider: provider, Redirect: redirect}, nil
}

func linkStateFromRequest(r *http.Request) string {
	state := r.URL.Query().Get("state")
	if state == "" {
		state = r.FormValue("state")
	}
	return state
}

func (a *Handler) completeProviderLink(w http.ResponseWriter, r *http.Request, gothUser goth.User, state *linkState) {
	if !strings.EqualFold(gothUser.Provider, state.Provider) {
		http.Error(w, "Sign-in method does not match", http.StatusBadRequest)
		return
	}

	sessionAccount, err := a.sessionAccountFromCookie(r)
	if err != nil {
		http.Redirect(w, r, "/login?redirect="+url.QueryEscape(state.Redirect), http.StatusSeeOther)
		return
	}
	if sessionAccount.ID.String() != state.AccountID {
		http.Error(w, "Sign-in method cannot change for another account", http.StatusForbidden)
		return
	}

	err = LinkProviderToAccount(a.encryptor, sessionAccount, gothUser)
	if errors.Is(err, models.ErrSignInIdentityInUse) {
		http.Redirect(w, r, linkErrorRedirectURL(state.Redirect, authErrorLinkFailed, gothUser.Provider), http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Errorf("Error linking %s for account %s: %v", gothUser.Provider, sessionAccount.ID, err)
		http.Error(w, "Failed to connect sign-in method", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, linkSuccessRedirectURL(state.Redirect, gothUser.Provider), http.StatusSeeOther)
}

func LinkProviderToAccount(encryptor crypto.Encryptor, account *models.Account, gothUser goth.User) error {
	existing, err := models.FindAccountByProvider(gothUser.Provider, gothUser.UserID)
	if err == nil && existing.ID != account.ID {
		return models.ErrSignInIdentityInUse
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return updateAccountProviders(encryptor, account, gothUser)
}

func linkSuccessRedirectURL(redirect, provider string) string {
	return appendAuthQuery(redirect, "auth_link_result", "connected", provider)
}

func linkErrorRedirectURL(redirect, code, provider string) string {
	return appendAuthQuery(redirect, authErrorParam, code, provider)
}

func appendAuthQuery(redirect, key, value, provider string) string {
	if redirect == "" || !isValidRedirectURL(redirect) {
		redirect = "/"
	}

	parsed, err := url.Parse(redirect)
	if err != nil {
		return "/"
	}

	query := parsed.Query()
	query.Set(key, value)
	query.Set(authProviderParam, provider)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
