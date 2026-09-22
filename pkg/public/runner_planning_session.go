package public

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	minPlanningHoldSeconds   = 1
	maxPlanningHoldSeconds   = 60
	maxPlanningActivityBytes = 256 * 1024

	// statusClientClosedRequest mirrors the codes.Canceled mapping described in
	// pkg/grpc/errors/grpcerrors.go.
	statusClientClosedRequest = 499
)

type planningSurveyRequest struct {
	Questions []models.PlanningSessionSurveyQuestion `json:"questions"`
}

type planningSpecRequest struct {
	Body string `json:"body"`
}

// planningScoreRequest is the body for both the Clarity and the Confidence
// score endpoints.
type planningScoreRequest struct {
	Score   float64 `json:"score"`
	Summary string  `json:"summary"`
}

type planningAgentMessageRequest struct {
	Text       string `json:"text"`
	ActivityID string `json:"activity_id"`
}

// planningCreateTaskRequest is one task the agent splits off the draft it
// refines, after the user confirmed the split.
type planningCreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ActivityID  string `json:"activity_id"`
}

func (s *Server) handleRunnerPlanningActivity(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	activityID, err := uuid.Parse(mux.Vars(r)["activity_id"])
	if err != nil {
		http.Error(w, "Invalid activity ID", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPlanningActivityBytes)
	var snapshot models.PlanningSessionActivitySnapshot
	if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if snapshot.ActivityID != activityID.String() || snapshot.SchemaVersion != 2 || snapshot.Turn < 1 || snapshot.Sequence < 1 || snapshot.StartedAt < 1 {
		http.Error(w, "Invalid activity snapshot", http.StatusBadRequest)
		return
	}
	if !validPlanningActivityStatus(snapshot.Status) {
		http.Error(w, "Invalid activity status", http.StatusBadRequest)
		return
	}
	session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	startedAt := time.UnixMilli(snapshot.StartedAt)
	var completedAt *time.Time
	if snapshot.CompletedAt != nil {
		value := time.UnixMilli(*snapshot.CompletedAt)
		completedAt = &value
	}
	now := time.Now()
	activity := models.PlanningSessionActivity{
		ID:            activityID,
		SessionID:     session.ID,
		SchemaVersion: snapshot.SchemaVersion,
		Provider:      strings.TrimSpace(snapshot.Provider),
		Status:        snapshot.Status,
		LastSequence:  snapshot.Sequence,
		Snapshot:      datatypes.NewJSONType(snapshot),
		StartedAt:     startedAt,
		CompletedAt:   completedAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if activity.Provider == "" {
		http.Error(w, "Invalid activity provider", http.StatusBadRequest)
		return
	}
	if err := session.UpsertActivity(database.DB(r.Context()), activity); err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "stored"})
}

func validPlanningActivityStatus(status string) bool {
	return slices.Contains([]string{
		models.PlanningSessionActivityStatusRunning,
		models.PlanningSessionActivityStatusPassed,
		models.PlanningSessionActivityStatusFailed,
		models.PlanningSessionActivityStatusCancelled,
		models.PlanningSessionActivityStatusTimedOut,
		models.PlanningSessionActivityStatusInterrupted,
	}, status)
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

func (s *Server) loadAnalysisPlanningSessionForRunner(r *http.Request, scope *runneraction.PlanningSessionScope) (*models.FactoryPlanningSession, error) {
	session, err := s.loadPlanningSessionForRunner(r, scope)
	if err != nil {
		return nil, err
	}
	if !session.IsAnalysisSession() {
		return nil, models.ErrFactoryPlanningSessionInvalid
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
		if r.Context().Err() != nil {
			writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
			return
		}
		session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
		if err != nil {
			writePlanningWaitError(w, r, session, err)
			return
		}
		if session.State == models.PlanningSessionStateEnded {
			writeJSON(w, http.StatusOK, map[string]any{"status": "ended"})
			return
		}
		if session.WaitState == models.PlanningWaitResolved {
			result, consumed, err := consumeResolvedWait(session, database.DB(r.Context()))
			if err != nil {
				writePlanningWaitError(w, r, session, err)
				return
			}
			if consumed {
				if r.Context().Err() != nil {
					restorePlanningWait(session, result)
					writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
					return
				}
				text, err := mintPlanningWaitText(r.Context(), session, result)
				if err != nil {
					restorePlanningWait(session, result)
					writePlanningWaitError(w, r, session, err)
					return
				}
				body, bodyErr := planningWaitMessageBody(r.Context(), session, result, text)
				if bodyErr != nil {
					restorePlanningWait(session, result)
					writePlanningWaitError(w, r, session, bodyErr)
					return
				}
				if err := writeJSON(w, http.StatusOK, body); err != nil {
					restorePlanningWait(session, result)
				}
				return
			}
		}
		if err := beginPlanningWaitAndNotify(database.DB(r.Context()), session); err != nil {
			writePlanningWaitError(w, r, session, err)
			return
		}
		if !time.Now().Before(deadline) {
			writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
			return
		}
		select {
		case <-r.Context().Done():
			writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) handleRunnerPlanningSpec(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	if err := proposePlanningSpecAndNotify(database.DB(r.Context()), session, req.Body); err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "shown"})
}

func (s *Server) handleRunnerPlanningClarity(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerPlanningScore(w, r, (*models.FactoryPlanningSession).ProposeClarity)
}

func (s *Server) handleRunnerPlanningConfidence(w http.ResponseWriter, r *http.Request) {
	s.handleRunnerPlanningScore(w, r, (*models.FactoryPlanningSession).ProposeConfidence)
}

func (s *Server) handleRunnerPlanningScore(
	w http.ResponseWriter,
	r *http.Request,
	propose func(*models.FactoryPlanningSession, *gorm.DB, float64, string) error,
) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	if err := propose(session, database.DB(r.Context()), req.Score, req.Summary); err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	publishPlanningScore(session)
	writeJSON(w, http.StatusOK, map[string]any{"status": "shown"})
}

func publishPlanningScore(session *models.FactoryPlanningSession) {
	if session == nil || session.DraftWorkOrderID == nil {
		return
	}
	err := messages.PublishFactoryWorkOrderUpdated(
		session.FactoryID.String(),
		session.DraftWorkOrderID.String(),
		factoryevents.EventTypeOrderCheckReported,
	)
	if err != nil {
		log.WithError(err).Warn("failed to publish planning score update")
	}
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
	session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	if err := session.ProposeSurvey(database.DB(r.Context()), models.PlanningSessionSurvey{
		Questions: req.Questions,
	}); err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	messages.PublishPlanningBoardStatus(session)
	writeJSON(w, http.StatusOK, map[string]any{"status": "shown"})
}

func (s *Server) handleRunnerPlanningAgentMessage(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningAgentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	activityID, ok := parseOptionalActivityID(w, req.ActivityID)
	if !ok {
		return
	}
	if err := session.RecordAgentMessageForActivity(database.DB(r.Context()), req.Text, activityID); err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "shown"})
}

// handleRunnerPlanningCreateTask creates a new draft for part of the work
// the session refines. The new draft fans out to the Backlog automation
// like a task created by hand, so it gets its own refine session.
func (s *Server) handleRunnerPlanningCreateTask(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticatePlanningSessionRunner(w, r)
	if !ok {
		return
	}
	var req planningCreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	activityID, ok := parseOptionalActivityID(w, req.ActivityID)
	if !ok {
		return
	}
	session, err := s.loadAnalysisPlanningSessionForRunner(r, scope)
	if err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	db := database.DB(r.Context())
	factoryModel, err := models.FindFactory(db, session.OrganizationID, session.FactoryID)
	if err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	order, err := session.CreateSplitTask(db, factoryModel, models.PlanningSplitTask{
		Title:       req.Title,
		Description: req.Description,
	}, activityID)
	if err != nil {
		writeRunnerPlanningError(w, r, session, err)
		return
	}
	workersctx.EmitWorkOrderCreated(db, factoryModel, order)
	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryModel.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for split task %s", order.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "created",
		"work_order_id": order.ID.String(),
		"key":           factoryModel.WorkOrderKey(order.Number),
		"title":         order.Title,
	})
}

// parseOptionalActivityID reads an optional activity id from a request body.
// It writes the 400 response itself and reports false when the id is malformed.
func parseOptionalActivityID(w http.ResponseWriter, raw string) (uuid.UUID, bool) {
	if strings.TrimSpace(raw) == "" {
		return uuid.Nil, true
	}
	activityID, err := uuid.Parse(raw)
	if err != nil {
		http.Error(w, "Invalid activity ID", http.StatusBadRequest)
		return uuid.Nil, false
	}
	return activityID, true
}

func beginPlanningWaitAndNotify(db *gorm.DB, session *models.FactoryPlanningSession) error {
	alreadyWaiting := session.WaitState == models.PlanningWaitPending || session.WaitState == models.PlanningWaitResolved
	if err := session.BeginWait(db); err != nil {
		return err
	}
	if alreadyWaiting || session.WaitState != models.PlanningWaitPending {
		return nil
	}
	messages.PublishPlanningBoardStatus(session)
	if !hasOutstandingPlanningQuestion(session) {
		return nil
	}
	messages.PublishPlanningAgentQuestion(session)
	return nil
}

func proposePlanningSpecAndNotify(db *gorm.DB, session *models.FactoryPlanningSession, body string) error {
	hadSpec := messages.HasPlanningReadyPlan(db, session)
	if err := session.ProposeSpec(db, body); err != nil {
		return err
	}
	if hadSpec {
		return nil
	}
	messages.PublishPlanningPlanReady(db, session)
	return nil
}

func hasOutstandingPlanningQuestion(session *models.FactoryPlanningSession) bool {
	return len(session.CurrentSurvey().Questions) > 0
}

func planningWaitMessageBody(
	ctx context.Context,
	session *models.FactoryPlanningSession,
	result models.PlanningWaitResult,
	text string,
) (map[string]any, error) {
	body := map[string]any{
		"status":         result.Kind,
		"text":           text,
		"work_order_id":  result.WorkOrderID,
		"work_order_key": result.WorkOrderKey,
	}
	if session == nil || !session.IsAnalysisSession() {
		return body, nil
	}
	continuation, err := models.AnalysisContinuationText(database.DB(ctx), session)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(continuation) != "" {
		body["continuation"] = continuation
	}
	return body, nil
}

func mintPlanningWaitText(ctx context.Context, session *models.FactoryPlanningSession, result models.PlanningWaitResult) (string, error) {
	if result.Kind != models.PlanningWaitKindMessage {
		return result.Text, nil
	}
	if session.DraftWorkOrderID == nil {
		return result.Text, nil
	}
	if len(blob.FileIDsInMarkdown(result.Text)) == 0 {
		return result.Text, nil
	}
	rewritten, _, err := storedfiles.DescriptionForDispatch(
		ctx,
		database.DB(ctx),
		blob.Current(),
		session.OrganizationID,
		session.FactoryID,
		*session.DraftWorkOrderID,
		result.Text,
		blob.DispatchDownloadTTL(0),
	)
	if err != nil {
		return "", err
	}
	return rewritten, nil
}

func consumeResolvedWait(session *models.FactoryPlanningSession, tx *gorm.DB) (models.PlanningWaitResult, bool, error) {
	result, err := session.ConsumeWait(tx)
	if errors.Is(err, models.ErrFactoryPlanningWaitIdle) {
		return models.PlanningWaitResult{}, false, nil
	}
	if err != nil {
		return models.PlanningWaitResult{}, false, err
	}
	return result, true, nil
}

func restorePlanningWait(session *models.FactoryPlanningSession, result models.PlanningWaitResult) {
	if err := session.RestoreWait(database.DB(context.Background()), result); err != nil {
		log.WithError(err).Error("failed to restore planning wait after a dropped write")
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.WithError(err).Error("failed to encode planning session runner response")
		return err
	}
	return nil
}

func isPlanningRequestCanceled(r *http.Request, err error) bool {
	if r == nil {
		return false
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return errors.Is(r.Context().Err(), context.Canceled)
}

func isTransientPlanningWaitDBError(err error) bool {
	if err == nil {
		return false
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && (errors.Is(opErr.Err, syscall.ECONNRESET) || errors.Is(opErr.Err, syscall.EPIPE)) {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "connection reset by peer") ||
		strings.Contains(message, "broken pipe") ||
		strings.Contains(message, "driver: bad connection")
}

func writePlanningWaitError(w http.ResponseWriter, r *http.Request, session *models.FactoryPlanningSession, err error) {
	if isPlanningRequestCanceled(r, err) || isTransientPlanningWaitDBError(err) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
		return
	}
	writeRunnerPlanningError(w, r, session, err)
}

func writeRunnerPlanningError(w http.ResponseWriter, r *http.Request, session *models.FactoryPlanningSession, err error) {
	switch {
	case errors.Is(err, models.ErrFactoryPlanningSessionInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, models.ErrFactoryPlanningSessionNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		http.Error(w, "planning session not found", http.StatusNotFound)
	case errors.Is(err, models.ErrFactoryPlanningSessionEnded):
		http.Error(w, "planning session has ended", http.StatusConflict)
	case isPlanningRequestCanceled(r, err):
		log.WithError(err).WithField("route", resolveCriticalHTTPRoute(r)).Info("runner planning session client disconnected")
		w.WriteHeader(statusClientClosedRequest)
	default:
		log.WithError(err).Error("runner planning session failed")
		captureRunnerPlanningErrorToSentry(r, session, err)
		http.Error(w, "Lookup failed", http.StatusInternalServerError)
	}
}

func captureRunnerPlanningErrorToSentry(r *http.Request, session *models.FactoryPlanningSession, err error) {
	hub := sentry.CurrentHub()
	if hub == nil || hub.Client() == nil {
		return
	}
	hub.WithScope(func(scope *sentry.Scope) {
		applyRunnerPlanningErrorTags(scope, r, session, err)
		hub.CaptureException(err)
	})
}

func applyRunnerPlanningErrorTags(scope *sentry.Scope, r *http.Request, session *models.FactoryPlanningSession, err error) {
	if r != nil {
		if route := resolveCriticalHTTPRoute(r); route != "" {
			scope.SetTag("route", route)
		}
	}
	if session != nil {
		if session.ID != uuid.Nil {
			scope.SetTag("planning_session_id", session.ID.String())
		}
		if session.DraftWorkOrderID != nil && *session.DraftWorkOrderID != uuid.Nil {
			scope.SetTag("draft_work_order_id", session.DraftWorkOrderID.String())
		}
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code != "" {
		scope.SetTag("postgres_error_code", pgErr.Code)
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
