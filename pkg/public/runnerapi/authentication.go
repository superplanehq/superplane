package runnerapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type runnerContextKey struct{}

func runnerFromContext(ctx context.Context) (*models.Runner, bool) {
	runner, ok := ctx.Value(runnerContextKey{}).(*models.Runner)
	return runner, ok
}

func (s *Server) authenticateRunner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "runner credential is required")
			return
		}

		runner, err := models.FindRunnerByAccessTokenHash(database.DB(r.Context()), crypto.HashToken(token))
		if errors.Is(err, models.ErrRunnerCredentialNotFound) {
			writeError(w, http.StatusUnauthorized, "runner credential is invalid")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "runner authentication failed")
			return
		}

		ctx := context.WithValue(r.Context(), runnerContextKey{}, runner)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, value, ok := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}
