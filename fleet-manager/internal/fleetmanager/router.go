package fleetmanager

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterOptions configures HTTP middleware.
type RouterOptions struct {
	// AuthToken when non-empty requires Authorization: Bearer <token> for /v1.
	AuthToken string
}

// NewRouter builds chi routes for fleet-manager.
func NewRouter(s *Server, opt RouterOptions) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.health)

	r.Route("/v1", func(r chi.Router) {
		if strings.TrimSpace(opt.AuthToken) != "" {
			r.Use(bearerAuth(opt.AuthToken))
		}
		r.Post("/tasks", s.createTask)
		r.Post("/tasks/claim", s.claimTask)
		r.Get("/tasks/{id}", s.getTask)
		r.Post("/tasks/{id}/complete", s.completeTask)
	})

	return r
}

func bearerAuth(want string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			raw = strings.TrimSpace(raw)
			if raw != want {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
