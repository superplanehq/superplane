package fleetmanager

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/superplane/runner/fleet-manager/internal/ec2provision"
	"github.com/superplane/runner/fleet-manager/internal/store"
	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/cwstream"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/webhook"
)

// Server exposes fleet-manager HTTP handlers.
type Server struct {
	Store   store.Store
	Webhook *webhook.Sender
	Log     *slog.Logger

	// TaskNotify wakes WebSocket runners when a new task is enqueued; nil disables notifications.
	TaskNotify *WaitHub

	// RunnerCancel maps active WebSocket runner connections for immediate cancel push; nil disables push.
	RunnerCancel *RunnerCancelHub

	// TaskCloudWatchLogGroup when set is returned on GET /v1/tasks/{id} and completion webhooks so
	// clients can open the matching stream in AWS. Runners must set RUNNER_CLOUDWATCH_LOG_GROUP (and matching prefix).
	TaskCloudWatchLogGroup        string
	TaskCloudWatchLogStreamPrefix string
	// TaskCloudWatchRegion is optional; included in task_log.cloudwatch.region for API clients.
	TaskCloudWatchRegion string

	// EC2Launcher when EC2 hot pool is enabled; used for optional /v1/admin diagnostics.
	EC2Launcher *ec2provision.Launcher

	// When EC2 provisioning is enabled with terminate-after-task, fleet-manager calls this after each
	// successful complete (runner_id must be the EC2 instance id from IMDS in user-data).
	TerminateRunnerAfterTaskEnabled bool
	TerminateRunnerInstance         func(ctx context.Context, instanceID string) error
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) taskLogForTask(taskID string) *api.TaskLogSink {
	if g := strings.TrimSpace(s.TaskCloudWatchLogGroup); g != "" {
		stream := cwstream.TaskLogStream(s.TaskCloudWatchLogStreamPrefix, taskID)
		return api.TaskLogSinkCloudWatchFromParts(g, stream, s.TaskCloudWatchRegion)
	}
	return nil
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
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

	switch {
	case hasArgv && hasShell:
		writeError(w, http.StatusBadRequest, "specify either command or commands, not both")
		return
	case !hasArgv && !hasShell:
		writeError(w, http.StatusBadRequest, "command or commands required")
		return
	}
	if strings.TrimSpace(req.WebhookURL) == "" {
		writeError(w, http.StatusBadRequest, "webhook_url required")
		return
	}
	if msg := api.ValidateExecutionTimeoutSeconds(req.ExecutionTimeoutSeconds); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	mode := models.ExecutionHost
	switch strings.ToLower(strings.TrimSpace(req.ExecutionMode)) {
	case "", string(models.ExecutionHost):
		mode = models.ExecutionHost
	case string(models.ExecutionDocker):
		mode = models.ExecutionDocker
		if strings.TrimSpace(req.DockerImage) == "" {
			writeError(w, http.StatusBadRequest, "docker_image required for docker execution_mode")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "invalid execution_mode")
		return
	}

	task := &models.Task{
		ID:            uuid.NewString(),
		WebhookURL:    req.WebhookURL,
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: mode,
		DockerImage:   req.DockerImage,
	}
	if hasShell {
		task.Commands = normalizedCmds
		task.Command = nil
	} else {
		task.Command = req.Command
		task.Commands = nil
	}
	if req.ExecutionTimeoutSeconds != nil {
		v := *req.ExecutionTimeoutSeconds
		task.ExecutionTimeoutSeconds = &v
	}
	if err := s.Store.CreateTask(r.Context(), task); err != nil {
		s.Log.Error("create task", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	if s.TaskNotify != nil {
		s.TaskNotify.Notify()
	}
	writeJSON(w, http.StatusCreated, api.CreateTaskResponse{ID: task.ID})
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
	lease := time.Duration(req.LeaseSeconds) * time.Second
	if lease <= 0 {
		lease = 5 * time.Minute
	}

	task, err := s.Store.ClaimTask(r.Context(), req.RunnerID, lease)
	if err != nil {
		s.Log.Error("claim task", slog.Any("err", err))
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
		s.Log.Error("get task", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "could not load task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	resp := api.TaskStatusResponse{
		ID:              task.ID,
		Status:          string(task.Status),
		Output:          task.Output,
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
	writeJSON(w, http.StatusOK, resp)
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
		s.Log.Error("complete task", slog.Any("err", err))
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
		s.Log.Error("cancel task", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "could not cancel task")
		return
	}
	switch outcome {
	case store.CancelOutcomeNotFound:
		writeError(w, http.StatusNotFound, "task not found")
		return
	case store.CancelOutcomeCanceledQueued:
		go s.DeliverWebhook(task)
	case store.CancelOutcomeCancelRequested:
		rid := strings.TrimSpace(task.RunnerID)
		if s.RunnerCancel != nil && rid != "" {
			if !s.RunnerCancel.PushCancel(rid, task.ID) && s.Log != nil {
				s.Log.Debug("runner cancel ws push not delivered",
					slog.String("runner_id", rid), slog.String("task_id", task.ID))
			}
		}
	}

	state := string(outcome)
	if task == nil {
		writeError(w, http.StatusInternalServerError, "cancel task missing row")
		return
	}
	writeJSON(w, http.StatusOK, api.CancelTaskResponse{
		ID:     task.ID,
		State:  state,
		Status: string(task.Status),
	})
}

// completeTaskCore runs Store.CompleteTask, delivers the webhook, and schedules optional EC2 termination.
// On conflict (wrong runner / bad state), err message contains "not found" or "wrong runner" for HTTP 409 mapping.
func (s *Server) completeTaskCore(ctx context.Context, taskID, runnerID string, req api.CompleteTaskRequest) (*models.Task, error) {
	resultJSON := ""
	if len(req.Result) > 0 {
		resultJSON = string(req.Result)
	}
	task, err := s.Store.CompleteTask(ctx, taskID, runnerID, req.ExitCode, req.Output, resultJSON, req.Error, req.Canceled)
	if err != nil {
		return nil, err
	}

	go s.DeliverWebhook(task)

	if !req.Canceled && s.TerminateRunnerAfterTaskEnabled && s.TerminateRunnerInstance != nil && isEC2InstanceID(runnerID) {
		go func(instanceID string) {
			tctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			if err := s.TerminateRunnerInstance(tctx, instanceID); err != nil && s.Log != nil {
				s.Log.Warn("terminate runner instance after task failed",
					slog.String("instance_id", instanceID), slog.Any("err", err))
			} else if s.Log != nil {
				s.Log.Info("terminate runner instance after task (ec2 shutdown)", slog.String("instance_id", instanceID))
			}
		}(runnerID)
	}

	return task, nil
}

// DeliverWebhook POSTs terminal task state to the task webhook URL.
func (s *Server) DeliverWebhook(task *models.Task) {
	if s.Webhook == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st := string(task.Status)
	exit := 0
	if task.ExitCode != nil {
		exit = *task.ExitCode
	}
	payload := api.WebhookPayload{
		TaskID:   task.ID,
		Status:   st,
		ExitCode: exit,
		Output:   task.Output,
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
	if err := s.Webhook.Deliver(ctx, task.WebhookURL, payload); err != nil {
		if s.Log != nil {
			s.Log.Warn("webhook delivery failed", slog.String("task_id", task.ID), slog.Any("err", err))
		}
	}
}

func (s *Server) adminManagedRunners(w http.ResponseWriter, r *http.Request) {
	if s.EC2Launcher == nil {
		writeError(w, http.StatusServiceUnavailable, "ec2 diagnostics unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	rows, err := s.EC2Launcher.ListManagedRunners(ctx)
	if err != nil {
		if s.Log != nil {
			s.Log.Warn("admin managed runners", slog.Any("err", err))
		}
		writeError(w, http.StatusBadGateway, "could not list instances")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) adminEc2Console(w http.ResponseWriter, r *http.Request) {
	if s.EC2Launcher == nil {
		writeError(w, http.StatusServiceUnavailable, "ec2 diagnostics unavailable")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("instance_id"))
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	out, err := s.EC2Launcher.ManagedInstanceConsoleOutput(ctx, id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}
