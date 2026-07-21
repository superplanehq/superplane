package broker

import (
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/livelogstoken"
)

func liveLogsCORS(origins []string) func(http.Handler) http.Handler {
	allowed := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" && !slices.Contains(allowed, origin) {
			allowed = append(allowed, origin)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" && slices.Contains(allowed, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Accept")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func liveLogsAuth(authToken string) func(http.Handler) http.Handler {
	authToken = strings.TrimSpace(authToken)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			if raw == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			taskID := strings.TrimSpace(chi.URLParam(r, "id"))
			if taskID == "" {
				writeError(w, http.StatusBadRequest, "id required")
				return
			}

			if authToken != "" && raw == authToken {
				next.ServeHTTP(w, r)
				return
			}

			if authToken == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if err := livelogstoken.Validate(raw, taskID, authToken); err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ParseLiveLogsCORSOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
