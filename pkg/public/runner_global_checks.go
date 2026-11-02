package public

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
)

const (
	runnerChecksMax             = 20
	runnerCheckReasonMaxRunes   = 1000
	runnerChecksBodyMaxBytes    = 64 * 1024
	runnerRemoveChecksBodyBytes = 16 * 1024
)

var runnerCheckLevels = []string{
	models.FactoryWorkOrderCheckLevelPositive,
	models.FactoryWorkOrderCheckLevelCaution,
	models.FactoryWorkOrderCheckLevelCritical,
}

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
	params, err := runnerCheckParams(request.Checks, target.canvasRunID)
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
	keys := []string{}
	for _, raw := range request.Keys {
		key := strings.TrimSpace(raw)
		if key != "" && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		http.Error(w, "keys is required", http.StatusBadRequest)
		return
	}
	if len(keys) > runnerChecksMax {
		http.Error(w, fmt.Sprintf("keys must have at most %d items", runnerChecksMax), http.StatusBadRequest)
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

func runnerCheckParams(inputs []runnerCheckInput, runID uuid.UUID) ([]models.FactoryWorkOrderCheckParams, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("checks is required")
	}
	if len(inputs) > runnerChecksMax {
		return nil, fmt.Errorf("checks must have at most %d items", runnerChecksMax)
	}
	params := make([]models.FactoryWorkOrderCheckParams, 0, len(inputs))
	seen := make([]string, 0, len(inputs))
	for index, input := range inputs {
		label := fmt.Sprintf("checks[%d]", index)
		key := strings.TrimSpace(input.Key)
		if slices.Contains(seen, key) {
			return nil, fmt.Errorf("%s.key repeats an earlier key", label)
		}
		seen = append(seen, key)
		if input.Score == nil || input.MaxScore == nil {
			return nil, fmt.Errorf("%s.score and %s.max_score are required", label, label)
		}
		if *input.Score > *input.MaxScore {
			return nil, fmt.Errorf("%s.score must be from 0 to max_score", label)
		}
		level := strings.ToLower(strings.TrimSpace(input.Level))
		if !slices.Contains(runnerCheckLevels, level) {
			return nil, fmt.Errorf("%s.level must be positive, caution, or critical", label)
		}
		reason := strings.TrimSpace(input.Reason)
		if reason == "" {
			return nil, fmt.Errorf("%s.reason is required", label)
		}
		if utf8.RuneCountInString(reason) > runnerCheckReasonMaxRunes {
			return nil, fmt.Errorf("%s.reason must be at most %d characters", label, runnerCheckReasonMaxRunes)
		}
		params = append(params, models.FactoryWorkOrderCheckParams{
			Key:      key,
			Name:     input.Name,
			Score:    *input.Score,
			MaxScore: *input.MaxScore,
			Level:    level,
			Summary:  reason,
			Run:      &factoryevents.RunRef{ID: runID},
		})
	}
	return params, nil
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
