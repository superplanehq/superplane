package broker

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/httpaccess"
)

// RouterOptions configures HTTP middleware.
type RouterOptions struct {
	AuthToken           string
	LiveLogsCORSOrigins []string
}

// NewRouter builds chi routes for task-broker.
func NewRouter(s *Server, opt RouterOptions) http.Handler {
	if s.RunnerDrain == nil {
		s.RunnerDrain = NewRunnerDrainHub()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	if s.Log != nil {
		r.Use(httpaccess.ChiAccessLog(s.Log))
	}

	r.Get("/healthz", s.health)

	r.Route("/v1", func(r chi.Router) {
		auth := strings.TrimSpace(opt.AuthToken)
		if auth == "" {
			panic("broker: AuthToken is required — use mandatory AUTH_TOKEN from main")
		}

		r.Route("/tasks/{id}/live-logs", func(r chi.Router) {
			r.Use(liveLogsCORS(opt.LiveLogsCORSOrigins))
			r.Use(liveLogsAuth(auth))
			r.Get("/", s.getTaskLiveLogs)
			r.Options("/", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(bearerAuth(auth))
			r.Get("/fleets", s.listFleets)
			r.Post("/fleets", s.registerFleet)
			r.Delete("/fleets/{id}", s.deleteFleet)
			r.Get("/fleets/{id}/task-counts", s.getFleetTaskCounts)
			r.Get("/tasks", s.listTasks)
			r.Post("/tasks", s.createTask)
			r.Post("/tasks/claim", s.claimTask)
			r.Get("/tasks/{id}", s.getTask)
			r.Post("/tasks/{id}/cancel", s.cancelTask)
			r.Post("/tasks/{id}/complete", s.completeTask)
			r.Post("/runners/drain", s.drainRunners)
			r.Get("/runners/stream", s.runnerStream)
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
