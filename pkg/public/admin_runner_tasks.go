package public

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
)

type adminRunnerTasksResponse struct {
	Configured bool                `json:"configured"`
	Tasks      []runner.ActiveTask `json:"tasks"`
	Error      string              `json:"error,omitempty"`
}

func (s *Server) adminListRunnerTasks(w http.ResponseWriter, r *http.Request) {
	broker, err := runner.NewBrokerClient(s.registry.HTTPContext())
	if err != nil {
		respondJSON(w, adminRunnerTasksResponse{
			Configured: false,
			Tasks:      []runner.ActiveTask{},
		})
		return
	}

	tasks, err := broker.ListActiveTasks()
	if err != nil {
		// The admin UI polls this endpoint every few seconds. Returning a 5xx
		// for transient broker failures floods Sentry with noise (the broker
		// is an external dependency, not a SuperPlane bug). Surface the error
		// to the UI inline with a 200 response so it can be displayed without
		// triggering server-error alerting.
		log.Warnf("admin: failed to list runner tasks: %v", err)
		respondJSON(w, adminRunnerTasksResponse{
			Configured: true,
			Tasks:      []runner.ActiveTask{},
			Error:      "Failed to list runner tasks from the task broker.",
		})
		return
	}

	if tasks == nil {
		tasks = []runner.ActiveTask{}
	}

	respondJSON(w, adminRunnerTasksResponse{
		Configured: true,
		Tasks:      tasks,
	})
}
