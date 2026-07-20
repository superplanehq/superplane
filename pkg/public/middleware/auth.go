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
	"github.com/superplanehq/superplane/pkg/impersonation"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

type contextKey string

const AccountContextKey contextKey = "account"
const EffectiveAccountContextKey contextKey = "effective_account"
const UserContextKey contextKey = "user"
const ImpersonationContextKey contextKey = "impersonation"
const ScopedTokenClaimsContextKey contextKey = "scopedTokenClaims"
const OrganizationNotFoundError string = "organization_not_found_error"
const AccountNotFoundError string = "account_not_found_error"

// ImpersonationInfo is stored in the request context when an admin
// is impersonating another user.
type ImpersonationInfo struct {
	AdminAccountID string
	Active         bool
	UserName       string
}

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

			account, err := getValidatedAccountFromCookie(r, jwtSigner)
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

			// If there's a valid impersonation session, resolve the
			// impersonated user's account so that non-admin handlers
			// (getAccount, listOrganizations, etc.) see the target
			// user's data instead of the admin's.
			if impAccount, info := resolveImpersonatedAccount(jwtSigner, r, account); impAccount != nil {
				ctx = context.WithValue(ctx, EffectiveAccountContextKey, impAccount)
				ctx = context.WithValue(ctx, ImpersonationContextKey, info)
			}

			authentication.MaybeRefreshAccountSession(w, r, jwtSigner, account)

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// resolveImpersonatedAccount checks for an impersonation cookie and,
// if valid, returns the impersonated Account so it can be used as the
// effective account for non-admin endpoints.
func resolveImpersonatedAccount(jwtSigner *jwt.Signer, r *http.Request, admin *models.Account) (*models.Account, *ImpersonationInfo) {
	tokenStr, err := impersonation.ReadCookie(r)
	if err != nil {
		return nil, nil
	}

	claims, err := impersonation.ValidateToken(jwtSigner, tokenStr)
	if err != nil {
		return nil, nil
	}

	if claims.AdminAccountID != admin.ID.String() || !admin.IsInstallationAdmin() {
		return nil, nil
	}

	if !admin.IsSessionFresh(claims.IssuedAt) {
		return nil, nil
	}

	impAccount, err := models.FindAccountByID(claims.ImpersonatedAccountID)
	if err != nil {
		return nil, nil
	}

	info := &ImpersonationInfo{
		AdminAccountID: claims.AdminAccountID,
		Active:         true,
		UserName:       impAccount.Name,
	}

	return impAccount, info
}

func OrganizationAuthMiddleware(jwtSigner *jwt.Signer) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := telemetry.StartSpan(r.Context(), "auth.organization_auth")
			defer span.End()

			//
			// If the authorization header is used,
			// we expect a user API token.
			//
			if r.Header.Get("Authorization") != "" {
				user, scopedClaims, err := authenticateUserByToken(ctx, r, jwtSigner)

				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				ctx = context.WithValue(ctx, UserContextKey, user)
				if scopedClaims != nil {
					ctx = context.WithValue(ctx, ScopedTokenClaimsContextKey, scopedClaims)
				}

				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}

			//
			// Otherwise, we authenticate the account with the cookie,
			// and expect an organization ID in the header or query parameters.
			//
			user, impersonationInfo, err := authenticateUserByCookie(ctx, jwtSigner, r)
			if err != nil {
				if err.Error() == OrganizationNotFoundError {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}

				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx = context.WithValue(ctx, UserContextKey, user)
			if impersonationInfo != nil {
				ctx = context.WithValue(ctx, ImpersonationContextKey, impersonationInfo)
			}

			if account, err := getValidatedAccountFromCookie(r, jwtSigner); err == nil {
				authentication.MaybeRefreshAccountSession(w, r, jwtSigner, account)
			}

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func authenticateUserByToken(ctx context.Context, r *http.Request, jwtSigner *jwt.Signer) (*models.User, *jwt.ScopedTokenClaims, error) {
	ctx, span := telemetry.StartSpan(ctx, "auth.authenticate_by_token")
	defer span.End()

	token, err := getBearerToken(r)
	if err != nil {
		return nil, nil, err
	}

	//
	// Try to authenticate the token as if it was a scoped-token first.
	//
	user, scopedClaims, err := authenticateUserByScopedToken(ctx, token, jwtSigner)
	if err == nil {
		return user, scopedClaims, nil
	}

	hashedToken := crypto.HashToken(token)
	user, err = models.FindActiveUserByTokenHashInTransaction(database.DB(ctx), hashedToken)
	if err != nil {
		return nil, nil, err
	}
	if user.IsExpiredAPIKey() {
		return nil, nil, fmt.Errorf("API key token expired")
	}

	return user, nil, nil
}

func getBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header not found")
	}

	headerParts := strings.Split(authHeader, "Bearer ")
	if len(headerParts) != 2 {
		return "", fmt.Errorf("invalid authorization header")
	}

	return strings.TrimSpace(headerParts[1]), nil
}

func authenticateUserByScopedToken(ctx context.Context, token string, jwtSigner *jwt.Signer) (*models.User, *jwt.ScopedTokenClaims, error) {
	claims, err := jwtSigner.ValidateScopedToken(token)
	if err != nil {
		return nil, nil, err
	}

	user, err := models.FindActiveUserByIDInTransaction(database.DB(ctx), claims.OrgID, claims.Subject)
	if err != nil {
		return nil, nil, err
	}

	// Reject scoped tokens minted before the owning account's most recent
	// password change so a password rotation also kills programmatic
	// credentials issued for that user. API keys have no owning
	// human account, so they're unaffected.
	if user.AccountID != nil && claims.IssuedAt != nil {
		account, err := models.FindAccountByID(user.AccountID.String())
		if err != nil {
			return nil, nil, err
		}

		if !account.IsSessionFresh(claims.IssuedAt.Unix()) {
			return nil, nil, fmt.Errorf("scoped token invalidated by password change")
		}
	}

	return user, claims, nil
}

func authenticateUserByCookie(ctx context.Context, jwtSigner *jwt.Signer, r *http.Request) (*models.User, *ImpersonationInfo, error) {
	ctx, span := telemetry.StartSpan(ctx, "auth.authenticate_by_cookie")
	defer span.End()

	// If a valid impersonation session exists, commit to it — never fall
	// through to the admin's own identity. This prevents silently showing
	// admin data when the impersonated user isn't in the requested org.
	user, info, err := resolveImpersonatedUser(ctx, jwtSigner, r)
	if err == nil {
		return user, info, nil
	}
	if isActiveImpersonation(jwtSigner, r) {
		return nil, nil, err
	}

	account, err := getValidatedAccountFromCookie(r, jwtSigner)
	if err != nil {
		return nil, nil, errors.New(AccountNotFoundError)
	}

	organizationID := findOrganizationID(r)
	if organizationID == "" {
		return nil, nil, errors.New(OrganizationNotFoundError)
	}

	user, err = models.FindActiveUserByEmailInTransaction(database.DB(ctx), organizationID, account.Email)
	if err != nil {
		return nil, nil, errors.New(OrganizationNotFoundError)
	}

	return user, nil, nil
}

// isActiveImpersonation returns true if there's a valid, non-expired
// impersonation token belonging to the current admin. Used to decide
// whether to commit to impersonation or fall through to normal auth.
func isActiveImpersonation(jwtSigner *jwt.Signer, r *http.Request) bool {
	tokenStr, err := impersonation.ReadCookie(r)
	if err != nil {
		return false
	}

	claims, err := impersonation.ValidateToken(jwtSigner, tokenStr)
	if err != nil {
		return false
	}

	admin, err := getValidatedAccountFromCookie(r, jwtSigner)
	if err != nil {
		return false
	}

	if claims.AdminAccountID != admin.ID.String() {
		return false
	}

	if !admin.IsInstallationAdmin() {
		return false
	}

	if !admin.IsSessionFresh(claims.IssuedAt) {
		return false
	}

	return true
}

// resolveImpersonatedUser checks if there's a valid impersonation session.
// It validates the impersonation token AND the admin's account token, then
// finds the impersonated user in the organization from the request header.
func resolveImpersonatedUser(ctx context.Context, jwtSigner *jwt.Signer, r *http.Request) (*models.User, *ImpersonationInfo, error) {
	tokenStr, err := impersonation.ReadCookie(r)
	if err != nil {
		return nil, nil, fmt.Errorf("no impersonation cookie")
	}

	claims, err := impersonation.ValidateToken(jwtSigner, tokenStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid impersonation token: %w", err)
	}

	// Double-validate: the admin's regular session must also be valid
	admin, err := getValidatedAccountFromCookie(r, jwtSigner)
	if err != nil {
		return nil, nil, fmt.Errorf("admin session invalid: %w", err)
	}

	if admin.ID.String() != claims.AdminAccountID {
		return nil, nil, fmt.Errorf("admin account mismatch")
	}

	// Verify the admin is still an installation admin
	if !admin.IsInstallationAdmin() {
		return nil, nil, fmt.Errorf("admin account no longer valid")
	}

	// Reject impersonation tokens minted before the admin's most recent
	// password change so a password rotation also kills impersonation
	// sessions opened from the same browser.
	if !admin.IsSessionFresh(claims.IssuedAt) {
		return nil, nil, fmt.Errorf("impersonation token invalidated by admin password change")
	}

	// Look up the impersonated account
	impAccount, err := models.FindAccountByID(claims.ImpersonatedAccountID)
	if err != nil {
		return nil, nil, fmt.Errorf("impersonated account not found: %w", err)
	}

	// Find the user in the org from the request header
	organizationID := findOrganizationID(r)
	if organizationID == "" {
		return nil, nil, errors.New(OrganizationNotFoundError)
	}

	user, err := models.FindActiveUserByEmailInTransaction(database.DB(ctx), organizationID, impAccount.Email)
	if err != nil {
		return nil, nil, errors.New(OrganizationNotFoundError)
	}

	info := &ImpersonationInfo{
		AdminAccountID: claims.AdminAccountID,
		Active:         true,
		UserName:       impAccount.Name,
	}

	return user, info, nil
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

// GetEffectiveAccountFromContext returns the impersonated account when
// an impersonation session is active, otherwise the real account.
// Use this in handlers that should reflect the impersonated user's data.
func GetEffectiveAccountFromContext(ctx context.Context) (*models.Account, bool) {
	if eff, ok := ctx.Value(EffectiveAccountContextKey).(*models.Account); ok {
		return eff, true
	}
	return GetAccountFromContext(ctx)
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
	case "/account", "/account/limits", "/account/password", "/organizations",
		"/apps/install/preview", "/apps/install":
		return true
	default:
		return strings.HasPrefix(path, "/api/v1/invite-links/")
	}
}

func getAccountFromCookie(r *http.Request, jwtSigner *jwt.Signer) (string, int64, error) {
	cookie, err := r.Cookie("account_token")
	if err != nil {
		return "", 0, fmt.Errorf("account token cookie not found")
	}

	claims, err := jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return "", 0, fmt.Errorf("invalid account token: %v", err)
	}

	if !authentication.IsAccountSessionWithinMaxAge(claims) {
		return "", 0, fmt.Errorf("session exceeded maximum age")
	}

	accountClaim, exists := claims["sub"]
	if !exists {
		return "", 0, fmt.Errorf("account ID missing from token")
	}

	accountID, ok := accountClaim.(string)
	if !ok {
		return "", 0, fmt.Errorf("invalid account ID in token")
	}

	iat, _ := claims["iat"].(float64)

	return accountID, int64(iat), nil
}

// getValidatedAccountFromCookie reads the account_token cookie, loads the
// matching account, and rejects the request if the token's iat is older
// than the account's PasswordChangedAt. Use this anywhere a cookie-based
// account session needs to be validated end-to-end.
func getValidatedAccountFromCookie(r *http.Request, jwtSigner *jwt.Signer) (*models.Account, error) {
	accountID, iat, err := getAccountFromCookie(r, jwtSigner)
	if err != nil {
		return nil, err
	}

	account, err := models.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	if !account.IsSessionFresh(iat) {
		return nil, fmt.Errorf("session invalidated by password change")
	}

	return account, nil
}

func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}

func GetImpersonationFromContext(ctx context.Context) (*ImpersonationInfo, bool) {
	info, ok := ctx.Value(ImpersonationContextKey).(*ImpersonationInfo)
	return info, ok
}

func GetScopedTokenClaimsFromContext(ctx context.Context) (*jwt.ScopedTokenClaims, bool) {
	claims, ok := ctx.Value(ScopedTokenClaimsContextKey).(*jwt.ScopedTokenClaims)
	return claims, ok
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
