package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

const fleetManagerContextKey contextKey = "fleet_manager"

// AdminAuthMiddleware authenticates admin UI sessions, named personal API
// tokens, and the single-host Fleet Manager token on fleet routes.
// Organization API keys and other scoped tokens are not valid admin credentials.
func AdminAuthMiddleware(jwtSigner *jwt.Signer, fleetManagerToken string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		cookieHandler := AccountAuthMiddleware(jwtSigner)(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				cookieHandler.ServeHTTP(w, r)
				return
			}

			rawToken, err := getBearerToken(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if fleetManagerTokenMatches(r, rawToken, fleetManagerToken) {
				ctx := context.WithValue(r.Context(), fleetManagerContextKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			user, err := authenticateUserByPersonalToken(r.Context(), crypto.HashToken(rawToken))
			if err != nil {
				if errors.Is(err, models.ErrAccountBlocked) {
					http.Error(w, models.AccountBlockedMessage, http.StatusForbidden)
					return
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if user.AccountID == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			account, err := models.FindAccountByID(user.AccountID.String())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), AccountContextKey, account)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireInstallationAdmin is a middleware that ensures the request
// is from an authenticated installation admin. Non-admin requests
// receive a 404 to avoid leaking the existence of admin endpoints.
func RequireInstallationAdmin() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsFleetManagerRequest(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}
			account, ok := GetAccountFromContext(r.Context())
			if !ok {
				http.NotFound(w, r)
				return
			}

			if !account.IsInstallationAdmin() {
				http.NotFound(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func fleetManagerTokenMatches(r *http.Request, provided, expected string) bool {
	if expected == "" || !isFleetRoute(r.URL.Path) {
		return false
	}
	providedHash := sha256.Sum256([]byte(provided))
	expectedHash := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) == 1
}

func isFleetRoute(path string) bool {
	const fleetPath = "/admin/api/installation/fleets"
	return path == fleetPath || strings.HasPrefix(path, fleetPath+"/")
}

func IsFleetManagerRequest(ctx context.Context) bool {
	authenticated, _ := ctx.Value(fleetManagerContextKey).(bool)
	return authenticated
}
