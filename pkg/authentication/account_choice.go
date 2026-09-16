package authentication

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/markbates/goth"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	authSelectStatePrefix = "select:"
	authSelectTTL         = 10 * time.Minute
)

type selectState struct {
	TokenHash    string
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
	token, err := a.storeSelectState(r.Context(), gothUser, getRedirectURL(r))
	if err != nil {
		log.Errorf("Error storing account choice state: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login/choose-account?token="+url.QueryEscape(token), http.StatusSeeOther)
}

func (a *Handler) storeSelectState(ctx context.Context, gothUser goth.User, redirectURL string) (string, error) {
	token, err := crypto.Base64String(32)
	if err != nil {
		return "", err
	}

	tokenHash := crypto.HashToken(token)
	accessToken, err := sealChoiceSecret(a.encryptor, tokenHash, gothUser.AccessToken)
	if err != nil {
		return "", err
	}
	refreshToken, err := sealChoiceSecret(a.encryptor, tokenHash, gothUser.RefreshToken)
	if err != nil {
		return "", err
	}

	state := &models.AccountChoiceState{
		TokenHash:    tokenHash,
		Provider:     gothUser.Provider,
		ProviderID:   gothUser.UserID,
		Redirect:     redirectURL,
		Email:        gothUser.Email,
		Name:         gothUser.Name,
		Nickname:     gothUser.NickName,
		AvatarURL:    gothUser.AvatarURL,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(authSelectTTL),
	}
	if !gothUser.ExpiresAt.IsZero() {
		expiresAt := gothUser.ExpiresAt
		state.TokenExpiresAt = &expiresAt
	}

	if err := models.CreateAccountChoiceState(database.DB(ctx), state); err != nil {
		return "", err
	}
	return token, nil
}

func (a *Handler) loadSelectState(ctx context.Context, token string) (*selectState, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("missing select token")
	}

	record, err := models.FindValidAccountChoiceState(database.DB(ctx), crypto.HashToken(token), time.Now())
	if err != nil {
		return nil, err
	}

	state, err := a.selectStateFromRecord(record)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (a *Handler) claimSelectState(ctx context.Context, tokenHash string) (*selectState, error) {
	record, err := models.ClaimAccountChoiceState(database.DB(ctx), tokenHash, time.Now())
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, errors.New("select token already used")
	}
	return a.selectStateFromRecord(record)
}

func (a *Handler) selectStateFromRecord(record *models.AccountChoiceState) (*selectState, error) {
	accessToken, err := openChoiceSecret(a.encryptor, record.TokenHash, record.AccessToken)
	if err != nil {
		return nil, err
	}
	refreshToken, err := openChoiceSecret(a.encryptor, record.TokenHash, record.RefreshToken)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Time{}
	if record.TokenExpiresAt != nil {
		expiresAt = *record.TokenExpiresAt
	}

	return &selectState{
		TokenHash:    record.TokenHash,
		Provider:     record.Provider,
		ProviderID:   record.ProviderID,
		Redirect:     record.Redirect,
		Email:        record.Email,
		Name:         record.Name,
		NickName:     record.Nickname,
		AvatarURL:    record.AvatarURL,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (a *Handler) handleListAccountChoice(w http.ResponseWriter, r *http.Request) {
	state, err := a.loadSelectState(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, "Invalid or expired selection", http.StatusBadRequest)
		return
	}

	items, err := a.accountChoiceItems(r.Context(), state)
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

	state, err := a.loadSelectState(r.Context(), r.FormValue("token"))
	if err != nil {
		http.Error(w, "Invalid or expired selection", http.StatusBadRequest)
		return
	}

	accountID := strings.TrimSpace(r.FormValue("account_id"))
	if _, err := uuid.Parse(accountID); err != nil {
		http.Error(w, "Invalid account", http.StatusBadRequest)
		return
	}

	account, err := a.chosenAccountForSelectState(r.Context(), state, accountID)
	if err != nil {
		http.Error(w, "Account is not available", http.StatusForbidden)
		return
	}

	claimed, err := a.claimSelectState(r.Context(), state.TokenHash)
	if err != nil {
		http.Error(w, "Invalid or expired selection", http.StatusBadRequest)
		return
	}

	gothUser := gothUserFromSelectState(claimed)
	if err := updateAccountProviders(a.encryptor, account, gothUser); err != nil {
		log.Errorf("Error updating account providers for choice %s: %v", account.ID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := IssueAccountSession(w, r, a.jwtSigner, account.ID.String()); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	redirectURL := claimed.Redirect
	if !isValidRedirectURL(redirectURL) {
		redirectURL = "/"
	}
	writePostAuthRedirect(w, r, redirectURL)
}

func (a *Handler) chosenAccountForSelectState(ctx context.Context, state *selectState, accountID string) (*models.Account, error) {
	accounts, err := models.FindAccountsByProvider(database.DB(ctx), state.Provider, state.ProviderID)
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

func (a *Handler) accountChoiceItems(ctx context.Context, state *selectState) ([]accountChoiceItem, error) {
	accounts, err := models.FindAccountsByProvider(database.DB(ctx), state.Provider, state.ProviderID)
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

func sealChoiceSecret(encryptor crypto.Encryptor, tokenHash, secret string) ([]byte, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, nil
	}
	return encryptor.Encrypt(context.Background(), []byte(secret), []byte(tokenHash))
}

func openChoiceSecret(encryptor crypto.Encryptor, tokenHash string, sealed []byte) (string, error) {
	if len(sealed) == 0 {
		return "", nil
	}
	plain, err := encryptor.Decrypt(context.Background(), sealed, []byte(tokenHash))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
