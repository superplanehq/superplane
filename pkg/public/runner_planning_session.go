package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type planningAskRequest struct {
	TimeoutSeconds int                              `json:"timeout_seconds"`
	Questions      []models.WorkOrderSurveyQuestion `json:"questions"`
}

type planningDraftRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type planningSayRequest struct {
	Text string `json:"text"`
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
	hold := clampWorkOrderSurveyHoldSeconds(r.URL.Query().Get("hold_seconds"))
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

func (s *Server) handleRunnerPlanningAsk(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningAskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	session, err := s.loadPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, err)
		return
	}
	survey, err := session.CreateSurvey(database.DB(r.Context()), req.Questions)
	if err != nil {
		writeRunnerPlanningError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      survey.ID.String(),
		"status":  survey.Status,
		"answers": survey.Answers,
	})
}

func (s *Server) handleRunnerPlanningSurveyWait(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	surveyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid survey id", http.StatusBadRequest)
		return
	}
	hold := clampWorkOrderSurveyHoldSeconds(r.URL.Query().Get("hold_seconds"))
	deadline := time.Now().Add(time.Duration(hold) * time.Second)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		db := database.DB(r.Context())
		survey, err := models.FindPlanningSessionSurvey(db, scope.OrganizationID, surveyID)
		if err != nil {
			writeRunnerPlanningError(w, err)
			return
		}
		if survey.SessionID != scope.SessionID {
			writeRunnerPlanningError(w, models.ErrFactoryWorkOrderSurveyNotFound)
			return
		}
		if survey.Status != models.FactoryWorkOrderSurveyPending {
			writeJSON(w, http.StatusOK, map[string]any{
				"id":      survey.ID.String(),
				"status":  survey.Status,
				"answers": survey.Answers,
			})
			return
		}
		if !time.Now().Before(deadline) {
			writeJSON(w, http.StatusOK, map[string]any{
				"id":     survey.ID.String(),
				"status": survey.Status,
			})
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

func (s *Server) handleRunnerPlanningSay(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningSayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	session, err := s.loadPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, err)
		return
	}
	if err := session.AppendAgentMessage(database.DB(r.Context()), req.Text); err != nil {
		writeRunnerPlanningError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent"})
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
	case errors.Is(err, models.ErrFactoryPlanningSessionInvalid),
		errors.Is(err, models.ErrFactoryWorkOrderSurveyInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, models.ErrFactoryPlanningSessionNotFound),
		errors.Is(err, models.ErrFactoryWorkOrderSurveyNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		http.Error(w, "planning session not found", http.StatusNotFound)
	case errors.Is(err, models.ErrFactoryPlanningSessionEnded):
		http.Error(w, "planning session has ended", http.StatusConflict)
	default:
		log.WithError(err).Error("runner planning session failed")
		http.Error(w, "Lookup failed", http.StatusInternalServerError)
	}
}
