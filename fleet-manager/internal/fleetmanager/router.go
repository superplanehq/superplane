package fleetmanager

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/superplane/runner/shared/httpaccess"
)

// RouterOptions configures HTTP middleware.
type RouterOptions struct {
	// AuthToken when non-empty requires Authorization: Bearer <token> for /v1.
	AuthToken string
	// DiagnosticsToken when non-empty enables /v1/admin/* (Bearer auth). Requires EC2Launcher on Server.
	DiagnosticsToken string
}

// NewRouter builds chi routes for fleet-manager.
func NewRouter(s *Server, opt RouterOptions) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	if s.Log != nil {
		r.Use(httpaccess.ChiAccessLog(s.Log))
	}

	r.Get("/healthz", s.health)

	if strings.TrimSpace(opt.DiagnosticsToken) != "" && s.EC2Launcher != nil {
		r.Route("/v1/admin", func(r chi.Router) {
			r.Use(bearerAuth(opt.DiagnosticsToken))
			r.Get("/managed-runners", s.adminManagedRunners)
			r.Get("/ec2-console-output", s.adminEc2Console)
		})
	}

	r.Route("/v1", func(r chi.Router) {
		if strings.TrimSpace(opt.AuthToken) != "" {
			r.Use(bearerAuth(opt.AuthToken))
		}
		r.Post("/tasks", s.createTask)
		r.Get("/runners/stream", s.runnerStream)
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
