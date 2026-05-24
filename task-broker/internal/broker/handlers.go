package broker

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/cwstream"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/webhook"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
)

// Server implements task-broker HTTP handlers.
type Server struct {
	Store   taskstore.Store
	Webhook *webhook.Sender
	Log     *slog.Logger

	TaskNotify   *WaitHub
	RunnerCancel *RunnerCancelHub

	TaskCloudWatchLogGroup        string
	TaskCloudWatchLogStreamPrefix string
	TaskCloudWatchRegion          string
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) registerFleet(w http.ResponseWriter, r *http.Request) {
	var req api.RegisterFleetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	f := &brokermodels.Fleet{
		ID:        req.ID,
		Labels:    taskstore.NormalizeLabels(req.Labels),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.Store.CreateFleet(r.Context(), f); err != nil {
		s.logErr("create fleet", err)
		writeError(w, http.StatusInternalServerError, "could not persist fleet")
		return
	}
	writeJSON(w, http.StatusCreated, fleetToResponse(f))
}

func (s *Server) listFleets(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListFleets(r.Context())
	if err != nil {
		s.logErr("list fleets", err)
		writeError(w, http.StatusInternalServerError, "could not list fleets")
		return
	}
	out := make([]api.FleetResponse, 0, len(list))
	for i := range list {
		out = append(out, *fleetToResponse(&list[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteFleet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	if err := s.Store.DeleteFleet(r.Context(), id); err != nil {
		s.logErr("delete fleet", err)
		writeError(w, http.StatusInternalServerError, "could not delete fleet")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func fleetToResponse(f *brokermodels.Fleet) *api.FleetResponse {
	if f == nil {
		return nil
	}
	return &api.FleetResponse{
		ID:        f.ID,
		Labels:    append([]string(nil), f.Labels...),
		CreatedAt: f.CreatedAt.Unix(),
	}
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req api.BrokerCreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.WebhookURL) == "" {
		writeError(w, http.StatusBadRequest, "webhook_url required")
		return
	}
	hasID := strings.TrimSpace(req.FleetID) != ""
	hasLabels := len(taskstore.NormalizeLabels(req.FleetLabels)) > 0
	switch {
	case hasID && hasLabels:
		writeError(w, http.StatusBadRequest, "specify either fleet_id or fleet_labels, not both")
		return
	case !hasID && !hasLabels:
		writeError(w, http.StatusBadRequest, "fleet_id or fleet_labels required")
		return
	}

	ctx := r.Context()
	var fleet *brokermodels.Fleet
	var err error
	if hasID {
		fleet, err = s.Store.GetFleet(ctx, strings.TrimSpace(req.FleetID))
	} else {
		fleet, err = s.Store.FindFleetByLabels(ctx, req.FleetLabels)
	}
	if err != nil {
		s.logErr("resolve fleet", err)
		writeError(w, http.StatusInternalServerError, "could not route fleet")
		return
	}
	if fleet == nil {
		writeError(w, http.StatusNotFound, "fleet not found")
		return
	}

	if msg := validateCreateTaskPayload(&req.CreateTaskRequest); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	hasArgv := len(req.Command) > 0
	var normalizedCmds []string
	for _, c := range req.Commands {
		c = strings.TrimSpace(c)
		if c != "" {
			normalizedCmds = append(normalizedCmds, c)
		}
	}
	hasShell := len(normalizedCmds) > 0

	mode := models.ExecutionHost
	switch strings.ToLower(strings.TrimSpace(req.ExecutionMode)) {
	case "", string(models.ExecutionHost):
		mode = models.ExecutionHost
	case string(models.ExecutionDocker):
		mode = models.ExecutionDocker
	default:
		writeError(w, http.StatusBadRequest, "invalid execution_mode")
		return
	}

	task := &models.Task{
		ID:            uuid.NewString(),
		FleetID:       fleet.ID,
		WebhookURL:    strings.TrimSpace(req.WebhookURL),
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: mode,
		DockerImage:   req.DockerImage,
		Environment:   api.CloneEnvironment(req.Environment),
	}
	if hasShell {
		task.Commands = normalizedCmds
	} else if hasArgv {
		task.Command = req.Command
	}
	if req.ExecutionTimeoutSeconds != nil {
		v := *req.ExecutionTimeoutSeconds
		task.ExecutionTimeoutSeconds = &v
	}
	if err := s.Store.CreateTask(ctx, task); err != nil {
		s.logErr("create task", err)
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	if s.TaskNotify != nil {
		s.TaskNotify.Notify()
	}
	writeJSON(w, http.StatusCreated, api.BrokerCreateTaskResponse{ID: task.ID})
}

func (s *Server) claimTask(w http.ResponseWriter, r *http.Request) {
	var req api.ClaimTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.RunnerID) == "" {
		writeError(w, http.StatusBadRequest, "runner_id required")
		return
	}
	if strings.TrimSpace(req.FleetID) == "" {
		writeError(w, http.StatusBadRequest, "fleet_id required")
		return
	}
	lease := time.Duration(req.LeaseSeconds) * time.Second
	if lease <= 0 {
		lease = 5 * time.Minute
	}

	task, err := s.Store.ClaimTask(r.Context(), req.RunnerID, req.FleetID, lease)
	if err != nil {
		s.logErr("claim task", err)
		writeError(w, http.StatusInternalServerError, "could not claim task")
		return
	}
	var payload *api.TaskPayload
	if task != nil {
		payload = api.TaskPayloadFrom(task)
	}
	writeJSON(w, http.StatusOK, api.ClaimTaskResponse{Task: payload})
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	task, err := s.Store.GetTask(r.Context(), id)
	if err != nil {
		s.logErr("get task", err)
		writeError(w, http.StatusInternalServerError, "could not load task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, taskStatusResponse(task, s))
}

func taskStatusResponse(task *models.Task, s *Server) api.TaskStatusResponse {
	mode := string(task.ExecutionMode)
	if mode == "" {
		mode = string(models.ExecutionHost)
	}
	resp := api.TaskStatusResponse{
		ID:              task.ID,
		Status:          string(task.Status),
		FleetID:         task.FleetID,
		CreatedAt:       task.CreatedAt.UTC(),
		ClaimedAt:       task.ClaimedAt,
		LeaseUntil:      task.LeaseUntil,
		RunnerID:        strings.TrimSpace(task.RunnerID),
		ExecutionMode:   mode,
		DockerImage:     strings.TrimSpace(task.DockerImage),
		Error:           task.ErrorMessage,
		CancelRequested: task.CancelRequested,
	}
	if g := strings.TrimSpace(s.TaskCloudWatchLogGroup); g != "" {
		resp.CloudWatchLogGroup = g
		resp.CloudWatchLogStream = cwstream.TaskLogStream(s.TaskCloudWatchLogStreamPrefix, task.ID)
	}
	resp.TaskLog = s.taskLogForTask(task.ID)
	if task.ExitCode != nil {
		ec := *task.ExitCode
		resp.ExitCode = &ec
	}
	if task.ExecutionTimeoutSeconds != nil {
		v := *task.ExecutionTimeoutSeconds
		resp.ExecutionTimeoutSeconds = &v
	}
	if strings.TrimSpace(task.ResultJSON) != "" {
		resp.Result = json.RawMessage(task.ResultJSON)
	}
	return resp
}

func (s *Server) completeTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	var req api.CompleteTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	runnerID := strings.TrimSpace(req.RunnerID)
	if runnerID == "" {
		runnerID = strings.TrimSpace(r.Header.Get("X-Runner-Id"))
	}
	if runnerID == "" {
		writeError(w, http.StatusBadRequest, "runner_id required (body or X-Runner-Id)")
		return
	}

	_, err := s.completeTaskCore(r.Context(), id, runnerID, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "wrong runner") {
			writeError(w, http.StatusConflict, "cannot complete task")
			return
		}
		s.logErr("complete task", err)
		writeError(w, http.StatusInternalServerError, "could not complete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	task, outcome, err := s.Store.RequestCancelTask(r.Context(), id)
	if err != nil {
		s.logErr("cancel task", err)
		writeError(w, http.StatusInternalServerError, "could not cancel task")
		return
	}
	switch outcome {
	case taskstore.CancelOutcomeNotFound:
		writeError(w, http.StatusNotFound, "task not found")
		return
	case taskstore.CancelOutcomeCanceledQueued:
		go s.DeliverWebhook(task)
	case taskstore.CancelOutcomeCancelRequested:
		rid := strings.TrimSpace(task.RunnerID)
		if s.RunnerCancel != nil && rid != "" {
			if !s.RunnerCancel.PushCancel(rid, task.ID) && s.Log != nil {
				s.Log.Debug("runner cancel ws push not delivered",
					slog.String("runner_id", rid), slog.String("task_id", task.ID))
			}
		}
	}
	if task == nil {
		writeError(w, http.StatusInternalServerError, "cancel task missing row")
		return
	}
	writeJSON(w, http.StatusOK, api.CancelTaskResponse{
		ID:     task.ID,
		State:  string(outcome),
		Status: string(task.Status),
	})
}

func (s *Server) completeTaskCore(ctx context.Context, taskID, runnerID string, req api.CompleteTaskRequest) (*models.Task, error) {
	resultJSON := ""
	if len(req.Result) > 0 {
		resultJSON = string(req.Result)
	}
	task, err := s.Store.CompleteTask(ctx, taskID, runnerID, req.ExitCode, resultJSON, req.Error, req.Canceled)
	if err != nil {
		return nil, err
	}
	go s.DeliverWebhook(task)
	return task, nil
}

func (s *Server) DeliverWebhook(task *models.Task) {
	if s.Webhook == nil || task == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	exit := 0
	if task.ExitCode != nil {
		exit = *task.ExitCode
	}
	payload := api.WebhookPayload{
		TaskID:   task.ID,
		Status:   string(task.Status),
		ExitCode: exit,
		Error:    task.ErrorMessage,
	}
	if g := strings.TrimSpace(s.TaskCloudWatchLogGroup); g != "" {
		payload.CloudWatchLogGroup = g
		payload.CloudWatchLogStream = cwstream.TaskLogStream(s.TaskCloudWatchLogStreamPrefix, task.ID)
	}
	payload.TaskLog = s.taskLogForTask(task.ID)
	if strings.TrimSpace(task.ResultJSON) != "" {
		payload.Result = json.RawMessage(task.ResultJSON)
	}
	if err := s.Webhook.Deliver(ctx, task.WebhookURL, payload); err != nil && s.Log != nil {
		s.Log.Warn("webhook delivery failed", slog.String("task_id", task.ID), slog.Any("err", err))
	}
}

func (s *Server) taskLogForTask(taskID string) *api.TaskLogSink {
	if g := strings.TrimSpace(s.TaskCloudWatchLogGroup); g != "" {
		stream := cwstream.TaskLogStream(s.TaskCloudWatchLogStreamPrefix, taskID)
		return api.TaskLogSinkCloudWatchFromParts(g, stream, s.TaskCloudWatchRegion)
	}
	return nil
}

func validateCreateTaskPayload(req *api.CreateTaskRequest) string {
	hasArgv := len(req.Command) > 0
	hasCmds := false
	for _, c := range req.Commands {
		if strings.TrimSpace(c) != "" {
			hasCmds = true
			break
		}
	}
	switch {
	case hasArgv && hasCmds:
		return "specify either command or commands, not both"
	case !hasArgv && !hasCmds:
		return "command or commands required"
	}
	mode := models.ExecutionMode(strings.ToLower(strings.TrimSpace(req.ExecutionMode)))
	switch mode {
	case "", models.ExecutionHost:
	case models.ExecutionDocker:
		if strings.TrimSpace(req.DockerImage) == "" {
			return "docker_image required for docker execution_mode"
		}
	default:
		return "invalid execution_mode"
	}
	if msg := api.ValidateExecutionTimeoutSeconds(req.ExecutionTimeoutSeconds); msg != "" {
		return msg
	}
	if msg := api.ValidateEnvironment(req.Environment); msg != "" {
		return msg
	}
	return ""
}

func (s *Server) logErr(msg string, err error) {
	if s.Log != nil {
		s.Log.Error(msg, slog.Any("err", err))
	}
}

func (s *Server) warn(msg string, attrs ...any) {
	if s.Log != nil {
		s.Log.Warn(msg, attrs...)
	}
}
