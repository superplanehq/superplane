package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	minPlanningHoldSeconds = 1
	maxPlanningHoldSeconds = 60
)

type planningDraftRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type planningSurveyRequest struct {
	Questions []models.PlanningSessionSurveyQuestion `json:"questions"`
}

func (s *Server) authenticatePlanningSessionRunner(w http.ResponseWriter, r *http.Request) (*runneraction.PlanningSessionScope, bool) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	scope, err := runneraction.ParsePlanningSessionToken(s.jwt, token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return scope, true
}

func (s *Server) loadPlanningSessionForRunner(r *http.Request, scope *runneraction.PlanningSessionScope) (*models.FactoryPlanningSession, error) {
	db := database.DB(r.Context())
	session, err := models.FindPlanningSession(db, scope.OrganizationID, scope.FactoryID, scope.SessionID)
	if err != nil {
		return nil, err
	}
	if session.CanvasRunID == nil || *session.CanvasRunID != scope.CanvasRunID {
		return nil, models.ErrFactoryPlanningSessionNotFound
	}
	return session, nil
}

func (s *Server) handleRunnerPlanningWait(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	hold := clampPlanningHoldSeconds(r.URL.Query().Get("hold_seconds"))
	deadline := time.Now().Add(time.Duration(hold) * time.Second)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		session, err := s.loadPlanningSessionForRunner(r, scope)
		if err != nil {
			writeRunnerPlanningError(w, err)
			return
		}
		if session.State == models.PlanningSessionStateEnded {
			writeJSON(w, http.StatusOK, map[string]any{"status": "ended"})
			return
		}
		if session.WaitState == models.PlanningWaitResolved {
			result, err := session.ConsumeWait(database.DB(r.Context()))
			if err != nil {
				writeRunnerPlanningError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"status":         result.Kind,
				"text":           result.Text,
				"work_order_id":  result.WorkOrderID,
				"work_order_key": result.WorkOrderKey,
			})
			return
		}
		if err := session.BeginWait(database.DB(r.Context())); err != nil {
			writeRunnerPlanningError(w, err)
			return
		}
		if !time.Now().Before(deadline) {
			writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) handleRunnerPlanningDraft(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	session, err := s.loadPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, err)
		return
	}
	if err := session.ProposeDraft(database.DB(r.Context()), models.PlanningSessionDraft{
		Title:       req.Title,
		Description: req.Description,
	}); err != nil {
		writeRunnerPlanningError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "shown"})
}

func (s *Server) handleRunnerPlanningSurvey(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningSurveyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	session, err := s.loadPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, err)
		return
	}
	if err := session.ProposeSurvey(database.DB(r.Context()), models.PlanningSessionSurvey{
		Questions: req.Questions,
	}); err != nil {
		writeRunnerPlanningError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "shown"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.WithError(err).Error("failed to encode planning session runner response")
	}
}

func writeRunnerPlanningError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrFactoryPlanningSessionInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, models.ErrFactoryPlanningSessionNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		http.Error(w, "planning session not found", http.StatusNotFound)
	case errors.Is(err, models.ErrFactoryPlanningSessionEnded):
		http.Error(w, "planning session has ended", http.StatusConflict)
	default:
		log.WithError(err).Error("runner planning session failed")
		http.Error(w, "Lookup failed", http.StatusInternalServerError)
	}
}

func clampPlanningHoldSeconds(raw string) int {
	hold := 45
	if raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil {
			hold = parsed
		}
	}
	if hold < minPlanningHoldSeconds {
		return minPlanningHoldSeconds
	}
	if hold > maxPlanningHoldSeconds {
		return maxPlanningHoldSeconds
	}
	return hold
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
