package public

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	runnerTaskListMax              = models.DefaultFactoryWorkOrderListLimit
	runnerTaskStateParamMax        = 3
	runnerTaskTitleMaxRunes        = 256
	runnerTaskDescriptionMaxRunes  = 20000
	runnerArtifactTitleMaxRunes    = 1024
	runnerTaskCreateBodyMaxBytes   = 128 * 1024
	runnerTaskMutationBodyMaxBytes = 8 * 1024
)

type runnerCreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type runnerCloseTaskRequest struct {
	Result string `json:"result"`
}

type runnerRemoveArtifactRequest struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}

type runnerTaskView struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Result string `json:"result,omitempty"`
	URL    string `json:"url"`
}

func (s *Server) handleRunnerListTasks(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeTasksRead)
	if !ok {
		return
	}

	states, err := runnerTaskStates(r.URL.Query()["state"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	limit, err := runnerListLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	factoryModel, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}

	orders, err := factoryModel.ListWorkOrderSummaries(db, models.ListFactoryWorkOrdersFilters{
		States:          states,
		Limit:           limit,
		ExcludeArchived: true,
	})
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}

	tasks := make([]runnerTaskView, 0, len(orders))
	for i := range orders {
		tasks = append(tasks, runnerTaskView{
			ID:    orders[i].ID.String(),
			Key:   factoryModel.WorkOrderKey(orders[i].Number),
			Title: orders[i].Title,
			State: orders[i].State,
			URL:   runnerTaskURL(s.BaseURL, factoryModel, orders[i].Number),
		})
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func (s *Server) handleRunnerCreateTask(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeTasksCreate)
	if !ok {
		return
	}

	var request runnerCreateTaskRequest
	if err := decodeRunnerJSON(w, r, &request, runnerTaskCreateBodyMaxBytes); err != nil {
		return
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(title) > runnerTaskTitleMaxRunes {
		http.Error(w, fmt.Sprintf("title must be at most %d characters", runnerTaskTitleMaxRunes), http.StatusBadRequest)
		return
	}
	description := strings.TrimSpace(request.Description)
	if utf8.RuneCountInString(description) > runnerTaskDescriptionMaxRunes {
		http.Error(w, fmt.Sprintf("description must be at most %d characters", runnerTaskDescriptionMaxRunes), http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	factoryModel, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}

	var order *models.FactoryWorkOrder
	err = db.Transaction(func(tx *gorm.DB) error {
		created, createErr := factoryModel.CreateWorkOrder(tx, title, description, nil, nil, nil)
		if createErr != nil {
			return createErr
		}
		order = created
		return nil
	})
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}

	workersctx.EmitWorkOrderCreated(db, factoryModel, order)
	publishRunnerTaskUpdate(factoryModel, order, factoryevents.EventTypeOrderStatusUpdated)
	_ = writeJSON(w, http.StatusCreated, runnerTaskViewFrom(s.BaseURL, factoryModel, order))
}

func (s *Server) handleRunnerCloseTask(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopeTasksClose)(w, r)
	if !ok {
		return
	}

	var request runnerCloseTaskRequest
	if err := decodeRunnerJSON(w, r, &request, runnerTaskMutationBodyMaxBytes); err != nil {
		return
	}
	result, err := runnerCloseResult(request.Result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	fromState := target.order.State
	// Pass this run so close does not cancel the agent that called the tool.
	changed, err := target.order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState:  models.FactoryWorkOrderStateClosed,
		Result:   result,
		Run:      &factoryevents.RunRef{ID: target.canvasRunID},
		App:      &factoryevents.AppRef{ID: target.canvasID},
		SkipSame: true,
	})
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}
	if !changed {
		http.Error(w, "Task is closed or archived", http.StatusConflict)
		return
	}
	order := target.order

	publishRunnerTaskUpdate(target.factory, order, factoryevents.EventTypeOrderStatusUpdated)
	notification := messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: target.factory.OrganizationID.String(),
		FactoryID:      target.factory.ID.String(),
		OrderID:        order.ID.String(),
		EventType:      factoryevents.EventTypeOrderStatusUpdated,
		FromState:      fromState,
		ToState:        models.FactoryWorkOrderStateClosed,
		Result:         result,
	}
	if err := notification.Publish(); err != nil {
		log.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
	}
	messages.PublishWorkOrderRunCancellations(db, order)
	_ = writeJSON(w, http.StatusOK, runnerTaskViewFrom(s.BaseURL, target.factory, order))
}

func (s *Server) handleRunnerRemoveArtifact(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopeArtifactsWrite)(w, r)
	if !ok {
		return
	}

	var request runnerRemoveArtifactRequest
	if err := decodeRunnerJSON(w, r, &request, runnerTaskMutationBodyMaxBytes); err != nil {
		return
	}
	key := strings.TrimSpace(request.Key)
	title := strings.TrimSpace(request.Title)
	if len(key) > models.MaxFactoryWorkOrderArtifactKeyBytes {
		http.Error(w, fmt.Sprintf("key must be at most %d bytes", models.MaxFactoryWorkOrderArtifactKeyBytes), http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(title) > runnerArtifactTitleMaxRunes {
		http.Error(w, fmt.Sprintf("title must be at most %d characters", runnerArtifactTitleMaxRunes), http.StatusBadRequest)
		return
	}

	removed, err := target.order.RemoveArtifact(database.DB(r.Context()), key, title, nil)
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}

	publishRunnerTaskUpdate(target.factory, target.order, factoryevents.EventTypeOrderArtifactsCleared)
	_ = writeJSON(w, http.StatusOK, map[string]any{
		"id":    removed.ID.String(),
		"title": artifactTitle(removed.Data),
	})
}

func runnerTaskViewFrom(baseURL string, factoryModel *models.Factory, order *models.FactoryWorkOrder) runnerTaskView {
	result := order.Result
	if result == models.FactoryWorkOrderResultManualRejected {
		result = models.FactoryWorkOrderResultRejected
	}
	return runnerTaskView{
		ID:     order.ID.String(),
		Key:    factoryModel.WorkOrderKey(order.Number),
		Title:  order.Title,
		State:  order.State,
		Result: result,
		URL:    runnerTaskURL(baseURL, factoryModel, order.Number),
	}
}

func runnerTaskURL(baseURL string, factoryModel *models.Factory, number int64) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return fmt.Sprintf("%s/%s/workspaces/%s/task/%d", base, factoryModel.OrganizationID, factoryModel.RouteSegment(), number)
}

func runnerTaskStates(values []string) ([]string, error) {
	if len(values) > runnerTaskStateParamMax {
		return nil, fmt.Errorf("state must be draft, open, or closed")
	}
	seen := make(map[string]struct{}, len(values))
	states := make([]string, 0, len(values))
	for _, value := range values {
		state := strings.ToLower(strings.TrimSpace(value))
		switch state {
		case models.FactoryWorkOrderStateDraft, models.FactoryWorkOrderStateOpen, models.FactoryWorkOrderStateClosed:
		default:
			return nil, fmt.Errorf("state must be draft, open, or closed")
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		states = append(states, state)
	}
	return states, nil
}

func runnerListLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return runnerTaskListMax, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}
	if value > runnerTaskListMax {
		return runnerTaskListMax, nil
	}
	return value, nil
}

func runnerCloseResult(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "completed":
		return models.FactoryWorkOrderResultCompleted, nil
	case "rejected":
		return models.FactoryWorkOrderResultManualRejected, nil
	case "failed":
		return models.FactoryWorkOrderResultFailed, nil
	default:
		return "", fmt.Errorf("result must be completed, rejected, or failed")
	}
}

func decodeRunnerJSON(w http.ResponseWriter, r *http.Request, dest any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(dest)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return nil
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		http.Error(w, "Request body is too large", http.StatusRequestEntityTooLarge)
		return err
	}
	http.Error(w, "Invalid JSON body", http.StatusBadRequest)
	return err
}

func publishRunnerTaskUpdate(factoryModel *models.Factory, order *models.FactoryWorkOrder, reason string) {
	if err := messages.PublishFactoryWorkOrderUpdated(factoryModel.ID.String(), order.ID.String(), reason); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
}

func writeRunnerTaskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrFactoryWorkOrderArtifactNotFound):
		http.Error(w, "Artifact not found", http.StatusNotFound)
	case errors.Is(err, models.ErrFactoryWorkOrderArtifactInvalid), errors.Is(err, models.ErrFactoryWorkOrderTitleRequired):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, models.ErrFactoryWorkOrderInvalidState):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		log.WithError(err).Error("runner task request failed")
		http.Error(w, "Task request failed", http.StatusInternalServerError)
	}
}

func artifactTitle(data datatypes.JSON) string {
	fields := map[string]any{}
	if err := json.Unmarshal(json.RawMessage(data), &fields); err != nil {
		return ""
	}
	title, _ := fields["title"].(string)
	return strings.TrimSpace(title)
}
