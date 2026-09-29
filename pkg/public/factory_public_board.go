package public

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
	"gorm.io/gorm"
)

const publicBoardPageSize = 100
const publicBoardMaxPages = 10

var errPublicBoardNotFound = errors.New("public board not found")

type publicBoard struct {
	WorkspaceName  string         `json:"workspaceName"`
	WorkspaceKey   string         `json:"workspaceKey"`
	ShowClarity    bool           `json:"showClarity"`
	ShowConfidence bool           `json:"showConfidence"`
	LineName       string         `json:"lineName"`
	Columns        []publicColumn `json:"columns"`
}

type publicColumn struct {
	Key    string       `json:"key"`
	Title  string       `json:"title"`
	Color  string       `json:"color,omitempty"`
	Labels []string     `json:"labels"`
	Cards  []publicCard `json:"cards"`
}

type publicCard struct {
	Title         string             `json:"title"`
	CreatedAt     time.Time          `json:"createdAt"`
	AssigneeName  string             `json:"assigneeName,omitempty"`
	Confidence    *int               `json:"confidence,omitempty"`
	Clarity       *int               `json:"clarity,omitempty"`
	PullRequest   *publicPullRequest `json:"pullRequest,omitempty"`
	Status        string             `json:"status,omitempty"`
	AgentQuestion bool               `json:"agentQuestion,omitempty"`
}

type publicPullRequest struct {
	Number     int64  `json:"number"`
	State      string `json:"state"`
	Mergeable  bool   `json:"mergeable"`
	ExtraCount int    `json:"extraCount"`
}

func (s *Server) handlePublicFactoryBoard(w http.ResponseWriter, r *http.Request) {
	board, err := loadPublicFactoryBoard(r.Context(), mux.Vars(r))
	if err != nil {
		if errors.Is(err, errPublicBoardNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		log.WithError(err).Error("failed to load public factory board")
		http.Error(w, "Failed to load board", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(board); err != nil {
		log.WithError(err).Error("failed to write public factory board")
	}
}

func (s *Server) handlePublicFactoryBoardWebSocket(w http.ResponseWriter, r *http.Request) {
	_, _, err := authorizePublicFactoryLine(r.Context(), mux.Vars(r))
	if err != nil {
		outcome := telemetry.WebSocketConnectionOutcomeAuthError
		if !errors.Is(err, errPublicBoardNotFound) {
			log.WithError(err).Error("failed to authorize public factory board socket")
			http.Error(w, "Failed to load board", http.StatusInternalServerError)
			return
		}
		telemetry.RecordWebSocketConnectionOutcome(r.Context(), ws.KindPublic, outcome)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	lineID := mux.Vars(r)["lineId"]
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		telemetry.RecordWebSocketConnectionOutcome(r.Context(), ws.KindPublic, telemetry.WebSocketConnectionOutcomeUpgradeError)
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.WithError(err).Error("failed to upgrade public factory board socket")
		}
		return
	}

	client := s.wsHub.NewClient(conn, eventdistributer.PublicLineTopic(lineID))
	<-client.Done
}

func broadcastPublicFactoryBoard(hub *ws.Hub, factoryID string) {
	if hub == nil {
		return
	}
	id, err := uuid.Parse(factoryID)
	if err != nil {
		return
	}

	db := database.DB(context.Background())
	var factory models.Factory
	if err := db.Where("id = ?", id).First(&factory).Error; err != nil {
		return
	}
	lines, err := factory.ListLines(db)
	if err != nil {
		return
	}

	open := factory.Public && factory.OnboardingCompletedAt != nil
	if open {
		enabled, featureErr := models.HasExperimentalFeature(factory.OrganizationID, features.FeatureFactories)
		if featureErr != nil || !enabled {
			open = false
		}
	}

	message := eventdistributer.PublicBoardChangedMessage()
	for _, line := range lines {
		topic := eventdistributer.PublicLineTopic(line.ID.String())
		if !open {
			hub.CloseTopic(topic)
			continue
		}
		hub.BroadcastToWorkflow(topic, message)
	}
}

func loadPublicFactoryBoard(ctx context.Context, vars map[string]string) (*publicBoard, error) {
	factory, line, err := authorizePublicFactoryLine(ctx, vars)
	if err != nil {
		return nil, err
	}
	return buildPublicBoard(database.DB(ctx), factory, line)
}

func authorizePublicFactoryLine(ctx context.Context, vars map[string]string) (*models.Factory, *models.FactoryLine, error) {
	db := database.DB(ctx)
	org, err := models.FindOrganizationByIDOrSlug(db, vars["org"])
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errPublicBoardNotFound
		}
		return nil, nil, err
	}

	enabled, err := models.HasExperimentalFeature(org.ID, features.FeatureFactories)
	if err != nil {
		return nil, nil, err
	}
	if !enabled {
		return nil, nil, errPublicBoardNotFound
	}

	factory, err := models.FindFactoryByKey(db, org.ID, models.NormalizeFactoryKey(vars["key"]))
	if err != nil {
		if errors.Is(err, models.ErrFactoryNotFound) {
			return nil, nil, errPublicBoardNotFound
		}
		return nil, nil, err
	}
	if !factory.Public || factory.OnboardingCompletedAt == nil {
		return nil, nil, errPublicBoardNotFound
	}

	lineID, err := uuid.Parse(vars["lineId"])
	if err != nil {
		return nil, nil, errPublicBoardNotFound
	}
	line, err := factory.FindLine(db, lineID)
	if err != nil {
		if errors.Is(err, models.ErrFactoryLineNotFound) {
			return nil, nil, errPublicBoardNotFound
		}
		return nil, nil, err
	}

	return factory, line, nil
}

func buildPublicBoard(db *gorm.DB, factory *models.Factory, line *models.FactoryLine) (*publicBoard, error) {
	orders, err := listPublicBoardOrders(db, factory, line.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(orders))
	for i := range orders {
		ids[i] = orders[i].ID
	}

	dispatches, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, ids)
	if err != nil {
		return nil, err
	}
	checks, err := models.ListChecksForWorkOrders(db, ids)
	if err != nil {
		return nil, err
	}
	pullRequests, err := models.ListPullRequestsByWorkOrderIDs(db, ids)
	if err != nil {
		return nil, err
	}
	sessions, err := models.ListAnalysisPlanningSessionsForWorkOrders(db, ids)
	if err != nil {
		return nil, err
	}
	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return nil, err
	}
	intakes, err := factory.ListIntakes(db)
	if err != nil {
		return nil, err
	}

	return assemblePublicBoard(factory, line, orders, dispatches, checks, pullRequests, sessions, canvases, intakes), nil
}

func listPublicBoardOrders(db *gorm.DB, factory *models.Factory, lineID uuid.UUID) ([]models.FactoryWorkOrder, error) {
	var orders []models.FactoryWorkOrder
	var before *uuid.UUID
	for page := 0; page < publicBoardMaxPages; page++ {
		batch, err := factory.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{
			LineID:      &lineID,
			PublicBoard: true,
			Limit:       publicBoardPageSize,
			BeforeID:    before,
		})
		if err != nil {
			return nil, err
		}
		orders = append(orders, batch...)
		if len(batch) < publicBoardPageSize {
			break
		}
		cursor := batch[len(batch)-1].ID
		before = &cursor
	}
	return orders, nil
}

func assemblePublicBoard(
	factory *models.Factory,
	line *models.FactoryLine,
	orders []models.FactoryWorkOrder,
	dispatches map[uuid.UUID][]models.FactoryWorkOrderLineDispatchRecord,
	checks map[uuid.UUID][]models.FactoryWorkOrderCheck,
	pullRequests map[uuid.UUID][]models.FactoryPullRequest,
	sessions map[uuid.UUID]*models.FactoryPlanningSession,
	canvases []models.Canvas,
	intakes []models.FactoryIntake,
) *publicBoard {
	planning := factory.Planning()
	names := map[uuid.UUID]string{}
	columnKey := map[uuid.UUID]string{}
	for _, canvas := range canvases {
		names[canvas.ID] = canvas.Name
		if canvas.ColumnKey != nil {
			columnKey[canvas.ID] = *canvas.ColumnKey
		}
	}

	steps := []models.FactoryLineStep(line.Steps)
	columns := []publicColumn{{
		Key:    "backlog",
		Title:  "Backlog",
		Color:  line.ColumnColorsValue()["backlog"],
		Labels: intakeLabels(intakes),
		Cards:  []publicCard{},
	}}
	for index, step := range steps {
		if isDoneStep(names[step.AppID], columnKey[step.AppID]) {
			continue
		}
		key := "phase-" + strconv.Itoa(index)
		title := strings.TrimSpace(names[step.AppID])
		if title == "" {
			title = "Step"
		}
		columns = append(columns, publicColumn{
			Key:    key,
			Title:  title,
			Color:  line.ColumnColorsValue()[key],
			Labels: []string{"Runs the " + title + " agent"},
			Cards:  []publicCard{},
		})
	}
	columns = append(columns,
		publicColumn{
			Key:    "verify",
			Title:  "Verify",
			Color:  line.ColumnColorsValue()["verify"],
			Labels: labelsForColumn(canvases, models.CanvasColumnKeyVerify),
			Cards:  []publicCard{},
		},
		publicColumn{
			Key:    "done",
			Title:  "Done",
			Color:  line.ColumnColorsValue()["done"],
			Labels: labelsForColumn(canvases, models.CanvasColumnKeyDone),
			Cards:  []publicCard{},
		},
	)

	byKey := map[string]*publicColumn{}
	for i := range columns {
		byKey[columns[i].Key] = &columns[i]
	}

	for i := range orders {
		order := &orders[i]
		key := publicCardColumn(order, line.ID, steps, names, columnKey, dispatches[order.ID])
		if key == "" {
			continue
		}
		column := byKey[key]
		if column == nil {
			continue
		}
		column.Cards = append(column.Cards, publicCardFromOrder(order, planning, checks[order.ID], pullRequests[order.ID], sessions[order.ID], dispatchesForLine(dispatches[order.ID], line.ID)))
	}

	return &publicBoard{
		WorkspaceName:  factory.Name,
		WorkspaceKey:   factory.Key,
		ShowClarity:    planning.Clarity,
		ShowConfidence: planning.Confidence,
		LineName:       line.Name,
		Columns:        columns,
	}
}

func dispatchesForLine(
	dispatches []models.FactoryWorkOrderLineDispatchRecord,
	lineID uuid.UUID,
) []models.FactoryWorkOrderLineDispatchRecord {
	matched := make([]models.FactoryWorkOrderLineDispatchRecord, 0, len(dispatches))
	for _, dispatch := range dispatches {
		if dispatch.LineID == lineID {
			matched = append(matched, dispatch)
		}
	}
	return matched
}

func publicCardColumn(
	order *models.FactoryWorkOrder,
	lineID uuid.UUID,
	steps []models.FactoryLineStep,
	names map[uuid.UUID]string,
	columnKey map[uuid.UUID]string,
	dispatches []models.FactoryWorkOrderLineDispatchRecord,
) string {
	switch order.State {
	case models.FactoryWorkOrderStateDraft:
		return "backlog"
	case models.FactoryWorkOrderStateClosed:
		if order.Result != models.FactoryWorkOrderResultCompleted && order.Result != models.FactoryWorkOrderResultFailed {
			return ""
		}
		return "done"
	}

	dispatch := newestDispatch(dispatches, lineID)
	if dispatch == nil {
		return "backlog"
	}
	if dispatch.QueueItem != nil {
		return phaseOrDone(dispatch.QueueItem.StepIndex, steps, names, columnKey)
	}

	execution := currentExecution(dispatch.Executions)
	if execution == nil {
		return "backlog"
	}
	lastStage := lastStageIndex(steps, names, columnKey)
	if execution.Status == models.FactoryWorkOrderExecutionStatusFinished &&
		executionResult(execution) == models.CanvasRunResultPassed &&
		lastStage >= 0 &&
		execution.StepIndex >= lastStage {
		return "verify"
	}
	return phaseOrDone(execution.StepIndex, steps, names, columnKey)
}

func phaseOrDone(stepIndex int, steps []models.FactoryLineStep, names map[uuid.UUID]string, columnKey map[uuid.UUID]string) string {
	if stepIndex < 0 || stepIndex >= len(steps) {
		return "backlog"
	}
	step := steps[stepIndex]
	if isDoneStep(names[step.AppID], columnKey[step.AppID]) {
		return "done"
	}
	return "phase-" + strconv.Itoa(stepIndex)
}

func lastStageIndex(steps []models.FactoryLineStep, names map[uuid.UUID]string, columnKey map[uuid.UUID]string) int {
	last := -1
	for index, step := range steps {
		if isDoneStep(names[step.AppID], columnKey[step.AppID]) {
			continue
		}
		last = index
	}
	return last
}

func isDoneStep(name, column string) bool {
	if strings.EqualFold(strings.TrimSpace(name), "done") {
		return true
	}
	return column == models.CanvasColumnKeyDone
}

func newestDispatch(dispatches []models.FactoryWorkOrderLineDispatchRecord, lineID uuid.UUID) *models.FactoryWorkOrderLineDispatchRecord {
	var newest *models.FactoryWorkOrderLineDispatchRecord
	for i := range dispatches {
		dispatch := &dispatches[i]
		if dispatch.LineID != lineID {
			continue
		}
		if newest == nil || dispatch.CreatedAt.After(newest.CreatedAt) {
			newest = dispatch
		}
	}
	return newest
}

func currentExecution(executions []models.FactoryWorkOrderExecutionRecord) *models.FactoryWorkOrderExecutionRecord {
	var best *models.FactoryWorkOrderExecutionRecord
	for i := range executions {
		execution := &executions[i]
		if best == nil || preferExecution(execution, best) {
			best = execution
		}
	}
	return best
}

func preferExecution(candidate, incumbent *models.FactoryWorkOrderExecutionRecord) bool {
	candidateActive := isActiveExecution(&candidate.FactoryWorkOrderExecution)
	incumbentActive := isActiveExecution(&incumbent.FactoryWorkOrderExecution)
	if candidateActive != incumbentActive {
		return candidateActive
	}
	if candidate.StepIndex != incumbent.StepIndex {
		return candidate.StepIndex > incumbent.StepIndex
	}
	return candidate.UpdatedAt.After(incumbent.UpdatedAt)
}

func isActiveExecution(execution *models.FactoryWorkOrderExecution) bool {
	return execution.Status == models.FactoryWorkOrderExecutionStatusPending ||
		execution.Status == models.FactoryWorkOrderExecutionStatusRunning
}

func executionResult(execution *models.FactoryWorkOrderExecutionRecord) string {
	if execution.Result != "" {
		return execution.Result
	}
	return execution.RunResult
}

func publicCardFromOrder(
	order *models.FactoryWorkOrder,
	planning models.FactoryPlanning,
	checks []models.FactoryWorkOrderCheck,
	pullRequests []models.FactoryPullRequest,
	session *models.FactoryPlanningSession,
	dispatches []models.FactoryWorkOrderLineDispatchRecord,
) publicCard {
	card := publicCard{
		Title:        order.Title,
		CreatedAt:    order.CreatedAt,
		AssigneeName: assigneeName(order),
		PullRequest:  pickPublicPullRequest(pullRequests),
		Status:       publicStatus(order, dispatches),
	}
	if order.State == models.FactoryWorkOrderStateDraft {
		if planning.Confidence {
			card.Confidence = scoreForCheck(checks, "Confidence score")
		}
		if planning.Clarity {
			card.Clarity = scoreForCheck(checks, "Clarity score")
		}
		card.AgentQuestion = sessionHasAgentQuestion(session)
	}
	return card
}

func assigneeName(order *models.FactoryWorkOrder) string {
	if len(order.Assignees) == 0 || order.Assignees[0].User == nil {
		return ""
	}
	return strings.TrimSpace(order.Assignees[0].User.Name)
}

func scoreForCheck(checks []models.FactoryWorkOrderCheck, name string) *int {
	for _, check := range checks {
		if check.Name != name {
			continue
		}
		score := check.Score
		if score > 5 {
			score = (score / 100) * 5
		}
		rounded := int(math.Round(score))
		if rounded < 0 {
			rounded = 0
		}
		if rounded > 5 {
			rounded = 5
		}
		return &rounded
	}
	return nil
}

func pickPublicPullRequest(pullRequests []models.FactoryPullRequest) *publicPullRequest {
	if len(pullRequests) == 0 {
		return nil
	}
	best := pullRequests[0]
	for _, candidate := range pullRequests[1:] {
		if pullRequestRank(candidate) < pullRequestRank(best) ||
			(pullRequestRank(candidate) == pullRequestRank(best) && candidate.Number > best.Number) {
			best = candidate
		}
	}
	extra := len(pullRequests) - 1
	return &publicPullRequest{
		Number:     best.Number,
		State:      best.State,
		Mergeable:  best.Mergeable && best.State == models.FactoryPullRequestStateOpen,
		ExtraCount: extra,
	}
}

func pullRequestRank(pullRequest models.FactoryPullRequest) int {
	switch pullRequest.State {
	case models.FactoryPullRequestStateOpen:
		return 0
	case models.FactoryPullRequestStateDraft:
		return 1
	case models.FactoryPullRequestStateMerged:
		return 2
	default:
		return 3
	}
}

func publicStatus(order *models.FactoryWorkOrder, dispatches []models.FactoryWorkOrderLineDispatchRecord) string {
	if order.State == models.FactoryWorkOrderStateClosed && order.Result == models.FactoryWorkOrderResultFailed {
		return "failed"
	}
	if order.State != models.FactoryWorkOrderStateOpen && order.State != models.FactoryWorkOrderStateDraft {
		return ""
	}
	if hasActiveDispatch(dispatches) {
		return ""
	}
	execution := latestExecution(dispatches)
	if execution != nil && executionResult(execution) == models.CanvasRunResultFailed {
		return "failed"
	}
	if execution != nil && executionResult(execution) == models.CanvasRunResultCancelled {
		return "stopped"
	}
	notes, err := order.StatusNotes()
	if err == nil && len(notes) > 0 {
		return "approval"
	}
	return ""
}

func hasActiveDispatch(dispatches []models.FactoryWorkOrderLineDispatchRecord) bool {
	for i := range dispatches {
		if dispatches[i].State == models.FactoryWorkOrderLineDispatchStateActive {
			return true
		}
	}
	return false
}

func latestExecution(dispatches []models.FactoryWorkOrderLineDispatchRecord) *models.FactoryWorkOrderExecutionRecord {
	var latest *models.FactoryWorkOrderExecutionRecord
	for i := range dispatches {
		for j := range dispatches[i].Executions {
			execution := &dispatches[i].Executions[j]
			if latest == nil || execution.UpdatedAt.After(latest.UpdatedAt) {
				latest = execution
			}
		}
	}
	return latest
}

func sessionHasAgentQuestion(session *models.FactoryPlanningSession) bool {
	if session == nil {
		return false
	}
	for _, question := range session.CurrentSurvey().Questions {
		if strings.TrimSpace(question.Prompt) != "" && len(question.Options) > 0 {
			return true
		}
	}
	return false
}

func intakeLabels(intakes []models.FactoryIntake) []string {
	labels := make([]string, 0, len(intakes))
	for _, intake := range intakes {
		name := intakeSourceName(intake.Source)
		if name == "" {
			continue
		}
		labels = append(labels, "Listens to "+name)
	}
	return labels
}

func intakeSourceName(source string) string {
	switch source {
	case models.FactoryIntakeSourceGitHubIssues:
		return "GitHub issues"
	case models.FactoryIntakeSourceSentryExceptions:
		return "Sentry exceptions"
	case models.FactoryIntakeSourcePagerDutyIncidents:
		return "PagerDuty incidents"
	case models.FactoryIntakeSourceProductiveTasks:
		return "Productive tasks"
	case models.FactoryIntakeSourceJiraIssues:
		return "Jira issues"
	case models.FactoryIntakeSourceDependabotAlerts:
		return "Dependabot alerts"
	case models.FactoryIntakeSourceDatadog:
		return "Datadog errors"
	default:
		return ""
	}
}

func labelsForColumn(canvases []models.Canvas, column string) []string {
	labels := []string{}
	for _, canvas := range canvases {
		if canvas.ColumnKey == nil || *canvas.ColumnKey != column {
			continue
		}
		name := strings.TrimSpace(canvas.Name)
		if name != "" {
			labels = append(labels, name)
		}
	}
	return labels
}
