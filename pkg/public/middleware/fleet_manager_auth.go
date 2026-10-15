package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const FleetManagerTokenPathEnvironment = "FLEET_MANAGER_TOKEN_PATH"

// FleetManagerAuthMiddleware accepts the Fleet Manager service token or an
// installation administrator. Apply it only to installation fleet routes.
func FleetManagerAuthMiddleware(jwtSigner *jwt.Signer, tokenPath string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		adminHandler := AdminAuthMiddleware(jwtSigner)(RequireInstallationAdmin()(next))

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if serviceTokenMatches(r, tokenPath) {
				next.ServeHTTP(w, r)
				return
			}

			adminHandler.ServeHTTP(w, r)
		})
	}
}

func serviceTokenMatches(r *http.Request, tokenPath string) bool {
	if tokenPath == "" || r.Header.Get("Authorization") == "" {
		return false
	}

	provided, err := getBearerToken(r)
	if err != nil || provided == "" {
		return false
	}

	expected, err := os.ReadFile(tokenPath)
	if err != nil {
		return false
	}

	expectedToken := strings.TrimSpace(string(expected))
	if expectedToken == "" {
		return false
	}

	providedDigest := sha256.Sum256([]byte(provided))
	expectedDigest := sha256.Sum256([]byte(expectedToken))
	return subtle.ConstantTimeCompare(providedDigest[:], expectedDigest[:]) == 1
}
