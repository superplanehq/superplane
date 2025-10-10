package authentication

import (
	"context"
	"encoding/base64"
	"html/template"
	"net/http"
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

type Handler struct {
	jwtSigner   *jwt.Signer
	authService authorization.Authorization
	encryptor   crypto.Encryptor
	isDev       bool
}

type ProviderConfig struct {
	Key         string
	Secret      string
	CallbackURL string
}

func NewHandler(jwtSigner *jwt.Signer, encryptor crypto.Encryptor, authService authorization.Authorization, appEnv string) *Handler {
	return &Handler{
		jwtSigner:   jwtSigner,
		encryptor:   encryptor,
		authService: authService,
		isDev:       appEnv == "development",
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

	account, err := findOrCreateAccount(mockUser.Name, mockUser.Email)
	if err != nil {
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

	account, err := findOrCreateAccount(gothUser.Name, gothUser.Email)
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

		//
		// TODO: this is not using the transaction properly
		//
		return a.authService.AssignRole(user.ID.String(), models.RoleOrgViewer, invitation.OrganizationID.String(), models.DomainTypeOrganization)
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
	t, err := template.New("login").Parse(loginTemplate)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	providers := goth.GetProviders()
	providerNames := make([]string, 0, len(providers))
	for name := range providers {
		providerNames = append(providerNames, name)
	}

	data := struct {
		Providers []string
	}{
		Providers: providerNames,
	}

	t.Execute(w, data)
}

func findOrCreateAccount(name, email string) (*models.Account, error) {
	account, err := models.FindAccountByEmail(email)
	if err == nil {
		return account, nil
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

const loginTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - Superplane</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            margin: 0;
            padding: 0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            background: white;
            padding: 2rem;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
            max-width: 400px;
            width: 90%;
        }
        .logo {
            font-size: 2rem;
            font-weight: bold;
            color: #333;
            margin-bottom: 0.5rem;
        }
        .subtitle {
            color: #666;
            margin-bottom: 2rem;
        }
        .login-btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            padding: 12px 24px;
            border-radius: 8px;
            text-decoration: none;
            font-weight: 500;
            transition: all 0.2s;
            width: 100%;
            box-sizing: border-box;
            margin-bottom: 12px;
        }
        .login-btn:last-child {
            margin-bottom: 0;
        }
        .login-btn.github {
            background: #24292e;
            color: white;
        }
        .login-btn.github:hover {
            background: #1a1e22;
        }
        .login-btn.google {
            background: #4285f4;
            color: white;
        }
        .login-btn.google:hover {
            background: #357ae8;
        }
        .provider-icon {
            width: 20px;
            height: 20px;
            margin-right: 8px;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="logo">Superplane</div>
        <div class="subtitle">Welcome back! Please sign in to continue.</div>
        
        {{range .Providers}}
        <a href="/auth/{{.}}" class="login-btn {{.}}">
            {{if eq . "github"}}
                <svg class="provider-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
                </svg>
                Continue with GitHub
            {{else if eq . "google"}}
                <svg class="provider-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                    <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                    <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                    <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
                </svg>
                Continue with Google
            {{end}}
        </a>
    {{end}}
    </div>
</body>
</html>
`
