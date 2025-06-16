package authentication

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type AuthenticationHandler struct {
	jwtSigner *jwt.Signer
	basePath  string
}

type AuthenticationUser struct {
	ID               string                   `json:"id"`
	Email            string                   `json:"email"`
	Name             string                   `json:"name"`
	AvatarURL        string                   `json:"avatar_url"`
	AccessToken      string                   `json:"access_token"`
	CreatedAt        time.Time                `json:"created_at"`
	AccountProviders []models.AccountProvider `json:"account_providers,omitempty"`
}

type ProviderConfig struct {
	Key         string
	Secret      string
	CallbackURL string
}

func NewAuthHandler(jwtSigner *jwt.Signer, basePath string) *AuthenticationHandler {
	return &AuthenticationHandler{
		jwtSigner: jwtSigner,
		basePath:  basePath,
	}
}

// InitializeProviders sets up the OAuth providers
func (a *AuthenticationHandler) InitializeProviders(providers map[string]ProviderConfig) {
	var gothProviders []goth.Provider

	for providerName, config := range providers {
		if config.Key == "" || config.Secret == "" {
			log.Warnf("%s OAuth not configured - missing key/secret", providerName)
			continue
		}

		switch providerName {
		case "github":
			gothProviders = append(gothProviders, github.New(config.Key, config.Secret, config.CallbackURL))
			log.Infof("GitHub OAuth provider initialized")
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

// RegisterRoutes adds authentication routes to the router
func (a *AuthenticationHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc(a.basePath+"/auth/me", a.handleMe).Methods("GET")
	router.HandleFunc(a.basePath+"/logout", a.handleLogout).Methods("GET")
	router.HandleFunc(a.basePath+"/login", a.handleLoginPage).Methods("GET")

	if os.Getenv("APP_ENV") == "development" {
		log.Info("Registering development authentication routes")
		// In dev: both auth and callback just auto-authenticate
		router.HandleFunc(a.basePath+"/auth/{provider}/callback", a.handleDevAuth).Methods("GET")
		router.HandleFunc(a.basePath+"/auth/{provider}", a.handleDevAuth).Methods("GET")
	} else {
		// Production OAuth routes
		router.HandleFunc(a.basePath+"/auth/{provider}/callback", a.handleAuthCallback).Methods("GET")
		router.HandleFunc(a.basePath+"/auth/{provider}", a.handleAuth).Methods("GET")
	}

	router.HandleFunc(a.basePath+"/auth/{provider}/disconnect", a.handleDisconnectProvider).Methods("POST")
}

func (a *AuthenticationHandler) handleAuth(w http.ResponseWriter, r *http.Request) {
	if gothUser, err := gothic.CompleteUserAuth(w, r); err == nil {
		log.Infof("User already authenticated: %s", gothUser.Email)
		a.handleSuccessfulAuth(w, r, gothUser)
	} else {
		log.Info("Starting OAuth flow")
		gothic.BeginAuthHandler(w, r)
	}
}

func (a *AuthenticationHandler) handleDevAuth(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("APP_ENV") != "development" {
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

	log.Infof("Development mode: auto-authenticating as %s via %s", mockUser.Email, provider)
	a.handleSuccessfulAuth(w, r, mockUser)
}

func (a *AuthenticationHandler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Errorf("Authentication error: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	log.Infof("Authentication successful for user: %s via %s", gothUser.Email, gothUser.Provider)
	a.handleSuccessfulAuth(w, r, gothUser)
}

func (a *AuthenticationHandler) handleSuccessfulAuth(w http.ResponseWriter, r *http.Request, gothUser goth.User) {
	dbUser, _, err := a.findOrCreateUserAndAccount(gothUser)
	if err != nil {
		log.Errorf("Error creating/finding user and account: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	token, err := a.jwtSigner.Generate(dbUser.ID.String(), 24*time.Hour)
	if err != nil {
		log.Errorf("Error generating JWT: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	callbackURL := r.URL.Query().Get("callback_url")
	state := r.URL.Query().Get("state")

	if callbackURL != "" {
		parsedURL, err := url.Parse(callbackURL)
		if err != nil {
			http.Error(w, "Invalid callback URL", http.StatusBadRequest)
			return
		}

		// Security check - only allow localhost callbacks
		if parsedURL.Hostname() != "localhost" && parsedURL.Hostname() != "127.0.0.1" {
			http.Error(w, "Invalid callback URL - only localhost allowed", http.StatusBadRequest)
			return
		}

		query := parsedURL.Query()
		query.Set("token", token)
		if state != "" {
			query.Set("state", state)
		}
		parsedURL.RawQuery = query.Encode()

		log.Infof("Redirecting CLI callback to: %s", parsedURL.String())
		http.Redirect(w, r, parsedURL.String(), http.StatusTemporaryRedirect)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(24 * time.Hour.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	if r.Header.Get("Accept") == "application/json" {
		accountProviders, _ := dbUser.GetAccountProviders()
		authUser := AuthenticationUser{
			ID:               dbUser.ID.String(),
			Email:            dbUser.Email,
			Name:             dbUser.Name,
			AvatarURL:        dbUser.AvatarURL,
			AccessToken:      token,
			CreatedAt:        dbUser.CreatedAt,
			AccountProviders: accountProviders,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(authUser)
	} else {
		http.Redirect(w, r, "/app", http.StatusTemporaryRedirect)
	}
}

func (a *AuthenticationHandler) handleDisconnectProvider(w http.ResponseWriter, r *http.Request) {
	user, err := a.getUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	provider := vars["provider"]

	accountProvider, err := user.GetAccountProvider(provider)
	if err != nil {
		http.Error(w, "Provider account not found", http.StatusNotFound)
		return
	}

	if err := accountProvider.Delete(); err != nil {
		log.Errorf("Error deleting repo host account: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Infof("User %s disconnected %s account", user.Email, provider)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Successfully disconnected %s account", provider),
	})
}

func (a *AuthenticationHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	gothic.Logout(w, r)

	// Clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	if r.Header.Get("Accept") == "application/json" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
	} else {
		http.Redirect(w, r, a.basePath+"/login", http.StatusTemporaryRedirect)
	}
}

func (a *AuthenticationHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := a.getUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accountProviders, err := user.GetAccountProviders()
	if err != nil {
		log.Errorf("Error getting repo host accounts: %v", err)
		accountProviders = []models.AccountProvider{}
	}

	authUser := AuthenticationUser{
		ID:               user.ID.String(),
		Email:            user.Email,
		Name:             user.Name,
		AvatarURL:        user.AvatarURL,
		CreatedAt:        user.CreatedAt,
		AccountProviders: accountProviders,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authUser)
}

func (a *AuthenticationHandler) handleLoginPage(w http.ResponseWriter, r *http.Request) {
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
		BasePath  string
		Providers []string
	}{
		BasePath:  a.basePath,
		Providers: providerNames,
	}

	t.Execute(w, data)
}

func (a *AuthenticationHandler) findOrCreateUserAndAccount(gothUser goth.User) (*models.User, *models.AccountProvider, error) {
	accountProvider, err := models.FindAccountProviderByProviderID(gothUser.Provider, gothUser.UserID)
	if err == nil {
		accountProvider.Username = gothUser.NickName
		accountProvider.Email = gothUser.Email
		accountProvider.Name = gothUser.Name
		accountProvider.AvatarURL = gothUser.AvatarURL
		accountProvider.AccessToken = gothUser.AccessToken
		accountProvider.RefreshToken = gothUser.RefreshToken
		if gothUser.ExpiresAt != (time.Time{}) {
			accountProvider.TokenExpiresAt = &gothUser.ExpiresAt
		}

		if err := accountProvider.Update(); err != nil {
			return nil, nil, err
		}

		user, err := models.FindUserByID(accountProvider.UserID.String())
		if err != nil {
			return nil, nil, err
		}

		user.Email = gothUser.Email
		user.Name = gothUser.Name
		user.AvatarURL = gothUser.AvatarURL
		user.Update()

		return user, accountProvider, nil
	}

	user, err := models.FindUserByEmail(gothUser.Email)
	if err != nil {
		user = &models.User{
			Email:     gothUser.Email,
			Name:      gothUser.Name,
			AvatarURL: gothUser.AvatarURL,
		}

		if err := user.Create(); err != nil {
			return nil, nil, err
		}
	} else {
		user.Name = gothUser.Name
		user.AvatarURL = gothUser.AvatarURL
		user.Update()
	}

	accountProvider = &models.AccountProvider{
		UserID:       user.ID,
		Provider:     gothUser.Provider,
		ProviderID:   gothUser.UserID,
		Username:     gothUser.NickName,
		Email:        gothUser.Email,
		Name:         gothUser.Name,
		AvatarURL:    gothUser.AvatarURL,
		AccessToken:  gothUser.AccessToken,
		RefreshToken: gothUser.RefreshToken,
	}

	if gothUser.ExpiresAt != (time.Time{}) {
		accountProvider.TokenExpiresAt = &gothUser.ExpiresAt
	}

	if err := accountProvider.Create(); err != nil {
		return nil, nil, err
	}

	return user, accountProvider, nil
}

func (a *AuthenticationHandler) getUserFromRequest(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie("auth_token")
	var token string

	if err == nil {
		token = cookie.Value
	} else {
		// Fallback to Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			return nil, fmt.Errorf("no authentication token provided")
		}

		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			return nil, fmt.Errorf("malformed authorization header")
		}
		token = authHeader[7:]
	}

	claims, err := a.jwtSigner.ValidateAndGetClaims(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	userID := claims["sub"].(string)
	user, err := models.FindUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	return user, nil
}

func (a *AuthenticationHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.getUserFromRequest(r)
		if err != nil {
			log.Errorf("User not found: %v", err)
			if r.Header.Get("Accept") == "application/json" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, a.basePath+"/login", http.StatusTemporaryRedirect)
			}
			return
		}
		log.Infof("User %s authenticated", user.Email)

		ctx := r.Context()
		ctx = SetUserInContext(ctx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
        .provider-icon {
            width: 20px;
            height: 20px;
            margin-right: 8px;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="logo">🛩️ Superplane</div>
        <div class="subtitle">Welcome back! Please sign in to continue.</div>
        
        {{range .Providers}}
        <a href="{{$.BasePath}}/auth/{{.}}" class="login-btn {{.}}">
            {{if eq . "github"}}
                <svg class="provider-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
                </svg>
                Continue with GitHub
            {{end}}
        </a>
    {{end}}
    </div>
</body>
</html>
`
