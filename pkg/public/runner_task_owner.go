package public

import (
	"net/http"

	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func (s *Server) handleRunnerTaskSuggestOwners(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerSuggestOwners(w, r, s.taskOwnerTarget)
}

func (s *Server) handleRunnerTaskAssignOwner(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerAssignOwner(w, r, s.taskOwnerTarget)
}

func (s *Server) taskOwnerTarget(w http.ResponseWriter, r *http.Request) (*taskOwnerTarget, bool) {
	scope, err := runner.ParseTaskOwnerToken(s.jwt, bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	if err := runner.ValidateTaskOwnerScope(database.DB(r.Context()), scope); err != nil {
		http.Error(w, "Task owner scope is invalid", http.StatusForbidden)
		return nil, false
	}
	target, err := s.loadTaskOwnerTarget(r.Context(), scope.OrganizationID, scope.FactoryID, scope.WorkOrderID, "")
	if err != nil {
		writeMergeConfidenceLookupError(w, err)
		return nil, false
	}
	if target.order.State == models.FactoryWorkOrderStateClosed || target.order.IsArchived() {
		http.Error(w, "Task is closed or archived", http.StatusConflict)
		return nil, false
	}
	return target, true
}
