package authentication

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	"golang.org/x/oauth2"
)

const (
	accountLinkStateTTL     = 10 * time.Minute
	accountLinkStatePurpose = "account-provider-link"
	accountLinkResultParam  = "provider_link"
	accountLinkProvider     = models.ProviderGitHub
	defaultAccountLinkPath  = "/"
)

type accountLinkStateClaims struct {
	AccountID string `json:"account_id"`
	Redirect  string `json:"redirect"`
	Nonce     string `json:"nonce"`
	Purpose   string `json:"purpose"`
	jwtLib.StandardClaims
}

type githubAccountLinkUser struct {
	ID           string
	Username     string
	Name         string
	Email        string
	AvatarURL    string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (a *Handler) BeginGitHubAccountLink(w http.ResponseWriter, r *http.Request, account *models.Account) {
	redirectPath := safeAccountLinkRedirect(r.URL.Query().Get("redirect"))

	if a.isDev && a.githubAccountLinkOAuth == nil {
		err := a.linkGitHubAccount(r.Context(), account, githubAccountLinkUser{
			ID:          "dev-github-user",
			Username:    "devuser",
			Name:        "Dev GitHub User",
			Email:       "dev-github@superplane.local",
			AvatarURL:   "https://github.com/github.png",
			AccessToken: "dev-github-link-token",
		})
		a.redirectAccountLinkResult(w, r, redirectPath, accountLinkResult(err))
		return
	}

	if a.githubAccountLinkOAuth == nil {
		a.redirectAccountLinkResult(w, r, redirectPath, "failure")
		return
	}

	state, err := a.newAccountLinkState(account.ID.String(), redirectPath)
	if err != nil {
		a.redirectAccountLinkResult(w, r, redirectPath, "failure")
		return
	}

	http.Redirect(w, r, a.githubAccountLinkOAuth.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (a *Handler) CompleteGitHubAccountLink(w http.ResponseWriter, r *http.Request, account *models.Account) {
	claims, err := a.parseAccountLinkState(r.URL.Query().Get("state"))
	if err != nil || claims.AccountID != account.ID.String() {
		http.Error(w, "Invalid account link state", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("error") != "" {
		a.redirectAccountLinkResult(w, r, claims.Redirect, "denied")
		return
	}

	if a.githubAccountLinkOAuth == nil {
		a.redirectAccountLinkResult(w, r, claims.Redirect, "failure")
		return
	}

	token, err := a.githubAccountLinkOAuth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		a.redirectAccountLinkResult(w, r, claims.Redirect, "failure")
		return
	}

	user, err := fetchGitHubAccountLinkUser(r.Context(), a.githubAccountLinkOAuth.Client(r.Context(), token), token)
	if err != nil {
		a.redirectAccountLinkResult(w, r, claims.Redirect, "failure")
		return
	}

	err = a.linkGitHubAccount(r.Context(), account, user)
	a.redirectAccountLinkResult(w, r, claims.Redirect, accountLinkResult(err))
}

func (a *Handler) newAccountLinkState(accountID, redirectPath string) (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	now := time.Now()
	claims := accountLinkStateClaims{
		AccountID: accountID,
		Redirect:  safeAccountLinkRedirect(redirectPath),
		Nonce:     base64.RawURLEncoding.EncodeToString(nonce),
		Purpose:   accountLinkStatePurpose,
		StandardClaims: jwtLib.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(accountLinkStateTTL).Unix(),
		},
	}

	token := jwtLib.NewWithClaims(jwtLib.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSigner.Secret))
}

func (a *Handler) parseAccountLinkState(state string) (*accountLinkStateClaims, error) {
	if state == "" {
		return nil, errors.New("account link state is required")
	}

	claims := &accountLinkStateClaims{}
	token, err := jwtLib.ParseWithClaims(state, claims, func(token *jwtLib.Token) (any, error) {
		if token.Method != jwtLib.SigningMethodHS256 {
			return nil, errors.New("invalid account link state algorithm")
		}
		return []byte(a.jwtSigner.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid account link state")
	}
	if claims.Purpose != accountLinkStatePurpose || claims.AccountID == "" || claims.Nonce == "" {
		return nil, errors.New("invalid account link state claims")
	}
	if claims.Redirect != safeAccountLinkRedirect(claims.Redirect) {
		return nil, errors.New("invalid account link redirect")
	}

	return claims, nil
}

func (a *Handler) linkGitHubAccount(ctx context.Context, account *models.Account, user githubAccountLinkUser) error {
	encryptedToken, err := a.encryptor.Encrypt(ctx, []byte(user.AccessToken), []byte(user.Email))
	if err != nil {
		return err
	}

	var expiresAt *time.Time
	if !user.ExpiresAt.IsZero() {
		expiresAt = &user.ExpiresAt
	}

	return models.SaveLinkedAccountProvider(database.DB(ctx), &models.AccountProvider{
		AccountID:      account.ID,
		Provider:       accountLinkProvider,
		ProviderID:     user.ID,
		Username:       user.Username,
		Email:          utils.NormalizeEmail(user.Email),
		Name:           user.Name,
		AvatarURL:      user.AvatarURL,
		AccessToken:    base64.StdEncoding.EncodeToString(encryptedToken),
		RefreshToken:   user.RefreshToken,
		TokenExpiresAt: expiresAt,
	})
}

func fetchGitHubAccountLinkUser(ctx context.Context, client *http.Client, token *oauth2.Token) (githubAccountLinkUser, error) {
	githubClient := github.NewClient(client)
	user, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		return githubAccountLinkUser{}, err
	}

	emails, _, err := githubClient.Users.ListEmails(ctx, nil)
	if err != nil {
		return githubAccountLinkUser{}, err
	}

	email := user.GetEmail()
	for _, candidate := range emails {
		if candidate.GetPrimary() {
			email = candidate.GetEmail()
			break
		}
	}
	if email == "" {
		return githubAccountLinkUser{}, errors.New("GitHub account has no primary email")
	}

	return githubAccountLinkUser{
		ID:           fmt.Sprint(user.GetID()),
		Username:     user.GetLogin(),
		Name:         user.GetName(),
		Email:        email,
		AvatarURL:    user.GetAvatarURL(),
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
	}, nil
}

func safeAccountLinkRedirect(raw string) string {
	if raw == "" {
		return defaultAccountLinkPath
	}
	if strings.Contains(raw, "\\") {
		return defaultAccountLinkPath
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Fragment != "" {
		return defaultAccountLinkPath
	}
	if !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return defaultAccountLinkPath
	}

	return parsed.String()
}

func accountLinkResult(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, models.ErrProviderLinkedToAnotherAccount),
		errors.Is(err, models.ErrAccountProviderConflict):
		return "conflict"
	default:
		return "failure"
	}
}

func (a *Handler) redirectAccountLinkResult(w http.ResponseWriter, r *http.Request, redirectPath, result string) {
	target, err := url.Parse(safeAccountLinkRedirect(redirectPath))
	if err != nil {
		target = &url.URL{Path: defaultAccountLinkPath}
	}

	query := target.Query()
	query.Set(accountLinkResultParam, result)
	query.Set("provider", accountLinkProvider)
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusSeeOther)
}
