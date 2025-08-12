package authentication

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type Handler struct {
	jwtSigner            *jwt.Signer
	encryptor            crypto.Encryptor
	authorizationService authorization.Authorization
	isDev                bool
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
		jwtSigner:            jwtSigner,
		encryptor:            encryptor,
		authorizationService: authorizationService,
		isDev:                appEnv == "development",
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

func (a *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/logout", a.handleLogout).Methods("GET")
	router.HandleFunc("/login", a.handleLoginPage).Methods("GET")
	router.HandleFunc("/organization/select", a.handleOrganizationSelectionPage).Methods("GET")
	router.HandleFunc("/organization/create", a.handleCreateOrganization).Methods("POST")

	//
	// If we are running the application locally,
	// we provide handlers that auto-autenticate to
	// avoid having to authenticate with GitHub every time.
	//
	// TODO: there's probably a better to do this other than
	// changing the handler based on a configuration.
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
	//
	// Look for an organization parameter in the URL.
	// if not provided, the user is not logging into an specific organization.
	//
	orgName := r.URL.Query().Get("org")
	if orgName == "" {
		gothUser, err := gothic.CompleteUserAuth(w, r)
		if err == nil {
			a.handleSuccessfulAuth(w, r, gothUser, nil)
			return
		}

		gothic.BeginAuthHandler(w, r)
		return
	}

	// TODO: verify user is with the proper auth mechanism as specified in org

	organization, err := models.FindOrganizationByName(orgName)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
	}

	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err == nil {
		a.handleSuccessfulAuth(w, r, gothUser, organization)
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

	orgName := r.URL.Query().Get("org")
	if orgName == "" {
		a.handleSuccessfulAuth(w, r, mockUser, nil)
		return
	}

	// TODO: verify user is with the proper auth mechanism as specified in org

	organization, err := models.FindOrganizationByName(orgName)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
	}

	a.handleSuccessfulAuth(w, r, mockUser, organization)
}

func (a *Handler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	account, err := findOrCreateAccount(user.Name, user.Email)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = updateAccountProviders(a.encryptor, account, user)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	orgName := r.URL.Query().Get("org")
	if orgName == "" {
		a.handleSuccessfulAuth(w, r, user, nil)
		return
	}

	organization, err := models.FindOrganizationByName(orgName)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	a.handleSuccessfulAuth(w, r, user, organization)
}

func (a *Handler) handleSuccessfulAuth(w http.ResponseWriter, r *http.Request, gothUser goth.User, organization *models.Organization) {
	//
	// TODO: should we create an account if one doesn't exist here?
	// TODO: do we even need account records at all?
	//
	account, err := models.FindAccountByEmail(gothUser.Email)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	//
	// If no organization has been selected yet,
	// we generate a token for the account and redirect to the organization selection page.
	//
	if organization == nil {
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

		http.Redirect(w, r, "/organization/select", http.StatusTemporaryRedirect)
		return
	}

	//
	// TODO: instead of rejecting, we should probably redirect
	// the user to log in as the organization requests it.
	//
	if !organization.IsProviderAllowed(gothUser.Provider) {
		http.Error(w, fmt.Sprintf("%s authentication is not allowed for this organization", gothUser.Provider), http.StatusForbidden)
		return
	}

	user, err := models.FindUserByEmail(organization.ID.String(), gothUser.Email)

	//
	// If user already exists, this user is already a member of the organization,
	// so just generate the token for it, and redirect to the home page.
	//
	if err == nil {
		a.generateJWT(w, r, user, organization)
		return
	}

	//
	// The user is not part of the organization,
	// so we need to check if the user has been invited to join.
	//
	invitation, err := models.FindPendingInvitation(organization.ID.String(), gothUser.Email)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		user, err = models.CreateUser(organization.ID, account.ID, gothUser.Email, gothUser.Name)
		if err != nil {
			return err
		}

		return invitation.Accept()
	})

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	a.generateJWT(w, r, user, organization)
}

func (a *Handler) generateJWT(w http.ResponseWriter, r *http.Request, user *models.User, organization *models.Organization) {
	token, err := a.jwtSigner.GenerateWithClaims(
		user.ID.String(),
		24*time.Hour,
		map[string]any{
			"org": organization.ID.String(),
		},
	)

	if err != nil {
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

	//
	// TODO: the URL should <BASE_URL>/<ORG_NAME> or <BASE_URL>/<ORG_SHORT_ID>
	//
	http.Redirect(w, r, "/app", http.StatusTemporaryRedirect)
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

func (a *Handler) getAccountFromCookie(r *http.Request) (*models.Account, error) {
	cookie, err := r.Cookie("account_token")
	if err != nil {
		return nil, err
	}

	claims, err := a.jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return nil, err
	}

	accountClaim, exists := claims["sub"]
	if !exists {
		return nil, err
	}

	accountID, ok := accountClaim.(string)
	if !ok {
		return nil, err
	}

	account, err := models.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (a *Handler) handleOrganizationSelectionPage(w http.ResponseWriter, r *http.Request) {
	account, err := a.getAccountFromCookie(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	providers, err := account.GetAccountProviders()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	organizations, err := models.FindUserOrganizationsByEmail(account.Email)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	t, err := template.New("org-selection").Parse(organizationSelectionTemplate)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Name          string
		Email         string
		Provider      string
		Organizations []models.Organization
	}{
		Name:          account.Name,
		Email:         account.Email,
		Provider:      providers[0].Provider,
		Organizations: organizations,
	}

	t.Execute(w, data)
}

func (a *Handler) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	account, err := a.getAccountFromCookie(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	displayName := r.FormValue("display_name")
	description := r.FormValue("description")

	if name == "" || displayName == "" {
		http.Error(w, "Name, display name, and email are required", http.StatusBadRequest)
		return
	}

	//
	// TODO: the organization creation should be in a transaction
	//

	//
	// Create the organization and set up roles for it.
	//
	organization, err := models.CreateOrganization(name, displayName, description)
	if err != nil {
		log.Errorf("Error creating organization: %v", err)
		http.Error(w, "Failed to create organization", http.StatusInternalServerError)
		return
	}

	err = a.authorizationService.SetupOrganizationRoles(organization.ID.String())
	if err != nil {
		log.Errorf("Error setting up organization roles for %s: %v", organization.Name, err)
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to set up organization roles", http.StatusInternalServerError)
		return
	}

	//
	// Create the owner user for it
	//
	user, err := models.CreateUser(organization.ID, account.ID, account.Email, account.Name)
	if err != nil {
		log.Errorf("Error creating user for new organization: %v", err)
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to create user account", http.StatusInternalServerError)
		return
	}

	err = a.authorizationService.CreateOrganizationOwner(user.ID.String(), organization.ID.String())
	if err != nil {
		log.Errorf("Error creating organization owner for %s: %v", organization.Name, err)
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to create organization owner", http.StatusInternalServerError)
		return
	}

	redirectURL := fmt.Sprintf("/auth/%s?org=%s", organization.AllowedProviders[0], organization.Name)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (a *Handler) getUserFromRequest(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		return nil, fmt.Errorf("cookie not found")
	}

	claims, err := a.jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	userClaim, exists := claims["sub"]
	if !exists {
		return nil, fmt.Errorf("user ID missing from token")
	}

	userID, ok := userClaim.(string)
	if !ok {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	orgID, exists := claims["org"]
	if !exists {
		return nil, fmt.Errorf("organization context missing from token")
	}

	user, err := models.FindUserByID(orgID.(string), userID)
	if err != nil {
		return nil, fmt.Errorf("user not found in organization: %v", err)
	}

	return user, nil
}

func (a *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.getUserFromRequest(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		ctx := r.Context()
		ctx = SetUserInContext(ctx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

//
// TODO: move these templates to their own files
//

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
