package authentication

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type Handler struct {
	jwtSigner           *jwt.Signer
	encryptor           crypto.Encryptor
	authorizationService authorization.Authorization
	isDev               bool
}

type User struct {
	ID               string                   `json:"id"`
	OrganizationID   string                   `json:"organization_id"`
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
	GitHubToken      string `json:"github_token"`
	OrganizationName string `json:"organization_name"`
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

func NewHandler(jwtSigner *jwt.Signer, encryptor crypto.Encryptor, authorizationService authorization.Authorization, appEnv string) *Handler {
	return &Handler{
		jwtSigner:           jwtSigner,
		encryptor:           encryptor,
		authorizationService: authorizationService,
		isDev:               appEnv == "development",
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
		case models.ProviderGitHub:
			gothProviders = append(gothProviders, github.New(config.Key, config.Secret, config.CallbackURL, models.ScopeUser))
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
	router.HandleFunc("/login/{organization}", a.handleOrganizationLoginPage).Methods("GET")

	// Token exchange route for CLI
	router.HandleFunc("/auth/token/exchange", a.handleTokenExchange).Methods("POST")

	// OAuth routes - register these before more specific routes to avoid conflicts
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

	// Organization selection page (shows orgs + create option) - register after provider routes
	router.HandleFunc("/organization/select", a.handleOrganizationSelectionPage).Methods("GET")
	router.HandleFunc("/organization/create", a.handleCreateOrganization).Methods("POST")
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

	if req.OrganizationName == "" {
		http.Error(w, "Organization name is required", http.StatusBadRequest)
		return
	}

	// Find organization
	organization, err := models.FindOrganizationByName(req.OrganizationName)
	if err != nil {
		log.Errorf("Organization not found: %v", err)
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	githubUser, err := a.getGitHubUserInfo(req.GitHubToken)
	if err != nil {
		log.Errorf("Failed to get GitHub user info: %v", err)
		http.Error(w, "Invalid GitHub token", http.StatusUnauthorized)
		return
	}

	if !organization.IsProviderAllowed(models.ProviderGitHub) {
		http.Error(w, "GitHub authentication is not allowed for this organization", http.StatusForbidden)
		return
	}

	if !organization.IsEmailDomainAllowed(githubUser.Email) {
		http.Error(w, "Email domain not allowed for this organization", http.StatusForbidden)
		return
	}

	accountProvider, err := a.findOrCreateAccountProviderInOrganization(githubUser, organization.ID)
	if err != nil {
		log.Errorf("Failed to find or create account provider: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
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

	dbUser, err := models.FindUserByID(accountProvider.UserID, organization.ID)
	if err != nil {
		log.Errorf("Failed to find user by ID %s: %v", accountProvider.UserID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	dbUser.Name = githubUser.Name
	dbUser.IsActive = true // Activate user on login

	if err := dbUser.Update(); err != nil {
		log.Warnf("Failed to update user info: %v", err)
	}

	token, err := a.jwtSigner.GenerateWithClaims(dbUser.ID.String(), 24*time.Hour, map[string]any{
		"org": dbUser.OrganizationID.String(),
	})

	if err != nil {
		log.Errorf("Error generating JWT: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	accountProviders, err := dbUser.GetAccountProviders()

	if err != nil {
		log.Errorf("Failed to get account providers for user %s: %v", dbUser.ID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	authUser := User{
		ID:               dbUser.ID.String(),
		OrganizationID:   dbUser.OrganizationID.String(),
		Name:             dbUser.Name,
		Email:            getPrimaryEmail(accountProviders),
		AvatarURL:        getPrimaryAvatar(accountProviders),
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
	log.Infof("handleAuth called: URL=%s, Method=%s", r.URL.String(), r.Method)

	// Get organization from query parameter - if not provided, we'll handle organization selection after auth
	orgName := r.URL.Query().Get("org")
	var organization *models.Organization

	if orgName != "" {
		log.Infof("handleAuth: Found org parameter: %s", orgName)
		var err error
		organization, err = models.FindOrganizationByName(orgName)
		if err != nil {
			log.Errorf("handleAuth: Organization %s not found: %v", orgName, err)
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}

		// Store organization ID in session for callback
		r = r.WithContext(context.WithValue(r.Context(), "organization_id", organization.ID))

		// If we have an organization context, always start fresh OAuth flow
		// to ensure proper authentication for this specific organization
		log.Infof("handleAuth: Starting OAuth flow for organization: %s", orgName)
		gothic.BeginAuthHandler(w, r)
		return
	}

	log.Info("handleAuth: No org parameter, checking existing auth")
	// No organization context - check if user is already authenticated
	if gothUser, err := gothic.CompleteUserAuth(w, r); err == nil {
		log.Infof("handleAuth: User already authenticated: %s", gothUser.Email)
		a.handleSuccessfulAuth(w, r, gothUser, organization)
	} else {
		log.Infof("handleAuth: No existing auth, starting OAuth flow: %v", err)
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

	// Check for organization parameter in dev mode too
	orgName := r.URL.Query().Get("org")
	var organization *models.Organization

	if orgName != "" {
		log.Infof("handleDevAuth: Found org parameter: %s", orgName)
		var err error
		organization, err = models.FindOrganizationByName(orgName)
		if err != nil {
			log.Errorf("handleDevAuth: Organization %s not found: %v", orgName, err)
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
	}

	mockUser := goth.User{
		UserID:      "dev-user-123",
		Email:       "dev@superplane.local",
		Name:        "Dev User",
		NickName:    "devuser",
		Provider:    provider,
		AvatarURL:   "https://github.com/github.png",
		AccessToken: "dev-token-" + provider,
	}

	log.Infof("Development mode: auto-authenticating as %s via %s, org=%v", mockUser.Email, provider, organization)
	a.handleSuccessfulAuth(w, r, mockUser, organization)
}

func (a *Handler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Errorf("Authentication error: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Get organization from query parameter - if not provided, we'll handle organization selection
	orgName := r.URL.Query().Get("org")
	var organization *models.Organization

	if orgName != "" {
		organization, err = models.FindOrganizationByName(orgName)
		if err != nil {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		log.Infof("Authentication successful for user: %s via %s in organization: %s", gothUser.Email, gothUser.Provider, orgName)
	} else {
		log.Infof("Authentication successful for user: %s via %s (no org context)", gothUser.Email, gothUser.Provider)
	}

	a.handleSuccessfulAuth(w, r, gothUser, organization)
}

func (a *Handler) handleSuccessfulAuth(w http.ResponseWriter, r *http.Request, gothUser goth.User, organization *models.Organization) {
	log.Infof("handleSuccessfulAuth called: user=%s, org=%v", gothUser.Email, organization)

	// If no organization context, redirect to organization selection page
	if organization == nil {
		log.Info("handleSuccessfulAuth: No organization, redirecting to selection")
		a.handleOrganizationSelection(w, r, gothUser)
		return
	}

	if !organization.IsProviderAllowed(gothUser.Provider) {
		http.Error(w, fmt.Sprintf("%s authentication is not allowed for this organization", gothUser.Provider), http.StatusForbidden)
		return
	}

	if !organization.IsEmailDomainAllowed(gothUser.Email) {
		http.Error(w, "Email domain not allowed for this organization", http.StatusForbidden)
		return
	}

	dbUser, _, err := a.findOrCreateUserAndAccountInOrganization(gothUser, organization.ID)
	if err != nil {
		log.Errorf("Error creating/finding user and account: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userIDStr := dbUser.ID.String()
	orgIDStr := organization.ID.String()
	log.Infof("Generating JWT for user %s in org %s", userIDStr, orgIDStr)

	token, err := a.jwtSigner.GenerateWithClaims(userIDStr, 24*time.Hour, map[string]any{
		"org": orgIDStr,
	})

	if err != nil {
		log.Errorf("Error generating JWT: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Infof("Generated JWT token, length: %d", len(token))

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
		accountProviders, err := dbUser.GetAccountProviders()

		if err != nil {
			log.Errorf("Error getting account providers for user %s: %v", dbUser.ID.String(), err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		authUser := User{
			ID:               dbUser.ID.String(),
			Name:             dbUser.Name,
			Email:            getPrimaryEmail(accountProviders),
			AvatarURL:        getPrimaryAvatar(accountProviders),
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
		Email:            getPrimaryEmail(accountProviders),
		AvatarURL:        getPrimaryAvatar(accountProviders),
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

func (a *Handler) handleOrganizationLoginPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgName := vars["organization"]

	// Verify organization exists
	organization, err := models.FindOrganizationByName(orgName)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	t, err := template.New("org-login").Parse(organizationLoginTemplate)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		AllowedProviders []string
		OrganizationName string
		DisplayName      string
	}{
		AllowedProviders: organization.AllowedProviders,
		OrganizationName: orgName,
		DisplayName:      organization.DisplayName,
	}

	t.Execute(w, data)
}

func (a *Handler) handleOrganizationSelection(w http.ResponseWriter, r *http.Request, gothUser goth.User) {
	log.Infof("handleOrganizationSelection: User %s needs to select organization", gothUser.Email)
	redirectURL := fmt.Sprintf("/organization/select?email=%s&provider=%s&name=%s", gothUser.Email, gothUser.Provider, gothUser.Name)
	log.Infof("handleOrganizationSelection: Redirecting to %s", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (a *Handler) handleOrganizationSelectionPage(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "Missing user information", http.StatusBadRequest)
		return
	}

	// Look up organizations for this user
	organizations, err := models.FindUserOrganizationsByEmail(email)
	if err != nil {
		log.Errorf("Error finding organizations for user %s: %v", email, err)
		organizations = []models.Organization{}
	}

	t, err := template.New("org-selection").Parse(organizationSelectionTemplate)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Email         string
		Name          string
		Provider      string
		Organizations []models.Organization
	}{
		Email:         email,
		Name:          r.URL.Query().Get("name"),
		Provider:      r.URL.Query().Get("provider"),
		Organizations: organizations,
	}

	t.Execute(w, data)
}

func (a *Handler) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	displayName := r.FormValue("display_name")
	description := r.FormValue("description")
	email := r.FormValue("email")
	provider := r.FormValue("provider")
	userName := r.FormValue("user_name")

	if name == "" || displayName == "" || email == "" {
		http.Error(w, "Name, display name, and email are required", http.StatusBadRequest)
		return
	}

	organization, err := models.CreateOrganization(name, displayName, description)
	if err != nil {
		log.Errorf("Error creating organization: %v", err)
		http.Error(w, "Failed to create organization", http.StatusInternalServerError)
		return
	}

	// Create the user as the organization owner (but don't create account provider yet)
	// The account provider will be created during the OAuth flow
	user := &models.User{
		OrganizationID: organization.ID,
		Email:          email,
		Name:           userName,
		IsActive:       true, // User is active since they're creating the org
	}

	if err := user.Create(); err != nil {
		log.Errorf("Error creating user for new organization: %v", err)
		// Clean up the organization if user creation fails
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to create user account", http.StatusInternalServerError)
		return
	}

	// Set up organization roles
	err = a.authorizationService.SetupOrganizationRoles(organization.ID.String())
	if err != nil {
		log.Errorf("Error setting up organization roles for %s: %v", organization.Name, err)
		// Clean up the organization and user if role setup fails
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to set up organization roles", http.StatusInternalServerError)
		return
	}
	log.Infof("Set up organization roles for %s (%s)", organization.Name, organization.ID.String())

	// Create organization owner
	err = a.authorizationService.CreateOrganizationOwner(user.ID.String(), organization.ID.String())
	if err != nil {
		log.Errorf("Error creating organization owner for %s: %v", organization.Name, err)
		// Clean up the organization and user if owner creation fails
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to create organization owner", http.StatusInternalServerError)
		return
	}
	log.Infof("Created organization owner for %s (%s) for user %s", organization.Name, organization.ID.String(), user.ID.String())

	log.Infof("Created organization %s with owner %s (%s)", organization.Name, user.Name, user.Email)

	// Redirect to auth with the new organization
	redirectURL := fmt.Sprintf("/auth/%s?org=%s", provider, organization.Name)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (a *Handler) findOrCreateUserAndAccountInOrganization(gothUser goth.User, organizationID uuid.UUID) (*models.User, *models.AccountProvider, error) {
	accountProvider, err := models.FindAccountProviderByProviderIDInOrganization(gothUser.Provider, gothUser.UserID, organizationID)
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

		user, err := models.FindUserByID(accountProvider.UserID, organizationID)
		if err != nil {
			return nil, nil, err
		}

		user.Name = gothUser.Name
		user.IsActive = true
		user.Update()

		return user, accountProvider, nil
	}

	user, err := a.findOrCreateUserInOrganization(gothUser, organizationID)
	if err != nil {
		return nil, nil, err
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

	userIDClaim, exists := claims["sub"]
	if !exists {
		return nil, fmt.Errorf("user ID missing from token")
	}

	userID, ok := userIDClaim.(string)
	if !ok {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	// Get organization ID from claims (required)
	orgClaim, exists := claims["org"]
	if !exists {
		return nil, fmt.Errorf("organization context missing from token")
	}

	orgIDStr, ok := orgClaim.(string)
	if !ok {
		return nil, fmt.Errorf("invalid organization context in token")
	}

	organizationID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID in token: %v", err)
	}

	user, err := models.FindUserByID(uuid.MustParse(userID), organizationID)
	if err != nil {
		return nil, fmt.Errorf("user not found in organization: %v", err)
	}

	return user, nil
}

func getPrimaryEmail(accountProviders []models.AccountProvider) string {
	if len(accountProviders) > 0 {
		return accountProviders[0].Email
	}
	return ""
}

func getPrimaryAvatar(accountProviders []models.AccountProvider) string {
	if len(accountProviders) > 0 {
		return accountProviders[0].AvatarURL
	}
	return ""
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

func (a *Handler) findOrCreateAccountProviderInOrganization(githubUser *GitHubUserInfo, organizationID uuid.UUID) (*models.AccountProvider, error) {
	accountProvider, err := models.FindAccountProviderByProviderIDInOrganization(models.ProviderGitHub, fmt.Sprintf("%d", githubUser.ID), organizationID)
	if err == nil {
		return accountProvider, nil
	}

	// Check if user exists in organization
	user, err := models.FindUserByEmail(githubUser.Email, organizationID)
	if err != nil {
		// Check for inactive invited user
		user, err = models.FindInactiveUserByEmail(githubUser.Email, organizationID)
		if err != nil {
			// Check for pending invitation and create user
			invitation, err := models.FindPendingInvitation(githubUser.Email, organizationID)
			if err != nil {
				return nil, fmt.Errorf("No invitation found for this email in the organization. Please contact an administrator")
			}

			// Check if invitation is expired
			if invitation.IsExpired() {
				invitation.Expire()
				return nil, fmt.Errorf("Invitation has expired. Please request a new invitation")
			}

			// Create user account from invitation
			user = &models.User{
				OrganizationID: organizationID,
				Email:          githubUser.Email,
				Name:           githubUser.Name,
				IsActive:       true, // Activate immediately upon successful auth
			}

			if err := user.Create(); err != nil {
				return nil, fmt.Errorf("Failed to create user account: %v", err)
			}

			// Mark invitation as accepted
			if err := invitation.Accept(); err != nil {
				log.Errorf("Failed to mark invitation as accepted: %v", err)
			}

			log.Infof("Created user account for %s in organization %s via invitation", githubUser.Email, organizationID)
		} else {
			// Accept any pending invitation for existing inactive user
			a.acceptPendingInvitation(githubUser.Email, organizationID)
		}
	}

	// Create account provider for existing user
	accountProvider = &models.AccountProvider{
		UserID:     user.ID,
		Provider:   models.ProviderGitHub,
		ProviderID: fmt.Sprintf("%d", githubUser.ID),
		Username:   githubUser.Login,
		Email:      githubUser.Email,
		Name:       githubUser.Name,
		AvatarURL:  githubUser.AvatarURL,
	}

	if err := accountProvider.Create(); err != nil {
		return nil, fmt.Errorf("Internal server error")
	}
	return accountProvider, nil
}

func (a *Handler) findOrCreateUserInOrganization(gothUser goth.User, organizationID uuid.UUID) (*models.User, error) {
	// Try to find existing user by email in organization
	user, err := models.FindUserByEmail(gothUser.Email, organizationID)
	if err == nil {
		if err := a.updateUserInfo(user, gothUser.Name); err != nil {
			return nil, err
		}
		return user, nil
	}

	// Try to find inactive (invited) user
	user, err = models.FindInactiveUserByEmail(gothUser.Email, organizationID)
	if err == nil {
		// Accept any pending invitation for this email
		a.acceptPendingInvitation(gothUser.Email, organizationID)

		if err := a.updateUserInfo(user, gothUser.Name); err != nil {
			return nil, err
		}
		return user, nil
	}

	// Check for pending invitation and create user
	invitation, err := models.FindPendingInvitation(gothUser.Email, organizationID)
	if err != nil {
		return nil, fmt.Errorf("No invitation found for this email in the organization. Please contact an administrator")
	}

	// Check if invitation is expired
	if invitation.IsExpired() {
		invitation.Expire()
		return nil, fmt.Errorf("Invitation has expired. Please request a new invitation")
	}

	// Create user account from invitation
	user = &models.User{
		OrganizationID: organizationID,
		Email:          gothUser.Email,
		Name:           gothUser.Name,
		IsActive:       true, // Activate immediately upon successful auth
	}

	if err := user.Create(); err != nil {
		return nil, fmt.Errorf("Failed to create user account: %v", err)
	}

	// Mark invitation as accepted
	if err := invitation.Accept(); err != nil {
		log.Errorf("Failed to mark invitation as accepted: %v", err)
		// Don't fail the auth process for this
	}

	log.Infof("Created user account for %s in organization %s via invitation", gothUser.Email, organizationID)
	return user, nil
}

func (a *Handler) acceptPendingInvitation(email string, organizationID uuid.UUID) {
	invitation, err := models.FindPendingInvitation(email, organizationID)
	if err != nil {
		return // No pending invitation found
	}

	if !invitation.IsExpired() {
		if err := invitation.Accept(); err != nil {
			log.Errorf("Failed to accept invitation: %v", err)
		} else {
			log.Infof("Automatically accepted invitation for %s in organization %s", email, organizationID)
		}
	}
}

func (a *Handler) updateUserInfo(user *models.User, name string) error {
	user.Name = name
	user.IsActive = true
	return user.Update()
}

const organizationLoginTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sign in to {{.DisplayName}} - Superplane</title>
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
        .org-name {
            font-size: 1.2rem;
            color: #666;
            margin-bottom: 0.5rem;
        }
        .subtitle {
            color: #888;
            margin-bottom: 2rem;
            font-size: 0.9rem;
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
        <div class="org-name">{{.DisplayName}}</div>
        <div class="subtitle">Sign in to your organization</div>
        
        {{range .AllowedProviders}}
        <a href="/auth/{{.}}?org={{$.OrganizationName}}" class="login-btn {{.}}">
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

const organizationSelectionTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Select Organization - Superplane</title>
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
        .container {
            background: white;
            padding: 2rem;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            max-width: 500px;
            width: 90%;
        }
        .header {
            text-align: center;
            margin-bottom: 2rem;
        }
        .logo {
            font-size: 2rem;
            font-weight: bold;
            color: #333;
            margin-bottom: 0.5rem;
        }
        .welcome {
            color: #666;
            margin-bottom: 1rem;
        }
        .section {
            margin-bottom: 2rem;
        }
        .section h3 {
            margin: 0 0 1rem 0;
            color: #333;
        }
        .org-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 1rem;
            border: 1px solid #e1e5e9;
            border-radius: 8px;
            margin-bottom: 0.5rem;
            background: #f8f9fa;
        }
        .org-info {
            flex: 1;
        }
        .org-name {
            font-weight: 600;
            color: #333;
        }
        .org-desc {
            font-size: 0.9rem;
            color: #666;
            margin-top: 0.25rem;
        }
        .btn {
            padding: 8px 16px;
            border-radius: 6px;
            border: none;
            cursor: pointer;
            font-weight: 500;
            text-decoration: none;
            display: inline-block;
            transition: all 0.2s;
        }
        .btn-primary {
            background: #007bff;
            color: white;
        }
        .btn-primary:hover {
            background: #0056b3;
        }
        .btn-secondary {
            background: #6c757d;
            color: white;
        }
        .btn-secondary:hover {
            background: #545b62;
        }
        .form-group {
            margin-bottom: 1rem;
        }
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            color: #333;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 1px solid #ddd;
            border-radius: 6px;
            font-size: 1rem;
            box-sizing: border-box;
        }
        .create-form {
            display: none;
            padding-top: 1rem;
            border-top: 1px solid #e1e5e9;
        }
        .no-orgs {
            text-align: center;
            color: #666;
            padding: 2rem;
            background: #f8f9fa;
            border-radius: 8px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">🛩️ Superplane</div>
            <div class="welcome">Welcome{{if .Name}}, {{.Name}}{{end}}!</div>
            <p>Choose an organization to continue, or create a new one.</p>
        </div>

        {{if .Organizations}}
        <div class="section">
            <h3>Your Organizations</h3>
            {{range .Organizations}}
            <div class="org-item">
                <div class="org-info">
                    <div class="org-name">{{.DisplayName}}</div>
                    <div class="org-desc">{{if .Description}}{{.Description}}{{else}}{{.Name}}{{end}}</div>
                </div>
                <a href="/auth/{{$.Provider}}?org={{.Name}}" class="btn btn-primary">Join</a>
            </div>
            {{end}}
        </div>
        {{else}}
        <div class="section">
            <div class="no-orgs">
                <p>You're not a member of any organizations yet.</p>
                <p>Create a new organization to get started!</p>
            </div>
        </div>
        {{end}}

        <div class="section">
            <button id="create-btn" class="btn btn-secondary" onclick="toggleCreateForm()">Create New Organization</button>
            
            <div id="create-form" class="create-form">
                <form action="/organization/create" method="post">
                    <input type="hidden" name="email" value="{{.Email}}">
                    <input type="hidden" name="provider" value="{{.Provider}}">
                    <input type="hidden" name="user_name" value="{{.Name}}">
                    <div class="form-group">
                        <label for="org-name">Organization Name</label>
                        <input type="text" id="org-name" name="name" placeholder="my-company" required>
                        <small style="color: #666;">Lowercase letters, numbers, and hyphens only</small>
                    </div>
                    <div class="form-group">
                        <label for="org-display-name">Display Name</label>
                        <input type="text" id="org-display-name" name="display_name" placeholder="My Company" required>
                    </div>
                    <div class="form-group">
                        <label for="org-description">Description (Optional)</label>
                        <input type="text" id="org-description" name="description" placeholder="Brief description of your organization">
                    </div>
                    <button type="submit" class="btn btn-primary">Create Organization</button>
                    <button type="button" class="btn btn-secondary" onclick="toggleCreateForm()">Cancel</button>
                </form>
            </div>
        </div>
    </div>

    <script>
        function toggleCreateForm() {
            const form = document.getElementById('create-form');
            const btn = document.getElementById('create-btn');
            if (form.style.display === 'none' || !form.style.display) {
                form.style.display = 'block';
                btn.textContent = 'Cancel';
            } else {
                form.style.display = 'none';
                btn.textContent = 'Create New Organization';
            }
        }
    </script>
</body>
</html>
`

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
