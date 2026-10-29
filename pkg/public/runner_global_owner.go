package public

import (
	"net/http"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
)

func (s *Server) handleRunnerGlobalSuggestOwners(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerSuggestOwners(w, r, s.runnerOwnerTarget(runneraction.ScopeOwnerRead))
}

func (s *Server) handleRunnerGlobalAssignOwner(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerAssignOwner(w, r, s.runnerOwnerTarget(runneraction.ScopeOwnerWrite))
}

// runnerOwnerTarget resolves the task of a runner token. Planning runs read
// the draft task from the session, because the draft can change during the
// session.
func (s *Server) runnerOwnerTarget(required runneraction.RunnerToolScope) ownerTargetLoader {
	return func(w http.ResponseWriter, r *http.Request) (*taskOwnerTarget, bool) {
		scope, ok := s.authenticateRunner(w, r, required)
		if !ok {
			return nil, false
		}
		if scope.Flow != runneraction.RunnerFlowPlanning {
			return s.openTaskOwnerTarget(w, r, scope.OrganizationID, scope.FactoryID, scope.WorkOrderID)
		}
		target, err := s.loadPlanningTaskOwnerTarget(r, &runneraction.PlanningSessionScope{
			OrganizationID: scope.OrganizationID,
			FactoryID:      scope.FactoryID,
			SessionID:      scope.PlanningSessionID,
			CanvasRunID:    scope.CanvasRunID,
		})
		if err != nil {
			writeRunnerPlanningError(w, r, sessionOf(target), err)
			return nil, false
		}
		return target, true
	}
}
