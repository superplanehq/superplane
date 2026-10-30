package runner

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ValidateRunnerScope rechecks the signed run associations on every request.
// A token outlives its run, so the backend must not trust the claims alone.
func ValidateRunnerScope(db *gorm.DB, scope *RunnerScope) error {
	execution, err := validateRunnerExecution(db, scope)
	if err != nil {
		return err
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, scope.CanvasID)
	if err != nil || canvas.OrganizationID != scope.OrganizationID {
		return fmt.Errorf("runner canvas does not match")
	}

	switch scope.Flow {
	case RunnerFlowPlanning:
		return validatePlanningRunnerScope(db, scope)
	case RunnerFlowMergeConfidence:
		if canvas.FactoryID == nil || *canvas.FactoryID != scope.FactoryID {
			return fmt.Errorf("runner canvas does not match")
		}
		if execution.State != models.CanvasNodeExecutionStateStarted {
			return fmt.Errorf("merge confidence execution is inactive")
		}
		return nil
	case RunnerFlowTask:
		if canvas.FactoryID == nil || *canvas.FactoryID != scope.FactoryID {
			return fmt.Errorf("runner canvas does not match")
		}
		linked, err := ResolveArtifactRunContext(db, scope.CanvasRunID)
		if err != nil || linked.OrganizationID != scope.OrganizationID || linked.FactoryID != scope.FactoryID ||
			linked.WorkOrderID != scope.WorkOrderID || linked.CanvasID != scope.CanvasID {
			return fmt.Errorf("runner task association does not match")
		}
		return nil
	default:
		return fmt.Errorf("invalid runner flow %q", scope.Flow)
	}
}

func validateRunnerExecution(db *gorm.DB, scope *RunnerScope) (*models.CanvasNodeExecution, error) {
	execution, err := models.FindNodeExecutionInTransaction(db, scope.CanvasID, scope.NodeExecutionID)
	if err != nil || execution.RunID != scope.CanvasRunID || execution.NodeID != scope.NodeID {
		return nil, fmt.Errorf("runner execution does not match")
	}
	run, err := models.FindUnscopedCanvasRun(db, scope.CanvasRunID)
	if err != nil || run.WorkflowID != scope.CanvasID {
		return nil, fmt.Errorf("runner run does not match")
	}
	return execution, nil
}

func validatePlanningRunnerScope(db *gorm.DB, scope *RunnerScope) error {
	session, err := models.FindPlanningSession(db, scope.OrganizationID, scope.FactoryID, scope.PlanningSessionID)
	if err != nil || session.CanvasRunID == nil || *session.CanvasRunID != scope.CanvasRunID || !session.IsAnalysisSession() {
		return fmt.Errorf("runner planning session does not match")
	}
	return nil
}
