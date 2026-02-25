package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type contextKey string

const AccountContextKey contextKey = "account"
const UserContextKey contextKey = "user"
const OrganizationNotFoundError string = "organization_not_found_error"
const AccountNotFoundError string = "account_not_found_error"

var ownerSetupEnabled = os.Getenv("OWNER_SETUP_ENABLED") == "yes"

var (
	ownerSetupMu          sync.RWMutex
	ownerSetupNeededCache *bool
)

func OwnerSetupEnabled() bool {
	return ownerSetupEnabled
}

func IsOwnerSetupRequired() bool {
	if !ownerSetupEnabled {
		return false
	}

	ownerSetupMu.RLock()
	if ownerSetupNeededCache != nil {
		val := *ownerSetupNeededCache
		ownerSetupMu.RUnlock()
		return val
	}
	ownerSetupMu.RUnlock()

	var count int64

	err := database.Conn().
		Model(&models.User{}).
		Limit(1).
		Count(&count).
		Error

	needed := err == nil && count == 0

	ownerSetupMu.Lock()
	ownerSetupNeededCache = &needed
	ownerSetupMu.Unlock()

	return needed
}

func MarkOwnerSetupCompleted() {
	if !ownerSetupEnabled {
		return
	}

	ownerSetupMu.Lock()
	val := false
	ownerSetupNeededCache = &val
	ownerSetupMu.Unlock()
}

func AccountAuthMiddleware(jwtSigner *jwt.Signer) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsOwnerSetupRequired() {
				path := r.URL.Path

				if isAccountAPIPath(path) {
					w.Header().Set("X-Owner-Setup-Required", "true")
					http.Error(w, "Owner setup required", http.StatusConflict)
					return
				}

				// Allow the setup flow and static assets through without auth
				if isOwnerSetupAllowedPath(path) {
					next.ServeHTTP(w, r)
					return
				}

				http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
				return
			}

			// Allow login-related paths without authentication
			if strings.HasPrefix(r.URL.Path, "/login") {
				next.ServeHTTP(w, r)
				return
			}

			// Allow static assets and Vite dev server paths without authentication
			// These are needed for the React app to load in development
			path := r.URL.Path
			if strings.HasPrefix(path, "/@") || strings.HasPrefix(path, "/src/") || strings.HasPrefix(path, "/node_modules/") || strings.HasPrefix(path, "/assets/") {
				next.ServeHTTP(w, r)
				return
			}

			accountID, err := getAccountFromCookie(r, jwtSigner)
			if err != nil {
				if isAccountAPIPath(r.URL.Path) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				authentication.ClearAccountCookie(w, r)
				redirectToLoginWithOriginalURL(w, r)
				return
			}

			account, err := models.FindAccountByID(accountID)
			if err != nil {
				if isAccountAPIPath(r.URL.Path) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				authentication.ClearAccountCookie(w, r)
				redirectToLoginWithOriginalURL(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), AccountContextKey, account)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func OrganizationAuthMiddleware(jwtSigner *jwt.Signer) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			//
			// If the authorization header is used,
			// we expect a user API token.
			//
			if r.Header.Get("Authorization") != "" {
				user, err := authenticateUserByToken(r)
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				ctx := context.WithValue(r.Context(), UserContextKey, user)
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}

			//
			// Otherwise, we authenticate the account with the cookie,
			// and expect an organization ID in the header or query parameters.
			//
			user, err := authenticateUserByCookie(jwtSigner, r)
			if err != nil {
				if err.Error() == OrganizationNotFoundError {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}

				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func authenticateUserByToken(r *http.Request) (*models.User, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header not found")
	}

	headerParts := strings.Split(authHeader, "Bearer ")
	if len(headerParts) != 2 {
		return nil, fmt.Errorf("invalid authorization header")
	}

	hashedToken := crypto.HashToken(headerParts[1])
	return models.FindActiveUserByTokenHash(hashedToken)
}

func authenticateUserByCookie(jwtSigner *jwt.Signer, r *http.Request) (*models.User, error) {
	accountID, err := getAccountFromCookie(r, jwtSigner)
	if err != nil {
		return nil, errors.New(AccountNotFoundError)
	}

	organizationID := findOrganizationID(r)
	if organizationID == "" {
		return nil, errors.New(OrganizationNotFoundError)
	}

	account, err := models.FindAccountByID(accountID)
	if err != nil {
		return nil, errors.New(AccountNotFoundError)
	}

	user, err := models.FindActiveUserByEmail(organizationID, account.Email)
	if err != nil {
		return nil, errors.New(OrganizationNotFoundError)
	}

	return user, nil
}

func findOrganizationID(r *http.Request) string {
	organizationID := r.Header.Get("x-organization-id")
	if organizationID != "" {
		return organizationID
	}

	organizationID = r.URL.Query().Get("organization_id")
	if organizationID != "" {
		return organizationID
	}

	return ""
}

func GetAccountFromContext(ctx context.Context) (*models.Account, bool) {
	account, ok := ctx.Value(AccountContextKey).(*models.Account)
	return account, ok
}

func isOwnerSetupAllowedPath(path string) bool {
	if path == "/setup" || strings.HasPrefix(path, "/assets") {
		return true
	}

	// Allow Vite dev server and module paths when running the
	// owner setup flow in development/e2e so that the SPA can
	// load its JS bundles, HMR client, and dependencies.
	if strings.HasPrefix(path, "/@") || strings.HasPrefix(path, "/src/") || strings.HasPrefix(path, "/node_modules/") {
		return true
	}

	switch path {
	case "/favicon.ico", "/robots.txt", "/manifest.webmanifest":
		return true
	}

	return false
}

func isAccountAPIPath(path string) bool {
	switch path {
	case "/account", "/organizations":
		return true
	default:
		return strings.HasPrefix(path, "/api/v1/invite-links/")
	}
}

func getAccountFromCookie(r *http.Request, jwtSigner *jwt.Signer) (string, error) {
	cookie, err := r.Cookie("account_token")
	if err != nil {
		return "", fmt.Errorf("account token cookie not found")
	}

	claims, err := jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return "", fmt.Errorf("invalid account token: %v", err)
	}

	accountClaim, exists := claims["sub"]
	if !exists {
		return "", fmt.Errorf("account ID missing from token")
	}

	accountID, ok := accountClaim.(string)
	if !ok {
		return "", fmt.Errorf("invalid account ID in token")
	}

	return accountID, nil
}

func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}

func redirectToLoginWithOriginalURL(w http.ResponseWriter, r *http.Request) {
	redirectURL := url.QueryEscape(r.URL.RequestURI())
	loginURL := fmt.Sprintf("/login?redirect=%s", redirectURL)
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

func ResetOwnerSetupStateForTests() {
	ownerSetupEnabled = true

	ownerSetupMu.Lock()
	ownerSetupNeededCache = nil
	ownerSetupMu.Unlock()
}
