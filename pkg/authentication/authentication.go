package authentication

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type Handler struct {
	jwtSigner *jwt.Signer
	encryptor crypto.Encryptor
	isDev     bool
}

type User struct {
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

type TokenExchangeRequest struct {
	GitHubToken string `json:"github_token"`
}

type TokenExchangeResponse struct {
	AccessToken string `json:"access_token"`
	User        User   `json:"user"`
}

type GitHubUserInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func NewHandler(jwtSigner *jwt.Signer, encryptor crypto.Encryptor, appEnv string) *Handler {
	return &Handler{
		jwtSigner: jwtSigner,
		encryptor: encryptor,
		isDev:     appEnv == "development",
	}
}

// InitializeProviders sets up the OAuth providers
func (a *Handler) InitializeProviders(providers map[string]ProviderConfig) {
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
func (a *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/auth/me", a.handleMe).Methods("GET")
	router.HandleFunc("/logout", a.handleLogout).Methods("GET")
	router.HandleFunc("/login", a.handleLoginPage).Methods("GET")

	// Token exchange route for CLI
	router.HandleFunc("/auth/token/exchange", a.handleTokenExchange).Methods("POST")

	if a.isDev {
		log.Info("Registering development authentication routes")
		// In dev: both auth and callback just auto-authenticate
		router.HandleFunc("/auth/{provider}/callback", a.handleDevAuth).Methods("GET")
		router.HandleFunc("/auth/{provider}", a.handleDevAuth).Methods("GET")
	} else {
		// Production OAuth routes
		router.HandleFunc("/auth/{provider}/callback", a.handleAuthCallback).Methods("GET")
		router.HandleFunc("/auth/{provider}", a.handleAuth).Methods("GET")
	}

	router.HandleFunc("/auth/{provider}/disconnect", a.handleDisconnectProvider).Methods("POST")
}

func (a *Handler) handleTokenExchange(w http.ResponseWriter, r *http.Request) {
	var req TokenExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.GitHubToken == "" {
		http.Error(w, "GitHub token is required", http.StatusBadRequest)
		return
	}

	githubUser, err := a.getGitHubUserInfo(req.GitHubToken)
	if err != nil {
		log.Errorf("Failed to get GitHub user info: %v", err)
		http.Error(w, "Invalid GitHub token", http.StatusUnauthorized)
		return
	}

	accountProvider, err := models.FindAccountProviderByProviderID("github", fmt.Sprintf("%d", githubUser.ID))
	if err != nil {
		// If not found by provider ID, try to find by email
		user, err := models.FindUserByEmail(githubUser.Email)
		if err != nil {
			log.Errorf("No existing account found for GitHub user %s (%s)", githubUser.Login, githubUser.Email)
			http.Error(w, "No existing account found. Please sign up through the web interface first.", http.StatusNotFound)
			return
		}

		accountProvider, err = user.GetAccountProvider("github")
		if err != nil {
			log.Errorf("User %s exists but has no GitHub account provider", githubUser.Email)
			http.Error(w, "Account exists but GitHub provider not connected. Please connect GitHub through the web interface.", http.StatusNotFound)
			return
		}
	}

	encryptedAccessToken, err := a.encryptor.Encrypt(context.Background(), []byte(req.GitHubToken), []byte(githubUser.Email))
	if err != nil {
		log.Errorf("Failed to encrypt access token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	accountProvider.AccessToken = base64.StdEncoding.EncodeToString(encryptedAccessToken)

	accountProvider.Username = githubUser.Login
	accountProvider.Email = githubUser.Email
	accountProvider.Name = githubUser.Name
	accountProvider.AvatarURL = githubUser.AvatarURL

	if err := accountProvider.Update(); err != nil {
		log.Errorf("Failed to update account provider: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	dbUser, err := models.FindUserByID(accountProvider.UserID.String())
	if err != nil {
		log.Errorf("Failed to find user by ID %s: %v", accountProvider.UserID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	dbUser.Name = githubUser.Name

	if err := dbUser.Update(); err != nil {
		log.Warnf("Failed to update user info: %v", err)
	}

	token, err := a.jwtSigner.Generate(dbUser.ID.String(), 24*time.Hour)
	if err != nil {
		log.Errorf("Error generating JWT: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	accountProviders, _ := dbUser.GetAccountProviders()
	authUser := User{
		ID:               dbUser.ID.String(),
		Name:             dbUser.Name,
		CreatedAt:        dbUser.CreatedAt,
		AccountProviders: accountProviders,
	}

	response := TokenExchangeResponse{
		AccessToken: token,
		User:        authUser,
	}

	log.Infof("Token exchange successful for user %s (%s)", dbUser.Name, dbUser.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (a *Handler) handleAuth(w http.ResponseWriter, r *http.Request) {
	if gothUser, err := gothic.CompleteUserAuth(w, r); err == nil {
		log.Infof("User already authenticated: %s", gothUser.Email)
		a.handleSuccessfulAuth(w, r, gothUser)
	} else {
		log.Info("Starting OAuth flow")
		gothic.BeginAuthHandler(w, r)
	}
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

	log.Infof("Development mode: auto-authenticating as %s via %s", mockUser.Email, provider)
	a.handleSuccessfulAuth(w, r, mockUser)
}

func (a *Handler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Errorf("Authentication error: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	log.Infof("Authentication successful for user: %s via %s", gothUser.Email, gothUser.Provider)
	a.handleSuccessfulAuth(w, r, gothUser)
}

func (a *Handler) handleSuccessfulAuth(w http.ResponseWriter, r *http.Request, gothUser goth.User) {
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
		authUser := User{
			ID:               dbUser.ID.String(),
			Name:             dbUser.Name,
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

func (a *Handler) handleDisconnectProvider(w http.ResponseWriter, r *http.Request) {
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
		log.Errorf("Error deleting account provider: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Infof("User %s disconnected %s account", user.ID, provider)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Successfully disconnected %s account", provider),
	})
}

func (a *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
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
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	}
}

func (a *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := a.getUserFromRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accountProviders, err := user.GetAccountProviders()
	if err != nil {
		log.Errorf("Error getting account providers: %v", err)
		accountProviders = []models.AccountProvider{}
	}

	authUser := User{
		ID:               user.ID.String(),
		Name:             user.Name,
		CreatedAt:        user.CreatedAt,
		AccountProviders: accountProviders,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authUser)
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

func (a *Handler) findOrCreateUserAndAccount(gothUser goth.User) (*models.User, *models.AccountProvider, error) {
	accountProvider, err := models.FindAccountProviderByProviderID(gothUser.Provider, gothUser.UserID)
	if err == nil {

		encryptedAccessToken, err := a.encryptor.Encrypt(context.Background(), []byte(gothUser.AccessToken), []byte(gothUser.Email))
		if err != nil {
			return nil, nil, err
		}

		accountProvider.AccessToken = base64.StdEncoding.EncodeToString(encryptedAccessToken)
		accountProvider.Username = gothUser.NickName
		accountProvider.Email = gothUser.Email
		accountProvider.Name = gothUser.Name
		accountProvider.AvatarURL = gothUser.AvatarURL
		accountProvider.RefreshToken = gothUser.RefreshToken
		if !gothUser.ExpiresAt.IsZero() {
			accountProvider.TokenExpiresAt = &gothUser.ExpiresAt
		}

		if err := accountProvider.Update(); err != nil {
			return nil, nil, err
		}

		user, err := models.FindUserByID(accountProvider.UserID.String())
		if err != nil {
			return nil, nil, err
		}

		user.Name = gothUser.Name
		user.Update()

		return user, accountProvider, nil
	}

	user, err := models.FindUserByEmail(gothUser.Email)
	if err != nil {
		user = &models.User{
			Name: gothUser.Name,
		}

		if err := user.Create(); err != nil {
			return nil, nil, err
		}
	} else {
		user.Name = gothUser.Name
		user.Update()
	}

	encryptedAccessToken, err := a.encryptor.Encrypt(context.Background(), []byte(gothUser.AccessToken), []byte(gothUser.Email))
	if err != nil {
		return nil, nil, err
	}

	accountProvider = &models.AccountProvider{
		UserID:       user.ID,
		Provider:     gothUser.Provider,
		ProviderID:   gothUser.UserID,
		Username:     gothUser.NickName,
		Email:        gothUser.Email,
		Name:         gothUser.Name,
		AvatarURL:    gothUser.AvatarURL,
		AccessToken:  base64.StdEncoding.EncodeToString(encryptedAccessToken),
		RefreshToken: gothUser.RefreshToken,
	}

	if !gothUser.ExpiresAt.IsZero() {
		accountProvider.TokenExpiresAt = &gothUser.ExpiresAt
	}

	if err := accountProvider.Create(); err != nil {
		return nil, nil, err
	}

	return user, accountProvider, nil
}

func (a *Handler) getUserFromRequest(r *http.Request) (*models.User, error) {
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

func (a *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.getUserFromRequest(r)
		if err != nil {
			log.Errorf("User not found: %v", err)
			if r.Header.Get("Accept") == "application/json" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			}
			return
		}

		ctx := r.Context()
		ctx = SetUserInContext(ctx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Handler) getGitHubUserInfo(token string) (*GitHubUserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "superplane-server/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var user GitHubUserInfo
	err = json.NewDecoder(resp.Body).Decode(&user)
	return &user, err
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
        <a href="/auth/{{.}}" class="login-btn {{.}}">
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
