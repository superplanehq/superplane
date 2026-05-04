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

	"github.com/superplane/runner/fleet-manager/internal/store"
	"github.com/superplane/runner/fleet-manager/internal/webhook"
	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

// Server exposes fleet-manager HTTP handlers.
type Server struct {
	Store   store.Store
	Webhook *webhook.Sender
	Log     *slog.Logger
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
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
	if err := s.Store.CreateTask(r.Context(), task); err != nil {
		s.Log.Error("create task", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
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

	task, err := s.Store.CompleteTask(r.Context(), id, runnerID, req.ExitCode, req.Output, req.Error)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "wrong runner") {
			writeError(w, http.StatusConflict, "cannot complete task")
			return
		}
		s.Log.Error("complete task", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "could not complete task")
		return
	}

	go s.deliverWebhook(task)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deliverWebhook(task *models.Task) {
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
	if err := s.Webhook.Deliver(ctx, task.WebhookURL, payload); err != nil {
		if s.Log != nil {
			s.Log.Warn("webhook delivery failed", slog.String("task_id", task.ID), slog.Any("err", err))
		}
	}
}
