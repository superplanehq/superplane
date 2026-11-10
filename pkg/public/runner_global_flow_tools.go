package public

import (
	"fmt"
	"net/http"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// The flow tools of the global runner MCP server reuse the legacy flow
// handlers. The runner token replaces the flow token, and its scope must
// allow the tool.

func (s *Server) handleRunnerGlobalPlanUpdate(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopePlanWrite)
	if !ok {
		return
	}
	planning, ok := scope.PlanningSessionScope()
	if !ok {
		writeRunnerFlowMismatch(w)
		return
	}
	s.runnerPlanningUpdate(w, r, planning)
}

func (s *Server) handleRunnerGlobalMergeCheck(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeMergeChecksWrite)
	if !ok {
		return
	}
	merge, ok := scope.MergeConfidenceScope()
	if !ok {
		writeRunnerFlowMismatch(w)
		return
	}
	s.reportRunnerMergeConfidenceCheck(w, r, merge)
}

func (s *Server) handleRunnerGlobalArtifactUpload(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerGlobalArtifactTarget(w, r)
	if !ok {
		return
	}
	s.uploadRunnerArtifact(w, r, target)
}

func (s *Server) handleRunnerGlobalArtifactLink(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerGlobalArtifactTarget(w, r)
	if !ok {
		return
	}
	s.attachRunnerArtifactLink(w, r, target)
}

func (s *Server) handleRunnerGlobalArtifactMarkdown(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerGlobalArtifactTarget(w, r)
	if !ok {
		return
	}
	s.attachRunnerArtifactMarkdown(w, r, target)
}

// runnerGlobalArtifactTarget resolves the task of a runner token for an
// artifact. A task run keeps the node execution checks of the artifact
// token. A planning run attaches to the session draft.
func (s *Server) runnerGlobalArtifactTarget(w http.ResponseWriter, r *http.Request) (*runnerArtifactTarget, bool) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeArtifactsWrite)
	if !ok {
		return nil, false
	}
	if artifact, ok := scope.ArtifactUploadScope(); ok {
		return artifactUploadTarget(artifact), true
	}
	task, ok := s.runnerTaskTarget(w, r, scope)
	if !ok {
		return nil, false
	}
	return &runnerArtifactTarget{
		OrganizationID: scope.OrganizationID,
		FactoryID:      scope.FactoryID,
		WorkOrderID:    task.order.ID,
		CanvasRunID:    scope.CanvasRunID,
		load: func(db *gorm.DB) (*runnerArtifactContext, error) {
			return loadRunnerFlowArtifactContext(db, scope, task)
		},
	}, true
}

// loadRunnerFlowArtifactContext attributes an artifact of a planning or a
// merge confidence run to the node that started the run.
func loadRunnerFlowArtifactContext(db *gorm.DB, scope *runneraction.RunnerScope, task *taskOwnerTarget) (*runnerArtifactContext, error) {
	nodeExecution, err := models.FindNodeExecutionInTransaction(db, scope.CanvasID, scope.NodeExecutionID)
	if err != nil || nodeExecution.RunID != scope.CanvasRunID || nodeExecution.NodeID != scope.NodeID {
		return nil, fmt.Errorf("node execution does not match token")
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, scope.CanvasID)
	if err != nil || canvas.OrganizationID != scope.OrganizationID || canvas.FactoryID == nil || *canvas.FactoryID != scope.FactoryID {
		return nil, fmt.Errorf("canvas does not match token")
	}
	node, err := models.FindCanvasNode(db, scope.CanvasID, scope.NodeID)
	if err != nil {
		return nil, err
	}
	return &runnerArtifactContext{
		Factory:       task.factory,
		Order:         task.order,
		Canvas:        canvas,
		Node:          node,
		NodeExecution: nodeExecution,
	}, nil
}

func writeRunnerFlowMismatch(w http.ResponseWriter) {
	http.Error(w, "This run cannot use this tool", http.StatusForbidden)
}
