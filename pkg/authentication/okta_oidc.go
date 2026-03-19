package authentication

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// Matches pkg/grpc/actions/organizations/okta_idp.go (OAuth client secret AAD).
const oktaOAuthClientSecretCredentialName = "okta_oauth_client_secret"

func (a *Handler) handleOktaAuthStart(w http.ResponseWriter, r *http.Request) {
	if a.publicAppBaseURL == "" {
		log.Error("Okta OIDC: public base URL is not configured")
		http.Error(w, "authentication is not configured", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	orgID := strings.TrimSpace(vars["org_id"])
	if _, err := uuid.Parse(orgID); err != nil {
		http.Error(w, "invalid organization", http.StatusBadRequest)
		return
	}

	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !org.IsProviderAllowed(models.ProviderOkta) {
		http.Error(w, "Okta sign-in is not enabled for this organization", http.StatusForbidden)
		return
	}

	idp, err := models.FindOrganizationOktaIDPByOrganizationID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Okta is not configured", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !idp.OIDCEnabled {
		http.Error(w, "Okta OIDC is disabled", http.StatusForbidden)
		return
	}
	if len(idp.OAuthClientSecretCiphertext) == 0 {
		http.Error(w, "Okta OIDC is not fully configured", http.StatusServiceUnavailable)
		return
	}

	redirect := strings.TrimSpace(r.URL.Query().Get("redirect"))
	if redirect != "" && !isValidRedirectURL(redirect) {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
	}

	state, err := a.jwtSigner.SignOktaOAuthState(orgID, redirect, 15*time.Minute)
	if err != nil {
		log.Errorf("Okta OIDC: sign state: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	issuer := strings.TrimRight(strings.TrimSpace(idp.IssuerBaseURL), "/")
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Errorf("Okta OIDC: NewProvider %q: %v", issuer, err)
		http.Error(w, "identity provider unavailable", http.StatusBadGateway)
		return
	}

	oauth2Cfg := oauth2.Config{
		ClientID:     idp.OAuthClientID,
		ClientSecret: "", // filled after decrypt in callback; authorize does not need it
		RedirectURL:  a.publicAppBaseURL + "/auth/okta/" + orgID + "/callback",
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}

	authURL := oauth2Cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (a *Handler) handleOktaAuthCallback(w http.ResponseWriter, r *http.Request) {
	if a.publicAppBaseURL == "" {
		http.Error(w, "authentication is not configured", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	orgID := strings.TrimSpace(vars["org_id"])
	if _, err := uuid.Parse(orgID); err != nil {
		http.Error(w, "invalid organization", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	stateOrg, redirect, err := a.jwtSigner.ParseOktaOAuthState(state)
	if err != nil || stateOrg != orgID {
		http.Error(w, "invalid or expired state", http.StatusBadRequest)
		return
	}
	if redirect != "" && !isValidRedirectURL(redirect) {
		redirect = "/"
	}
	if redirect == "" {
		redirect = "/"
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "organization not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !org.IsProviderAllowed(models.ProviderOkta) {
		http.Error(w, "Okta sign-in is not enabled for this organization", http.StatusForbidden)
		return
	}

	idp, err := models.FindOrganizationOktaIDPByOrganizationID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Okta is not configured", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !idp.OIDCEnabled {
		http.Error(w, "Okta OIDC is disabled", http.StatusForbidden)
		return
	}

	clientSecretBytes, err := a.encryptor.Decrypt(r.Context(), idp.OAuthClientSecretCiphertext, []byte(oktaOAuthClientSecretCredentialName))
	if err != nil {
		log.Errorf("Okta OIDC: decrypt client secret: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	issuer := strings.TrimRight(strings.TrimSpace(idp.IssuerBaseURL), "/")
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Errorf("Okta OIDC: NewProvider %q: %v", issuer, err)
		http.Error(w, "identity provider unavailable", http.StatusBadGateway)
		return
	}

	oauth2Cfg := oauth2.Config{
		ClientID:     idp.OAuthClientID,
		ClientSecret: string(clientSecretBytes),
		RedirectURL:  a.publicAppBaseURL + "/auth/okta/" + orgID + "/callback",
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}

	tok, err := oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		log.Warnf("Okta OIDC: token exchange failed: %v", err)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Error(w, "missing id_token", http.StatusUnauthorized)
		return
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: idp.OAuthClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Warnf("Okta OIDC: id_token verify failed: %v", err)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Sub           string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Warnf("Okta OIDC: claims: %v", err)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}
	if claims.Sub == "" {
		http.Error(w, "invalid identity token", http.StatusUnauthorized)
		return
	}

	email := utils.NormalizeEmail(strings.TrimSpace(claims.Email))
	if email == "" {
		http.Error(w, "email claim is required", http.StatusUnauthorized)
		return
	}
	if claims.EmailVerified != nil && !*claims.EmailVerified {
		http.Error(w, "email is not verified", http.StatusForbidden)
		return
	}

	user, err := models.FindActiveUserByEmail(orgID, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "no SuperPlane user exists for this email in the organization", http.StatusForbidden)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user.IsServiceAccount() {
		http.Error(w, "unsupported account type", http.StatusForbidden)
		return
	}
	if user.AccountID == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := models.FindScimMappingByOrganizationAndUserID(database.Conn(), orgID, user.ID.String()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "sign-in requires an account provisioned for this organization", http.StatusForbidden)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	account, err := models.FindAccountByID(user.AccountID.String())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	displayName := strings.TrimSpace(claims.Name)
	if displayName == "" {
		displayName = user.Name
	}

	if err := upsertOktaAccountProvider(a.encryptor, account, issuer, claims.Sub, email, displayName, claims.Picture); err != nil {
		log.Errorf("Okta OIDC: account provider: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := a.acceptPendingInvitations(account); err != nil {
		log.Errorf("Okta OIDC: invitations: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := a.issueAccountSession(w, r, account.ID.String()); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
}

func upsertOktaAccountProvider(
	encryptor crypto.Encryptor,
	account *models.Account,
	issuerBase, sub, email, name, picture string,
) error {
	providerUserID := crypto.HashToken(strings.TrimSuffix(issuerBase, "/") + "\x00" + sub)
	placeholderTok, err := encryptor.Encrypt(context.Background(), []byte("-"), []byte(email))
	if err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString(placeholderTok)

	var existing models.AccountProvider
	q := database.Conn().Where("account_id = ? AND provider = ?", account.ID, models.ProviderOkta).First(&existing)
	if q.Error == nil {
		existing.ProviderID = providerUserID
		existing.Username = sub
		existing.Email = email
		existing.Name = name
		existing.AvatarURL = picture
		existing.AccessToken = enc
		return database.Conn().Save(&existing).Error
	}
	if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
		return q.Error
	}

	ap := &models.AccountProvider{
		AccountID:   account.ID,
		Provider:    models.ProviderOkta,
		ProviderID:  providerUserID,
		Username:    sub,
		Email:       email,
		Name:        name,
		AvatarURL:   picture,
		AccessToken: enc,
	}
	return database.Conn().Create(ap).Error
}
