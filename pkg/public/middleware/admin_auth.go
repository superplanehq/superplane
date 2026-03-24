package middleware

import (
	"net/http"

	"github.com/gorilla/mux"
)

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
