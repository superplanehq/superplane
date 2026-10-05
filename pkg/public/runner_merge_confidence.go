package public

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	factorycomp "github.com/superplanehq/superplane/pkg/components/factory"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

var errMergeConfidenceScope = errors.New("merge confidence run does not match")

type mergeConfidenceCheckRequest struct {
	Check   string  `json:"check"`
	Score   float64 `json:"score"`
	Summary string  `json:"summary"`
}

func (s *Server) handleRunnerMergeConfidenceCheck(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateMergeConfidenceRunner(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var req mergeConfidenceCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	factoryModel, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		writeMergeConfidenceLookupError(w, err)
		return
	}
	order, err := factoryModel.FindWorkOrder(db, scope.WorkOrderID)
	if err != nil {
		writeMergeConfidenceLookupError(w, err)
		return
	}
	automation, err := mergeConfidenceAutomation(db, scope)
	if err != nil {
		writeMergeConfidenceLookupError(w, err)
		return
	}
	if _, err := factorycomp.ReportMergeConfidenceCheck(db, order, scope.CanvasRunID, automation, req.Check, req.Score, req.Summary, scope.EnabledChecks); err != nil {
		writeMergeConfidenceReportError(w, err)
		return
	}
	if err := messages.PublishFactoryWorkOrderUpdated(order.FactoryID.String(), order.ID.String(), factoryevents.EventTypeOrderCheckReported); err != nil {
		log.WithError(err).Warn("failed to publish merge confidence check")
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "reported"})
}

func (s *Server) authenticateMergeConfidenceRunner(w http.ResponseWriter, r *http.Request) (*runneraction.MergeConfidenceScope, bool) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	scope, err := runneraction.ParseMergeConfidenceToken(s.jwt, token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return scope, true
}

func writeMergeConfidenceLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, models.ErrFactoryNotFound) || errors.Is(err, models.ErrFactoryWorkOrderNotFound) || errors.Is(err, errMergeConfidenceScope) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

// mergeConfidenceAutomation attributes the check to the canvas that owns the
// signed run. The task list shows a Verify row only when the check carries
// that canvas.
func mergeConfidenceAutomation(db *gorm.DB, scope *runneraction.MergeConfidenceScope) (*factoryevents.AutomationRef, error) {
	run, err := models.FindUnscopedCanvasRun(db, scope.CanvasRunID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errMergeConfidenceScope
		}
		return nil, err
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, run.WorkflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errMergeConfidenceScope
		}
		return nil, err
	}
	if canvas.OrganizationID != scope.OrganizationID || canvas.FactoryID == nil || *canvas.FactoryID != scope.FactoryID {
		return nil, errMergeConfidenceScope
	}
	execution, err := models.FindNodeExecutionInTransaction(db, canvas.ID, scope.NodeExecutionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errMergeConfidenceScope
		}
		return nil, err
	}
	if execution.RunID != scope.CanvasRunID {
		return nil, errMergeConfidenceScope
	}
	ref := &factoryevents.AutomationRef{
		NodeID:  execution.NodeID,
		AppID:   canvas.ID,
		AppName: canvas.Name,
	}
	node, err := models.FindCanvasNode(db, canvas.ID, execution.NodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if node != nil {
		ref.NodeName = node.Name
	}
	return ref, nil
}

func writeMergeConfidenceReportError(w http.ResponseWriter, err error) {
	if errors.Is(err, factorycomp.ErrMergeConfidenceInvalid) || errors.Is(err, factorycomp.ErrMergeConfidenceDisabled) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}
