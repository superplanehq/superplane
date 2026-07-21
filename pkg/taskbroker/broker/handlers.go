package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	brokermetrics "github.com/superplanehq/superplane/pkg/taskbroker/metrics"
	brokermodels "github.com/superplanehq/superplane/pkg/taskbroker/models"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/cwstream"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/webhook"
	taskstore "github.com/superplanehq/superplane/pkg/taskbroker/store"
)

// Server implements task-broker HTTP handlers.
type Server struct {
	Store   taskstore.Store
	Webhook *webhook.Sender
	Log     *slog.Logger
	Metrics *brokermetrics.BrokerMetrics

	TaskNotify   *WaitHub
	RunnerCancel *RunnerCancelHub
	RunnerDrain  *RunnerDrainHub

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
		ID:          req.ID,
		Provisioner: strings.TrimSpace(req.Provisioner),
		Arch:        strings.TrimSpace(req.Arch),
		Size:        strings.TrimSpace(req.Size),
		CreatedAt:   time.Now().UTC(),
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

func (s *Server) getFleetTaskCounts(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	fleet, err := s.Store.GetFleet(r.Context(), id)
	if err != nil {
		s.logErr("get fleet for counts", err)
		writeError(w, http.StatusInternalServerError, "could not load fleet")
		return
	}
	if fleet == nil {
		writeError(w, http.StatusNotFound, "fleet not found")
		return
	}
	queued, claimed, err := s.Store.CountTasksByFleet(r.Context(), id)
	if err != nil {
		s.logErr("count tasks by fleet", err)
		writeError(w, http.StatusInternalServerError, "could not count tasks")
		return
	}
	claimedRunnerIDs, err := s.Store.ClaimedRunnerIDsByFleet(r.Context(), id)
	if err != nil {
		s.logErr("claimed runner ids by fleet", err)
		writeError(w, http.StatusInternalServerError, "could not load claimed runner ids")
		return
	}
	writeJSON(w, http.StatusOK, api.FleetTaskCountsResponse{
		Queued:           queued,
		Claimed:          claimed,
		ClaimedRunnerIDs: claimedRunnerIDs,
	})
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

func (s *Server) drainRunners(w http.ResponseWriter, r *http.Request) {
	var req api.DrainRunnersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.FleetID = strings.TrimSpace(req.FleetID)
	if req.FleetID == "" {
		writeError(w, http.StatusBadRequest, "fleet_id required")
		return
	}
	if len(req.RunnerIDs) == 0 {
		writeError(w, http.StatusBadRequest, "runner_ids required")
		return
	}
	if s.RunnerDrain == nil {
		writeError(w, http.StatusServiceUnavailable, "runner drain unavailable")
		return
	}

	statuses := s.RunnerDrain.Drain(req.FleetID, req.RunnerIDs)
	if s.TaskNotify != nil {
		s.TaskNotify.Notify()
	}
	if s.Log != nil {
		drained, busy := drainStatusCounts(statuses)
		s.Log.Info("runner_drain",
			slog.String("fleet_id", req.FleetID),
			slog.Int("drained_count", drained),
			slog.Int("busy_count", busy))
	}
	writeJSON(w, http.StatusOK, api.DrainRunnersResponse{Runners: statuses})
}

func drainStatusCounts(statuses []api.DrainRunnerStatus) (drained int, busy int) {
	for _, status := range statuses {
		switch status.State {
		case api.DrainRunnerStateDrained:
			drained++
		case api.DrainRunnerStateBusy:
			busy++
		}
	}
	return drained, busy
}

func fleetToResponse(f *brokermodels.Fleet) *api.FleetResponse {
	if f == nil {
		return nil
	}
	return &api.FleetResponse{
		ID:          f.ID,
		Provisioner: f.Provisioner,
		Arch:        f.Arch,
		Size:        f.Size,
		CreatedAt:   f.CreatedAt.Unix(),
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
	fleetID := strings.TrimSpace(req.FleetID)
	if fleetID == "" {
		writeError(w, http.StatusBadRequest, "fleet_id required")
		return
	}

	ctx := r.Context()
	fleet, err := s.Store.GetFleet(ctx, fleetID)
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

	normalizedCmds := api.NormalizeCommands(req.Commands)
	script := strings.TrimSpace(req.Script)
	kind := api.EffectiveRunMode(&req.CreateTaskRequest)

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
		RunMode:       kind,
		WebhookURL:    strings.TrimSpace(req.WebhookURL),
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: mode,
		DockerImage:   req.DockerImage,
		Environment:   api.CloneEnvironment(req.Environment),
		Files:         api.NormalizeFiles(req.Files),
	}
	switch kind {
	case models.RunModeJavaScript, models.RunModePython, models.RunModeBash:
		task.Script = script
		task.SetupCommands = api.NormalizeCommandLines(req.SetupCommands)
		if len(bytes.TrimSpace(req.MessageChain)) > 0 {
			if !json.Valid(req.MessageChain) {
				writeError(w, http.StatusBadRequest, "message_chain must be valid JSON")
				return
			}
			task.MessageChainJSON = string(req.MessageChain)
		} else {
			task.MessageChainJSON = "{}"
		}
	case models.RunModeCommandList:
		task.Commands = normalizedCmds
	case models.RunModeArgv:
		task.Command = req.Command
	default:
		writeError(w, http.StatusBadRequest, "invalid run_mode")
		return
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
	s.recordTaskCreated(ctx, task.FleetID)
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

	runnerID := strings.TrimSpace(req.RunnerID)
	if s.RunnerDrain != nil && !s.RunnerDrain.TryStartClaim(runnerID) {
		writeJSON(w, http.StatusOK, api.ClaimTaskResponse{})
		return
	}
	task, err := s.Store.ClaimTask(r.Context(), req.RunnerID, req.FleetID, lease)
	if err != nil {
		if s.RunnerDrain != nil {
			s.RunnerDrain.FinishClaim(runnerID, "")
		}
		s.logErr("claim task", err)
		writeError(w, http.StatusInternalServerError, "could not claim task")
		return
	}
	if s.RunnerDrain != nil {
		taskID := ""
		if task != nil {
			taskID = task.ID
		}
		s.RunnerDrain.FinishClaim(runnerID, taskID)
	}
	if task != nil {
		s.recordTaskStartLatency(r.Context(), task)
		if s.Log != nil {
			s.Log.Info("task_claimed",
				slog.String("task_id", task.ID),
				slog.String("runner_id", task.RunnerID),
				slog.String("fleet_id", task.FleetID),
			)
		}
	}
	var payload *api.TaskPayload
	if task != nil {
		payload = api.TaskPayloadFrom(task)
	}
	writeJSON(w, http.StatusOK, api.ClaimTaskResponse{Task: payload})
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.Store.ListActiveTasks(r.Context())
	if err != nil {
		s.logErr("list active tasks", err)
		writeError(w, http.StatusInternalServerError, "could not list tasks")
		return
	}
	out := make([]api.TaskStatusResponse, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, taskStatusResponse(task, s))
	}
	writeJSON(w, http.StatusOK, api.ListTasksResponse{Tasks: out})
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
	if s.RunnerDrain != nil {
		s.RunnerDrain.CompleteTask(runnerID, id)
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
		s.recordTaskCompleted(r.Context(), task)
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

func (s *Server) completeTaskCore(ctx context.Context, taskID, runnerID string, req api.CompleteTaskRequest) (*taskstore.CompleteTaskResult, error) {
	resultJSON := ""
	if len(req.Result) > 0 {
		resultJSON = string(req.Result)
	}
	result, err := s.Store.CompleteTask(ctx, taskstore.CompleteTaskRequest{
		ID:           taskID,
		RunnerID:     runnerID,
		ExitCode:     req.ExitCode,
		ResultJSON:   resultJSON,
		ErrorMessage: req.Error,
		Canceled:     req.Canceled,
		FailureKind:  req.FailureKind,
	})
	if err != nil {
		return nil, err
	}
	task := result.Task
	if result.Outcome == taskstore.CompleteTaskOutcomeRequeued {
		s.recordTaskUnclaimed(ctx, task.FleetID)
		if s.TaskNotify != nil {
			s.TaskNotify.Notify()
		}
		if s.Log != nil {
			s.Log.Info("task_requeued_after_runner_infra_failure",
				slog.String("task_id", taskID),
				slog.String("runner_id", runnerID),
				slog.String("fleet_id", task.FleetID),
				slog.Int("infra_retry_count", task.InfraRetryCount),
			)
		}
		return result, nil
	}

	s.recordTaskCompleted(ctx, task)
	if s.Log != nil {
		outcome := "succeeded"
		if req.Canceled {
			outcome = "canceled"
		} else if req.ExitCode != 0 {
			outcome = "failed"
		}
		s.Log.Info("task_completed",
			slog.String("task_id", taskID),
			slog.String("runner_id", runnerID),
			slog.String("fleet_id", task.FleetID),
			slog.String("outcome", outcome),
			slog.Int("exit_code", req.ExitCode),
		)
	}
	go s.DeliverWebhook(task)
	return result, nil
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
	start := time.Now()
	err := s.Webhook.Deliver(ctx, task.WebhookURL, payload)
	s.recordWebhookDelivery(ctx, task.FleetID, err == nil, time.Since(start))
	if err != nil && s.Log != nil {
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
	kind := api.EffectiveRunMode(req)
	hasArgv := len(req.Command) > 0
	hasCmds := len(api.NormalizeCommands(req.Commands)) > 0
	hasSetup := len(api.NormalizeCommandLines(req.SetupCommands)) > 0
	script := strings.TrimSpace(req.Script)
	hasScript := script != ""
	hasChain := len(bytes.TrimSpace(req.MessageChain)) > 0

	switch kind {
	case models.RunModeCommandList:
		if !hasCmds {
			return "commands required for run_mode command_list"
		}
		if hasArgv || hasScript || hasChain || hasSetup {
			return "only commands allowed for run_mode command_list"
		}
	case models.RunModeArgv:
		if !hasArgv {
			return "command required for run_mode argv"
		}
		if hasCmds || hasScript || hasChain || hasSetup {
			return "only command allowed for run_mode argv"
		}
	case models.RunModeJavaScript:
		if !hasScript {
			return "script required for run_mode javascript_script"
		}
		if hasArgv || hasCmds {
			return "only script, setup_commands, and message_chain allowed for run_mode javascript_script"
		}
		if hasChain && !json.Valid(req.MessageChain) {
			return "message_chain must be valid JSON"
		}
	case models.RunModePython:
		if !hasScript {
			return "script required for run_mode python_script"
		}
		if hasArgv || hasCmds {
			return "only script, setup_commands, and message_chain allowed for run_mode python_script"
		}
		if hasChain && !json.Valid(req.MessageChain) {
			return "message_chain must be valid JSON"
		}
	case models.RunModeBash:
		if !hasScript {
			return "script required for run_mode bash_script"
		}
		if hasArgv || hasCmds {
			return "only script, setup_commands, and message_chain allowed for run_mode bash_script"
		}
		if hasChain && !json.Valid(req.MessageChain) {
			return "message_chain must be valid JSON"
		}
	default:
		if strings.TrimSpace(req.RunMode) != "" {
			return "invalid run_mode"
		}
		return "command, commands, or script required"
	}
	if strings.TrimSpace(req.RunMode) != "" && kind != models.RunMode(strings.ToLower(strings.TrimSpace(req.RunMode))) {
		return "run_mode does not match request body"
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
	if msg := api.ValidateFiles(req.Files); msg != "" {
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

func (s *Server) recordTaskCreated(ctx context.Context, fleetID string) {
	if s.Metrics == nil {
		return
	}
	s.Metrics.TaskCreated(ctx, fleetID)
}

func (s *Server) recordTaskCompleted(ctx context.Context, task *models.Task) {
	if s.Metrics == nil || task == nil {
		return
	}
	s.Metrics.TaskCompleted(ctx, task.FleetID, taskOutcome(task.Status))
}

func (s *Server) recordTaskStartLatency(ctx context.Context, task *models.Task) {
	if s.Metrics == nil || task == nil {
		return
	}
	s.Metrics.TaskStartLatency(ctx, task.FleetID, time.Since(task.CreatedAt))
}

func (s *Server) recordTaskUnclaimed(ctx context.Context, fleetID string) {
	if s.Metrics == nil {
		return
	}
	s.Metrics.TaskUnclaimed(ctx, fleetID)
}

func (s *Server) recordWebhookDelivery(ctx context.Context, fleetID string, succeeded bool, duration time.Duration) {
	if s.Metrics == nil {
		return
	}
	outcome := "failed"
	if succeeded {
		outcome = "succeeded"
	}
	s.Metrics.WebhookDelivered(ctx, fleetID, outcome, duration)
}

func (s *Server) recordRunnerConnectedSpinup(ctx context.Context, fleetID string, launchRequestedAt int64) {
	if s.Metrics == nil || launchRequestedAt <= 0 {
		return
	}
	s.Metrics.InstanceSpinupDuration(ctx, fleetID, "runner_connected", time.Since(time.Unix(launchRequestedAt, 0)))
}

func (s *Server) recordLeaseReaped(ctx context.Context, fleetID string) {
	if s.Metrics == nil {
		return
	}
	s.Metrics.LeaseReaped(ctx, fleetID)
}

// RecordLeaseReaped emits lease.reaps (used by the main lease-reap loop).
func (s *Server) RecordLeaseReaped(ctx context.Context, fleetID string) {
	s.recordLeaseReaped(ctx, fleetID)
}

// RecordTaskCompleted emits tasks.completed (used by the main lease-reap loop).
func (s *Server) RecordTaskCompleted(ctx context.Context, task *models.Task) {
	s.recordTaskCompleted(ctx, task)
}

func taskOutcome(status models.TaskStatus) string {
	switch status {
	case models.StatusSucceeded:
		return "succeeded"
	case models.StatusFailed:
		return "failed"
	case models.StatusCanceled:
		return "canceled"
	default:
		return string(status)
	}
}
