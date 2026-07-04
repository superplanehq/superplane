package public

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
)

type adminRunnerTasksResponse struct {
	Configured bool                `json:"configured"`
	Tasks      []runner.ActiveTask `json:"tasks"`
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
		log.Errorf("admin: failed to list runner tasks: %v", err)
		http.Error(w, "Failed to list runner tasks", http.StatusBadGateway)
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
