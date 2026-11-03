package public

import (
	"errors"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
)

const (
	runnerChecksBodyMaxBytes    = 64 * 1024
	runnerRemoveChecksBodyBytes = 16 * 1024
)

type runnerCheckInput struct {
	Key      string   `json:"key"`
	Name     string   `json:"name"`
	Score    *float64 `json:"score"`
	MaxScore *float64 `json:"max_score"`
	Level    string   `json:"level"`
	Reason   string   `json:"reason"`
}

type runnerAddChecksRequest struct {
	Checks []runnerCheckInput `json:"checks"`
}

type runnerRemoveChecksRequest struct {
	Keys []string `json:"keys"`
}

type runnerCheckView struct {
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	Score    float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
	Level    string  `json:"level"`
	Reason   string  `json:"reason"`
}

func (s *Server) handleRunnerListChecks(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeChecksRead)
	if !ok {
		return
	}
	ref := strings.TrimSpace(r.URL.Query().Get("task"))
	if ref == "" {
		http.Error(w, "task is required", http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	factoryModel, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		writeRunnerChecksError(w, err)
		return
	}
	order, err := factoryModel.FindWorkOrderByRef(db, ref)
	if err == nil && order.IsArchived() {
		err = models.ErrFactoryWorkOrderNotFound
	}
	if err != nil {
		writeRunnerChecksError(w, err)
		return
	}
	checks, err := order.ListChecks(db)
	if err != nil {
		writeRunnerChecksError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"checks": runnerCheckViews(checks)})
}

func (s *Server) handleRunnerAddChecks(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopeChecksWrite)(w, r)
	if !ok {
		return
	}
	var request runnerAddChecksRequest
	if err := decodeRunnerJSON(w, r, &request, runnerChecksBodyMaxBytes); err != nil {
		return
	}
	params, err := models.AgentCheckParams(agentCheckInputs(request.Checks), &factoryevents.RunRef{ID: target.canvasRunID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	checks, err := target.order.ReportAgentChecks(database.DB(r.Context()), params)
	if err != nil {
		writeRunnerChecksError(w, err)
		return
	}
	publishRunnerTaskUpdate(target.factory, target.order, factoryevents.EventTypeOrderCheckReported)
	_ = writeJSON(w, http.StatusOK, map[string]any{"checks": runnerCheckViews(checks)})
}

func (s *Server) handleRunnerRemoveChecks(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopeChecksWrite)(w, r)
	if !ok {
		return
	}
	var request runnerRemoveChecksRequest
	if err := decodeRunnerJSON(w, r, &request, runnerRemoveChecksBodyBytes); err != nil {
		return
	}
	keys, err := models.AgentCheckKeys(request.Keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	removed, err := target.order.RemoveAgentChecks(database.DB(r.Context()), keys)
	if err != nil {
		writeRunnerChecksError(w, err)
		return
	}
	if len(removed) > 0 {
		publishRunnerTaskUpdate(target.factory, target.order, factoryevents.EventTypeOrderCheckReported)
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"removed": removed})
}

func agentCheckInputs(inputs []runnerCheckInput) []models.AgentCheckInput {
	result := make([]models.AgentCheckInput, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, models.AgentCheckInput{
			Key:      input.Key,
			Name:     input.Name,
			Score:    input.Score,
			MaxScore: input.MaxScore,
			Level:    input.Level,
			Reason:   input.Reason,
		})
	}
	return result
}

func runnerCheckViews(checks []models.FactoryWorkOrderCheck) []runnerCheckView {
	views := make([]runnerCheckView, 0, len(checks))
	for _, check := range checks {
		views = append(views, runnerCheckView{
			Key:      check.Key,
			Name:     check.Name,
			Score:    check.Score,
			MaxScore: check.MaxScore,
			Level:    check.Level,
			Reason:   check.Summary,
		})
	}
	return views
}

func writeRunnerChecksError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrFactoryWorkOrderNotFound), errors.Is(err, models.ErrFactoryNotFound):
		http.Error(w, "Task not found", http.StatusNotFound)
	case errors.Is(err, models.ErrFactoryWorkOrderCheckOwned):
		http.Error(w, err.Error()+"; use another key", http.StatusConflict)
	case errors.Is(err, models.ErrFactoryWorkOrderCheckInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		log.WithError(err).Error("runner checks request failed")
		http.Error(w, "Checks request failed", http.StatusInternalServerError)
	}
}
