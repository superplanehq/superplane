package authentication

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const SignupDisabledError = "signup is currently disabled"

type Handler struct {
	jwtSigner   *jwt.Signer
	authService authorization.Authorization
	encryptor   crypto.Encryptor
	isDev       bool
	templateDir string
	blockSignup bool
}

type ProviderConfig struct {
	Key         string
	Secret      string
	CallbackURL string
}

func NewHandler(jwtSigner *jwt.Signer, encryptor crypto.Encryptor, authService authorization.Authorization, appEnv string, templateDir string, blockSignup bool) *Handler {
	return &Handler{
		jwtSigner:   jwtSigner,
		encryptor:   encryptor,
		authService: authService,
		isDev:       appEnv == "development",
		templateDir: templateDir,
		blockSignup: blockSignup,
	}
}

func (a *Handler) InitializeProviders(providers map[string]ProviderConfig) {
	var gothProviders []goth.Provider

	for providerName, config := range providers {
		if config.Key == "" || config.Secret == "" {
			log.Warnf("%s OAuth not configured - missing key/secret", providerName)
			continue
		}

		switch providerName {
		case models.ProviderGitHub:
			gothProviders = append(gothProviders, github.New(config.Key, config.Secret, config.CallbackURL, "user:email"))
			log.Infof("GitHub OAuth provider initialized")
		case models.ProviderGoogle:
			gothProviders = append(gothProviders, google.New(config.Key, config.Secret, config.CallbackURL, "email", "profile"))
			log.Infof("Google OAuth provider initialized")
		default:
			log.Warnf("Unknown provider: %s", providerName)
		}
	}

	if len(gothProviders) > 0 {
		goth.UseProviders(gothProviders...)
	} else {
		log.Warn("No OAuth providers configured")
	}
}

func (a *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/logout", a.handleLogout).Methods("GET")
	router.HandleFunc("/login", a.handleLoginPage).Methods("GET")

	//
	// If we are running the application locally,
	// we provide handlers that auto-autenticate to
	// avoid having to authenticate with GitHub every time.
	//
	if a.isDev {
		log.Info("Registering development authentication routes")
		router.HandleFunc("/auth/{provider}/callback", a.handleDevAuth).Methods("GET")
		router.HandleFunc("/auth/{provider}", a.handleDevAuth).Methods("GET")
		return
	}

	router.HandleFunc("/auth/{provider}/callback", a.handleAuthCallback).Methods("GET")
	router.HandleFunc("/auth/{provider}", a.handleAuth).Methods("GET")
}

func (a *Handler) handleAuth(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err == nil {
		a.handleSuccessfulAuth(w, r, gothUser)
		return
	}

	gothic.BeginAuthHandler(w, r)
}

func (a *Handler) handleDevAuth(w http.ResponseWriter, r *http.Request) {
	if !a.isDev {
		http.Error(w, "Not available in production", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	provider := vars["provider"]
	mockUser := goth.User{
		UserID:      "dev-user-123",
		Email:       "dev@superplane.local",
		Name:        "Dev User",
		NickName:    "devuser",
		Provider:    provider,
		AvatarURL:   "https://github.com/github.png",
		AccessToken: "dev-token-" + provider,
	}

	account, err := a.findOrCreateAccount(mockUser.Name, mockUser.Email)

	if err != nil {
		if err.Error() == SignupDisabledError {
			http.Error(w, SignupDisabledError, http.StatusForbidden)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = updateAccountProviders(a.encryptor, account, mockUser)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = a.acceptPendingInvitations(account)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	a.handleSuccessfulAuth(w, r, mockUser)
}

func (a *Handler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	account, err := a.findOrCreateAccount(gothUser.Name, gothUser.Email)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = updateAccountProviders(a.encryptor, account, gothUser)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = a.acceptPendingInvitations(account)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	a.handleSuccessfulAuth(w, r, gothUser)
}

func (a *Handler) acceptPendingInvitations(account *models.Account) error {
	invitations, err := account.FindPendingInvitations()
	if err != nil {
		log.Errorf("Error finding pending invitations for account %s: %v", account.Email, err)
		return err
	}

	for _, invitation := range invitations {
		err := a.acceptInvitation(invitation, account)
		if err != nil {
			log.Errorf("Error accepting invitation to %s for account %s: %v", invitation.OrganizationID, account.Email, err)
			return err
		}
	}

	return nil
}

func (a *Handler) acceptInvitation(invitation models.OrganizationInvitation, account *models.Account) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		user, err := models.CreateUserInTransaction(tx, invitation.OrganizationID, account.ID, account.Email, account.Name)
		if err != nil {
			return err
		}

		invitation.State = models.InvitationStateAccepted
		invitation.UpdatedAt = time.Now()
		err = tx.Save(&invitation).Error
		if err != nil {
			return err
		}

		err = a.authService.AssignRole(user.ID.String(), models.RoleOrgViewer, invitation.OrganizationID.String(), models.DomainTypeOrganization)
		if err != nil {
			return err
		}

		return nil
	})
}

func (a *Handler) handleSuccessfulAuth(w http.ResponseWriter, r *http.Request, gothUser goth.User) {
	account, err := models.FindAccountByEmail(gothUser.Email)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	token, err := a.jwtSigner.Generate(account.ID.String(), 24*time.Hour)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "account_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(24 * time.Hour.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (a *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	gothic.Logout(w, r)

	// Clear the account cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "account_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}

func (a *Handler) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("account_token")
	if err == nil {
		_, err := a.jwtSigner.ValidateAndGetClaims(cookie.Value)
		if err == nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
	}

	templatePath := filepath.Join(a.templateDir, "login.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Errorf("Error parsing login template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	providers := goth.GetProviders()
	providerNames := make([]string, 0, len(providers))
	for name := range providers {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)

	data := struct {
		Providers []string
	}{
		Providers: providerNames,
	}

	err = t.Execute(w, data)
	if err != nil {
		log.Errorf("Error executing login template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (a *Handler) findOrCreateAccount(name, email string) (*models.Account, error) {
	account, err := models.FindAccountByEmail(email)
	if err == nil {
		return account, nil
	}

	if a.blockSignup {
		log.Warnf("Signup blocked for email: %s", email)
		return nil, fmt.Errorf(SignupDisabledError)
	}

	account, err = models.CreateAccount(name, email)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func updateAccountProviders(encryptor crypto.Encryptor, account *models.Account, gothUser goth.User) error {
	accessToken, err := encryptor.Encrypt(context.Background(), []byte(gothUser.AccessToken), []byte(gothUser.Email))
	if err != nil {
		return err
	}

	accountProvider, err := account.FindAccountProviderByID(gothUser.Provider, gothUser.UserID)

	//
	// If we already have an account provider for this provider and provider ID, we just update it.
	//
	if err == nil {
		accountProvider.AccessToken = base64.StdEncoding.EncodeToString(accessToken)
		accountProvider.Username = gothUser.NickName
		accountProvider.Email = gothUser.Email
		accountProvider.Name = gothUser.Name
		accountProvider.AvatarURL = gothUser.AvatarURL
		accountProvider.RefreshToken = gothUser.RefreshToken
		if !gothUser.ExpiresAt.IsZero() {
			accountProvider.TokenExpiresAt = &gothUser.ExpiresAt
		}

		return database.Conn().Save(accountProvider).Error
	}

	//
	// Otherwise, we create a new account provider.
	//
	accountProvider = &models.AccountProvider{
		AccountID:      account.ID,
		Provider:       gothUser.Provider,
		ProviderID:     gothUser.UserID,
		Username:       gothUser.NickName,
		Email:          gothUser.Email,
		Name:           gothUser.Name,
		AvatarURL:      gothUser.AvatarURL,
		AccessToken:    base64.StdEncoding.EncodeToString(accessToken),
		RefreshToken:   gothUser.RefreshToken,
		TokenExpiresAt: &gothUser.ExpiresAt,
	}

	return database.Conn().Create(accountProvider).Error
}
