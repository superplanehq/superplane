package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	minWorkOrderSurveyHoldSeconds = 5
	maxWorkOrderSurveyHoldSeconds = 55
)

type createRunnerWorkOrderSurveyRequest struct {
	WorkOrderID    string                           `json:"work_order_id"`
	TimeoutSeconds int                              `json:"timeout_seconds"`
	Questions      []models.WorkOrderSurveyQuestion `json:"questions"`
}

type runnerWorkOrderSurveyResponse struct {
	ID      string                         `json:"id"`
	Status  string                         `json:"status"`
	Answers []models.WorkOrderSurveyAnswer `json:"answers"`
}

func (s *Server) handleCreateRunnerWorkOrderSurvey(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateWorkOrderSurveyRunner(w, r)
	if !ok {
		return
	}

	var req createRunnerWorkOrderSurveyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.WorkOrderID != "" && req.WorkOrderID != scope.WorkOrderID.String() {
		http.Error(w, "work_order_id does not match this run", http.StatusForbidden)
		return
	}

	db := database.DB(r.Context())
	var survey *models.FactoryWorkOrderSurvey
	var created bool
	err := db.Transaction(func(tx *gorm.DB) error {
		factoryModel, err := models.FindFactory(tx, scope.OrganizationID, scope.FactoryID)
		if err != nil {
			return err
		}
		order, err := factoryModel.FindWorkOrder(tx, scope.WorkOrderID)
		if err != nil {
			return err
		}
		var executionID *uuid.UUID
		if scope.ExecutionID != uuid.Nil {
			executionID = &scope.ExecutionID
		}
		survey, created, err = order.CreateSurvey(tx, models.FactoryWorkOrderSurveyParams{
			CanvasRunID:    scope.CanvasRunID,
			ExecutionID:    executionID,
			TimeoutSeconds: req.TimeoutSeconds,
			Questions:      req.Questions,
		})
		return err
	})
	if err != nil {
		writeRunnerWorkOrderSurveyError(w, err)
		return
	}

	if created {
		factories.PublishWorkOrderSurveyUpdated(scope.OrganizationID, scope.FactoryID, scope.WorkOrderID, survey.ID, true)
	}
	writeRunnerWorkOrderSurvey(w, http.StatusOK, survey)
}

func (s *Server) handleWaitRunnerWorkOrderSurvey(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateWorkOrderSurveyRunner(w, r)
	if !ok {
		return
	}

	surveyID, err := uuid.Parse(strings.TrimSpace(mux.Vars(r)["id"]))
	if err != nil {
		http.Error(w, "Invalid survey id", http.StatusBadRequest)
		return
	}

	hold := clampWorkOrderSurveyHoldSeconds(r.URL.Query().Get("hold_seconds"))
	deadline := time.Now().Add(time.Duration(hold) * time.Second)
	wait := runneraction.SubscribeWorkOrderSurvey(surveyID)
	defer runneraction.UnsubscribeWorkOrderSurvey(surveyID, wait)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	deadlineTimer := time.NewTimer(time.Until(deadline))
	defer deadlineTimer.Stop()

	for {
		survey, err := s.loadRunnerWorkOrderSurvey(r, scope, surveyID)
		if err != nil {
			writeRunnerWorkOrderSurveyError(w, err)
			return
		}
		if survey.Status != models.FactoryWorkOrderSurveyPending {
			writeRunnerWorkOrderSurvey(w, http.StatusOK, survey)
			return
		}
		if !time.Now().Before(deadline) {
			writeRunnerWorkOrderSurvey(w, http.StatusOK, survey)
			return
		}

		select {
		case <-r.Context().Done():
			return
		case <-wait:
		case <-ticker.C:
		case <-deadlineTimer.C:
		}
	}
}

func (s *Server) authenticateWorkOrderSurveyRunner(w http.ResponseWriter, r *http.Request) (*runneraction.WorkOrderSurveyScope, bool) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	scope, err := runneraction.ParseWorkOrderSurveyToken(s.jwt, token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return scope, true
}

func (s *Server) loadRunnerWorkOrderSurvey(
	r *http.Request,
	scope *runneraction.WorkOrderSurveyScope,
	surveyID uuid.UUID,
) (*models.FactoryWorkOrderSurvey, error) {
	db := database.DB(r.Context())
	survey, err := models.FindWorkOrderSurvey(db, scope.OrganizationID, surveyID)
	if err != nil {
		return nil, err
	}
	if survey.WorkOrderID != scope.WorkOrderID || survey.CanvasRunID != scope.CanvasRunID {
		return nil, models.ErrFactoryWorkOrderSurveyNotFound
	}
	if err := survey.ExpireIfDue(db, time.Now()); err != nil {
		return nil, err
	}
	if survey.Status != models.FactoryWorkOrderSurveyPending {
		runneraction.NotifyWorkOrderSurvey(survey.ID)
	}
	return survey, nil
}

func writeRunnerWorkOrderSurvey(w http.ResponseWriter, status int, survey *models.FactoryWorkOrderSurvey) {
	answers := []models.WorkOrderSurveyAnswer{}
	if len(survey.Answers) > 0 {
		answers = survey.Answers
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(runnerWorkOrderSurveyResponse{
		ID:      survey.ID.String(),
		Status:  survey.Status,
		Answers: answers,
	}); err != nil {
		log.WithError(err).Error("failed to encode work order survey response")
	}
}

func writeRunnerWorkOrderSurveyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrFactoryWorkOrderSurveyInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, models.ErrFactoryWorkOrderSurveyConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, models.ErrFactoryWorkOrderSurveyNotFound),
		errors.Is(err, models.ErrFactoryNotFound),
		errors.Is(err, models.ErrFactoryWorkOrderNotFound):
		http.Error(w, "work order survey not found", http.StatusNotFound)
	default:
		log.WithError(err).Error("runner work order survey failed")
		http.Error(w, "Lookup failed", http.StatusInternalServerError)
	}
}

func clampWorkOrderSurveyHoldSeconds(raw string) int {
	hold := 45
	if raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil {
			hold = parsed
		}
	}
	if hold < minWorkOrderSurveyHoldSeconds {
		return minWorkOrderSurveyHoldSeconds
	}
	if hold > maxWorkOrderSurveyHoldSeconds {
		return maxWorkOrderSurveyHoldSeconds
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
