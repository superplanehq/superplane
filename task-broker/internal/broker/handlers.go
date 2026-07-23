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

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/cwstream"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/webhook"
	brokermetrics "github.com/superplane/runner/task-broker/internal/metrics"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
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
	req.RunnerIDs = compactRunnerIDs(req.RunnerIDs)
	reason, ok := drainReason(req.Reason)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid drain reason")
		return
	}
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

	// Sticky-drain first so new claims are blocked, then reload persisted claims.
	// Loading before Drain raced with FinishClaim and let soft-drain treat a
	// real claim as a stale hub entry (false terminate under unhealthy).
	statuses := s.RunnerDrain.Drain(req.FleetID, req.RunnerIDs)
	claimedTaskIDs, err := s.Store.ClaimedTaskIDsByRunners(r.Context(), req.FleetID, req.RunnerIDs)
	if err != nil {
		s.logErr("claimed task ids by runners", err)
		writeError(w, http.StatusInternalServerError, "could not load claimed runner tasks")
		return
	}
	statuses = mergePersistedClaimedTasks(statuses, claimedTaskIDs)
	recoveredTasks := []api.RunnerTaskRecovery(nil)
	if reason == api.DrainReasonUnhealthy {
		var err error
		statuses, recoveredTasks, err = s.handleUnhealthyDrain(r.Context(), req.FleetID, req.RunnerIDs, statuses, claimedTaskIDs, req.TerminationConfirmed)
		if err != nil {
			s.logErr("handle unhealthy runner drain", err)
			writeError(w, http.StatusInternalServerError, "could not drain unhealthy runner tasks")
			return
		}
	}
	if s.TaskNotify != nil {
		s.TaskNotify.Notify()
	}
	if s.Log != nil {
		drained, busy := drainStatusCounts(statuses)
		s.Log.Info("runner_drain",
			slog.String("fleet_id", req.FleetID),
			slog.String("reason", string(reason)),
			slog.Int("drained_count", drained),
			slog.Int("busy_count", busy),
			slog.Int("recovered_task_count", len(recoveredTasks)))
	}
	writeJSON(w, http.StatusOK, api.DrainRunnersResponse{Runners: statuses, RecoveredTasks: recoveredTasks})
}

func drainReason(reason api.DrainReason) (api.DrainReason, bool) {
	switch reason {
	case "", api.DrainReasonScaleDown:
		return api.DrainReasonScaleDown, true
	case api.DrainReasonUnhealthy:
		return api.DrainReasonUnhealthy, true
	default:
		return "", false
	}
}

func mergePersistedClaimedTasks(statuses []api.DrainRunnerStatus, claimedTaskIDs map[string]string) []api.DrainRunnerStatus {
	if len(claimedTaskIDs) == 0 {
		return statuses
	}
	out := make([]api.DrainRunnerStatus, len(statuses))
	copy(out, statuses)
	for i := range out {
		taskID, ok := claimedTaskIDs[out[i].RunnerID]
		if !ok {
			continue
		}
		out[i].State = api.DrainRunnerStateBusy
		if out[i].ActiveTaskID == "" {
			out[i].ActiveTaskID = taskID
		}
	}
	return out
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

func (s *Server) handleUnhealthyDrain(ctx context.Context, fleetID string, runnerIDs []string, statuses []api.DrainRunnerStatus, claimedTaskIDs map[string]string, terminationConfirmed bool) ([]api.DrainRunnerStatus, []api.RunnerTaskRecovery, error) {
	busyRunnerIDs, activeTaskIDs := busyRunnerRecoveryInputs(statuses)
	if len(busyRunnerIDs) == 0 {
		return statuses, nil, nil
	}
	if terminationConfirmed {
		return s.finalizeTerminatedRunnerTasks(ctx, fleetID, runnerIDs, statuses, activeTaskIDs)
	}
	// Healthz failures under load are not proof the runner is dead. Keep claimed
	// work busy so fleet-manager defers TerminateInstances; sticky drain already
	// blocks new claims. Only clear stale hub "busy" with no persisted claim.
	return s.softDrainUnhealthyBusyRunners(statuses, claimedTaskIDs), nil, nil
}

func (s *Server) softDrainUnhealthyBusyRunners(statuses []api.DrainRunnerStatus, claimedTaskIDs map[string]string) []api.DrainRunnerStatus {
	out := make([]api.DrainRunnerStatus, len(statuses))
	copy(out, statuses)
	for i := range out {
		if out[i].State != api.DrainRunnerStateBusy {
			continue
		}
		runnerID := strings.TrimSpace(out[i].RunnerID)
		if _, hasClaim := claimedTaskIDs[runnerID]; hasClaim {
			continue
		}
		activeTaskID := strings.TrimSpace(out[i].ActiveTaskID)
		if activeTaskID == "" {
			// claimInProgress with no task id yet — keep busy so we do not
			// terminate mid-claim.
			continue
		}
		if s.RunnerDrain != nil {
			s.RunnerDrain.CompleteTask(runnerID, activeTaskID)
		}
		out[i].State = api.DrainRunnerStateDrained
		out[i].ActiveTaskID = ""
	}
	return out
}

func (s *Server) finalizeTerminatedRunnerTasks(ctx context.Context, fleetID string, runnerIDs []string, statuses []api.DrainRunnerStatus, activeTaskIDs map[string]string) ([]api.DrainRunnerStatus, []api.RunnerTaskRecovery, error) {
	// Freeze remaining claimed tasks before finalizing so the lease reaper cannot
	// requeue them while we confirm EC2 termination. Authentic CompleteTask may
	// still win this race and is accepted by the store.
	if _, err := s.Store.MarkLostRunnerTasksTerminating(ctx, fleetID, runnerIDs); err != nil {
		return nil, nil, err
	}
	recoveries, err := s.Store.FinalizeTerminatedRunnerTasks(ctx, fleetID, runnerIDs)
	if err != nil {
		return nil, nil, err
	}

	out := make([]api.RunnerTaskRecovery, 0, len(recoveries))
	recoveredRunnerIDs := make(map[string]struct{}, len(recoveries))
	for _, recovery := range recoveries {
		if s.RunnerDrain != nil {
			s.RunnerDrain.CompleteTask(recovery.RunnerID, recovery.ID)
		}
		recoveredRunnerIDs[recovery.RunnerID] = struct{}{}
		state := recoveryState(recovery.Status)
		out = append(out, api.RunnerTaskRecovery{
			RunnerID: recovery.RunnerID,
			TaskID:   recovery.ID,
			State:    state,
		})
		s.afterLostRunnerRecovery(ctx, recovery, state)
	}
	if s.RunnerDrain != nil {
		for runnerID, taskID := range activeTaskIDs {
			if _, recovered := recoveredRunnerIDs[runnerID]; recovered {
				continue
			}
			s.RunnerDrain.CompleteTask(runnerID, taskID)
		}
	}

	statuses = markTerminatingBusyRunnersDrained(statuses, activeTaskIDs, recoveredRunnerIDs)
	if s.Log != nil {
		s.Log.Info("lost_runner_tasks_finalized_after_termination",
			slog.String("fleet_id", fleetID),
			slog.Int("task_count", len(out)),
			slog.Any("tasks", out))
	}
	return statuses, out, nil
}

func busyRunnerRecoveryInputs(statuses []api.DrainRunnerStatus) ([]string, map[string]string) {
	runnerIDs := make([]string, 0)
	activeTaskIDs := make(map[string]string)
	for _, status := range statuses {
		if status.State != api.DrainRunnerStateBusy {
			continue
		}
		runnerID := strings.TrimSpace(status.RunnerID)
		if runnerID == "" {
			continue
		}
		runnerIDs = append(runnerIDs, runnerID)
		if taskID := strings.TrimSpace(status.ActiveTaskID); taskID != "" {
			activeTaskIDs[runnerID] = taskID
		}
	}
	return runnerIDs, activeTaskIDs
}

func markTerminatingBusyRunnersDrained(statuses []api.DrainRunnerStatus, activeTaskIDs map[string]string, recoveredRunnerIDs map[string]struct{}) []api.DrainRunnerStatus {
	out := make([]api.DrainRunnerStatus, len(statuses))
	copy(out, statuses)
	for i := range out {
		if out[i].State != api.DrainRunnerStateBusy {
			continue
		}
		_, recovered := recoveredRunnerIDs[out[i].RunnerID]
		activeTaskID := strings.TrimSpace(activeTaskIDs[out[i].RunnerID])
		if !recovered && activeTaskID == "" {
			continue
		}
		out[i].State = api.DrainRunnerStateDrained
		out[i].ActiveTaskID = ""
	}
	return out
}

func recoveryState(status models.TaskStatus) api.RunnerTaskRecoveryState {
	switch status {
	case models.StatusCanceled:
		return api.RunnerTaskRecoveryStateCanceled
	default:
		return api.RunnerTaskRecoveryStateFailed
	}
}

func (s *Server) afterLostRunnerRecovery(ctx context.Context, recovery taskstore.LostRunnerTaskRecovery, state api.RunnerTaskRecoveryState) {
	task, err := s.Store.GetTask(ctx, recovery.ID)
	if err != nil {
		s.logErr("get recovered lost runner task", err)
		return
	}
	s.recordTaskCompleted(ctx, task)
	go s.DeliverWebhook(task)
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
		writeError(w, http.StatusBadRequest, err.Error())
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
	if req.WebhookPayloadSizeLimit > 0 {
		task.WebhookPayloadSizeLimit = req.WebhookPayloadSizeLimit
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
		if strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "wrong runner") ||
			strings.Contains(err.Error(), "not claimed") ||
			strings.Contains(err.Error(), "termination pending") {
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
	if result.Outcome == taskstore.CompleteTaskOutcomeAlreadyTerminal {
		if s.Log != nil {
			s.Log.Info("task_complete_duplicate",
				slog.String("task_id", taskID),
				slog.String("runner_id", runnerID),
				slog.String("fleet_id", task.FleetID),
				slog.String("status", string(task.Status)),
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
