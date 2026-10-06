package public

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/blob"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
)

func (s *Server) handleRunnerTaskLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	allowed, err := s.authService.CheckOrganizationPermission(
		r.Context(),
		user.ID.String(),
		user.OrganizationID.String(),
		"canvases",
		"read",
	)
	if err != nil {
		middleware.SetServerError(r.Context(), err, nil)
		http.Error(w, "Authorization check failed", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	canvasID, err := uuid.Parse(strings.TrimSpace(vars["canvas_id"]))
	if err != nil {
		http.Error(w, "Invalid canvas id", http.StatusBadRequest)
		return
	}
	executionID, err := uuid.Parse(strings.TrimSpace(vars["execution_id"]))
	if err != nil {
		http.Error(w, "Invalid execution id", http.StatusBadRequest)
		return
	}
	access, err := runneraction.ResolveLiveLogAccess(user.OrganizationID, canvasID, executionID)
	if err != nil {
		writeRunnerLiveLogSessionError(w, r, err)
		return
	}
	if access.TaskBackend != core.RunnerTaskBackendIntegrated {
		http.Error(w, "Logs use the legacy runner backend", http.StatusNotFound)
		return
	}
	taskID, err := uuid.Parse(access.BrokerTaskID)
	if err != nil {
		http.Error(w, "Logs are not available for this execution", http.StatusNotFound)
		return
	}
	task, err := models.FindRunnerTask(database.Conn(), taskID)
	if err != nil || task.OrganizationID != user.OrganizationID {
		http.Error(w, "Logs are not available for this execution", http.StatusNotFound)
		return
	}

	s.serveRunnerTaskLogs(w, r, task, r.URL.Query().Get("after"))
}

func (s *Server) serveRunnerTaskLogs(
	w http.ResponseWriter,
	r *http.Request,
	task *models.RunnerTask,
	cursor string,
) {
	lifecycle, err := task.FindLifecycle(database.DB(r.Context()))
	if errors.Is(err, models.ErrTaskLogLifecycleNotFound) {
		if cursor != "" {
			writeRunnerLogReset(w)
			return
		}
		s.serveFinalRunnerTaskLog(w, r, task, nil)
		return
	}
	if err != nil {
		writeCouldNotReadTaskLogs(w, r, err)
		return
	}

	switch lifecycle.State {
	case models.RunnerTaskLogStateArchived:
		if cursor == "" {
			s.serveFinalRunnerTaskLog(w, r, task, lifecycle)
			return
		}
		if lifecycle.FinalCursor != nil && cursor == *lifecycle.FinalCursor {
			writeRunnerLogState(w, models.RunnerTaskLogStateArchived, cursor)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	store := runnerlogs.Current()
	if store == nil || store.Name() != lifecycle.ActiveStore {
		http.Error(w, "Active log storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	result, err := store.ReadAfter(r.Context(), task.ID, cursor)
	if errors.Is(err, runnerlogs.ErrInvalidCursor) {
		if lifecycle.State == models.RunnerTaskLogStateArchived {
			writeRunnerLogReset(w)
			return
		}
		http.Error(w, "Invalid log cursor", http.StatusBadRequest)
		return
	}
	if errors.Is(err, runnerlogs.ErrNotFound) {
		if lifecycle.State == models.RunnerTaskLogStateActive ||
			lifecycle.State == models.RunnerTaskLogStateArchivable ||
			lifecycle.State == models.RunnerTaskLogStateArchiving {
			writeRunnerLogState(w, lifecycle.State, "0")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeRunnerLogReset(w)
		return
	}
	if err != nil {
		writeCouldNotReadTaskLogs(w, r, err)
		return
	}
	defer result.Content.Close()

	state := lifecycle.State
	if state == models.RunnerTaskLogStateArchived && lifecycle.FinalCursor != nil &&
		result.Cursor != *lifecycle.FinalCursor {
		state = models.RunnerTaskLogStateArchiving
	}
	writeRunnerLogState(w, state, result.Cursor)
	content, err := io.ReadAll(result.Content)
	if err != nil {
		writeCouldNotReadTaskLogs(w, r, err)
		return
	}
	if len(content) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	_, _ = w.Write(content)
}

func writeCouldNotReadTaskLogs(w http.ResponseWriter, r *http.Request, err error) {
	middleware.SetServerError(r.Context(), err, nil)
	http.Error(w, "Could not read task logs", http.StatusInternalServerError)
}

func writeRunnerLogState(w http.ResponseWriter, state, cursor string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set(runnerlogs.HeaderState, state)
	w.Header().Set(runnerlogs.HeaderCursor, cursor)
}

func writeRunnerLogReset(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set(runnerlogs.HeaderState, models.RunnerTaskLogStateArchived)
	w.Header().Set(runnerlogs.HeaderReset, "true")
	w.WriteHeader(http.StatusConflict)
}

func (s *Server) serveFinalRunnerTaskLog(
	w http.ResponseWriter,
	r *http.Request,
	task *models.RunnerTask,
	lifecycle *models.RunnerTaskLogLifecycle,
) {
	provider := blob.Current()
	key := ""
	cursor := ""
	if lifecycle != nil {
		if lifecycle.FinalObjectKey != nil {
			key = *lifecycle.FinalObjectKey
		}
		if lifecycle.FinalCursor != nil {
			cursor = *lifecycle.FinalCursor
		}
	}
	if key == "" {
		installationID, err := models.GetInstallationID(database.DB(r.Context()))
		if err != nil {
			writeCouldNotReadTaskLogs(w, r, err)
			return
		}
		key = runnerlogs.FinalKey(installationID, task.OrganizationID, task.ID)
	}
	info, err := provider.Head(r.Context(), key)
	if errors.Is(err, blob.ErrNotFound) {
		http.Error(w, "Task logs not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeCouldNotReadTaskLogs(w, r, err)
		return
	}

	reader, err := provider.Get(r.Context(), key)
	if err != nil {
		writeCouldNotReadTaskLogs(w, r, err)
		return
	}
	defer reader.Close()
	writeRunnerLogState(w, models.RunnerTaskLogStateArchived, cursor)
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	_, _ = io.Copy(w, reader)
}
