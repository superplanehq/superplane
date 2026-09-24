package broker

import (
	"context"
	"net/http"
	"strings"

	"github.com/superplane/runner/shared/opaquetoken"
)

type runnerIdentity struct {
	RunnerID string
	FleetID  string
}

type runnerIdentityKey struct{}

func bearerToken(r *http.Request) string {
	return strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
}

func runnerIdentityFromContext(ctx context.Context) (runnerIdentity, bool) {
	identity, ok := ctx.Value(runnerIdentityKey{}).(runnerIdentity)
	return identity, ok
}

func (s *Server) runnerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		credential, err := s.Store.GetRunnerByAccessTokenHash(r.Context(), opaquetoken.Hash(token))
		if err != nil {
			s.logErr("authenticate runner", err)
			writeError(w, http.StatusInternalServerError, "could not authenticate runner")
			return
		}
		if credential == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx := context.WithValue(r.Context(), runnerIdentityKey{}, runnerIdentity{
			RunnerID: credential.RunnerID,
			FleetID:  credential.FleetID,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) controlOrRunnerAuth(controlToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if bearerToken(r) == controlToken {
				next.ServeHTTP(w, r)
				return
			}
			s.runnerAuth(next).ServeHTTP(w, r)
		})
	}
}
