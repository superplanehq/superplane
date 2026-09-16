package authentication

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/markbates/goth"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	authSelectStatePrefix = "select:"
	authSelectIntent      = "select"
	authSelectTTL         = 10 * time.Minute
)

type selectState struct {
	Provider     string
	ProviderID   string
	Redirect     string
	Email        string
	Name         string
	NickName     string
	AvatarURL    string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type accountChoiceItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func (a *Handler) redirectToAccountChoice(w http.ResponseWriter, r *http.Request, gothUser goth.User) {
	token, err := a.signSelectState(gothUser, getRedirectURL(r))
	if err != nil {
		log.Errorf("Error signing account choice state: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login/choose-account?token="+url.QueryEscape(token), http.StatusSeeOther)
}

func (a *Handler) signSelectState(gothUser goth.User, redirectURL string) (string, error) {
	claims := map[string]string{
		"sub":           gothUser.UserID,
		"intent":        authSelectIntent,
		"provider":      gothUser.Provider,
		"provider_id":   gothUser.UserID,
		"redirect":      redirectURL,
		"jti":           uuid.NewString(),
		"email":         gothUser.Email,
		"name":          gothUser.Name,
		"nickname":      gothUser.NickName,
		"avatar":        gothUser.AvatarURL,
		"access_token":  gothUser.AccessToken,
		"refresh_token": gothUser.RefreshToken,
	}
	if !gothUser.ExpiresAt.IsZero() {
		claims["expires_at"] = gothUser.ExpiresAt.UTC().Format(time.RFC3339)
	}

	token, err := a.jwtSigner.GenerateWithClaims(authSelectTTL, claims)
	if err != nil {
		return "", err
	}
	return authSelectStatePrefix + token, nil
}

func (a *Handler) parseSelectState(state string) (*selectState, error) {
	if !strings.HasPrefix(state, authSelectStatePrefix) {
		return nil, errors.New("not a select state")
	}

	claims, err := a.jwtSigner.ValidateAndGetClaims(strings.TrimPrefix(state, authSelectStatePrefix))
	if err != nil {
		return nil, err
	}

	intent, _ := claims["intent"].(string)
	if intent != authSelectIntent {
		return nil, errors.New("invalid select intent")
	}

	provider, _ := claims["provider"].(string)
	providerID, _ := claims["provider_id"].(string)
	nonce, _ := claims["jti"].(string)
	if provider == "" || providerID == "" || nonce == "" {
		return nil, errors.New("invalid select state")
	}

	redirect, _ := claims["redirect"].(string)
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	nickname, _ := claims["nickname"].(string)
	avatar, _ := claims["avatar"].(string)
	accessToken, _ := claims["access_token"].(string)
	refreshToken, _ := claims["refresh_token"].(string)
	expiresAt := time.Time{}
	if rawExpires, _ := claims["expires_at"].(string); rawExpires != "" {
		parsed, parseErr := time.Parse(time.RFC3339, rawExpires)
		if parseErr == nil {
			expiresAt = parsed
		}
	}

	return &selectState{
		Provider:     provider,
		ProviderID:   providerID,
		Redirect:     redirect,
		Email:        email,
		Name:         name,
		NickName:     nickname,
		AvatarURL:    avatar,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (a *Handler) handleListAccountChoice(w http.ResponseWriter, r *http.Request) {
	state, err := a.parseSelectState(r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, "Invalid or expired selection", http.StatusBadRequest)
		return
	}

	items, err := a.accountChoiceItems(state)
	if err != nil {
		log.Errorf("Error listing account choices: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if len(items) == 0 {
		http.Error(w, "No matching accounts", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", jsonContentType)
	if err := json.NewEncoder(w).Encode(map[string]any{"accounts": items}); err != nil {
		log.Errorf("Error encoding account choices: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (a *Handler) completeAccountChoice(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	state, err := a.parseSelectState(r.FormValue("token"))
	if err != nil {
		http.Error(w, "Invalid or expired selection", http.StatusBadRequest)
		return
	}

	accountID := strings.TrimSpace(r.FormValue("account_id"))
	if _, err := uuid.Parse(accountID); err != nil {
		http.Error(w, "Invalid account", http.StatusBadRequest)
		return
	}

	account, err := a.chosenAccountForSelectState(state, accountID)
	if err != nil {
		http.Error(w, "Account is not available", http.StatusForbidden)
		return
	}

	gothUser := gothUserFromSelectState(state)
	if err := updateAccountProviders(a.encryptor, account, gothUser); err != nil {
		log.Errorf("Error updating account providers for choice %s: %v", account.ID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := IssueAccountSession(w, r, a.jwtSigner, account.ID.String()); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	redirectURL := state.Redirect
	if !isValidRedirectURL(redirectURL) {
		redirectURL = "/"
	}
	writePostAuthRedirect(w, r, redirectURL)
}

func (a *Handler) chosenAccountForSelectState(state *selectState, accountID string) (*models.Account, error) {
	accounts, err := models.FindAccountsByProvider(database.Conn(), state.Provider, state.ProviderID)
	if err != nil {
		return nil, err
	}

	for _, candidate := range unblockedAccounts(accounts) {
		if candidate.ID.String() == accountID {
			return candidate, nil
		}
	}
	return nil, errors.New("account is not a candidate")
}

func (a *Handler) accountChoiceItems(state *selectState) ([]accountChoiceItem, error) {
	accounts, err := models.FindAccountsByProvider(database.Conn(), state.Provider, state.ProviderID)
	if err != nil {
		return nil, err
	}

	items := make([]accountChoiceItem, 0, len(accounts))
	for _, account := range unblockedAccounts(accounts) {
		avatarURL := ""
		provider, err := account.FindAccountProviderByID(state.Provider, state.ProviderID)
		if err == nil {
			avatarURL = provider.AvatarURL
		}
		items = append(items, accountChoiceItem{
			ID:        account.ID.String(),
			Name:      account.Name,
			Email:     account.Email,
			AvatarURL: avatarURL,
		})
	}
	return items, nil
}

func gothUserFromSelectState(state *selectState) goth.User {
	return goth.User{
		Provider:     state.Provider,
		UserID:       state.ProviderID,
		Email:        state.Email,
		Name:         state.Name,
		NickName:     state.NickName,
		AvatarURL:    state.AvatarURL,
		AccessToken:  state.AccessToken,
		RefreshToken: state.RefreshToken,
		ExpiresAt:    state.ExpiresAt,
	}
}
