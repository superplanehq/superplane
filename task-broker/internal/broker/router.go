package broker

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterOptions configures HTTP middleware.
type RouterOptions struct {
	AuthToken string
}

// NewRouter builds chi routes for task-broker.
func NewRouter(s *Server, opt RouterOptions) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.health)

	r.Route("/v1", func(r chi.Router) {
		// Completion callbacks come from downstream fleet-manager; must not require caller bearer auth.
		r.Post("/webhooks/complete/{brokerTaskID}", s.webhookComplete)
		auth := strings.TrimSpace(opt.AuthToken)
		if auth == "" {
			panic("broker: AuthToken is required — use mandatory AUTH_TOKEN from main")
		}
		r.Group(func(r chi.Router) {
			r.Use(bearerAuth(auth))
			r.Get("/fleets", s.listFleets)
			r.Post("/fleets", s.registerFleet)
			r.Delete("/fleets/{id}", s.deleteFleet)
			r.Post("/tasks", s.createBrokerTask)
			r.Get("/tasks/{id}", s.getBrokerTask)
		})
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
