package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

// AdminAuthMiddleware authenticates admin UI sessions and named personal API
// tokens. Organization API keys and scoped tokens are not valid admin
// credentials.
func AdminAuthMiddleware(jwtSigner *jwt.Signer) mux.MiddlewareFunc {
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
