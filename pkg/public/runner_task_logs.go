package public

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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

const (
	runnerLogLongPollDuration = 25 * time.Second
	runnerLogPollInterval     = 500 * time.Millisecond
	runnerLogSignedURLTTL     = 5 * time.Minute
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
		writeRunnerLiveLogSessionError(w, err)
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

	afterChunk := int64(-1)
	if value := r.URL.Query().Get("after_chunk"); value != "" {
		afterChunk, err = strconv.ParseInt(value, 10, 64)
		if err != nil || afterChunk < -1 {
			http.Error(w, "Invalid after_chunk", http.StatusBadRequest)
			return
		}
	}
	s.serveRunnerTaskLogs(w, r, task, afterChunk)
}

func (s *Server) serveRunnerTaskLogs(
	w http.ResponseWriter,
	r *http.Request,
	task *models.RunnerTask,
	afterChunk int64,
) {
	deadline := time.NewTimer(runnerLogLongPollDuration)
	defer deadline.Stop()
	ticker := time.NewTicker(runnerLogPollInterval)
	defer ticker.Stop()

	for {
		upload, err := models.FindTaskLogUpload(database.Conn(), task.ID)
		switch {
		case err == nil && upload.FinalizingAt != nil:
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusAccepted)
			return
		case err == nil && upload.NextChunkSequence > afterChunk+1:
			s.writeRunnerLogChunks(w, r, task, afterChunk+1, upload.NextChunkSequence)
			return
		case err != nil && !errors.Is(err, models.ErrTaskLogUploadNotFound):
			http.Error(w, "Could not read task logs", http.StatusInternalServerError)
			return
		case errors.Is(err, models.ErrTaskLogUploadNotFound):
			s.serveFinalRunnerTaskLog(w, r, task)
			return
		}

		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			w.WriteHeader(http.StatusNoContent)
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) writeRunnerLogChunks(
	w http.ResponseWriter,
	r *http.Request,
	task *models.RunnerTask,
	first, next int64,
) {
	provider := blob.Current()
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Last-Chunk", strconv.FormatInt(next-1, 10))
	for sequence := first; sequence < next; sequence++ {
		reader, err := provider.Get(
			r.Context(),
			runnerlogs.ChunkKey(task.OrganizationID, task.ID, sequence),
		)
		if err != nil {
			http.Error(w, "Could not read task logs", http.StatusInternalServerError)
			return
		}
		_, copyErr := io.Copy(w, reader)
		closeErr := reader.Close()
		if copyErr != nil || closeErr != nil {
			return
		}
	}
}

func (s *Server) serveFinalRunnerTaskLog(
	w http.ResponseWriter,
	r *http.Request,
	task *models.RunnerTask,
) {
	provider := blob.Current()
	key := runnerlogs.FinalKey(task.OrganizationID, task.ID)
	info, err := provider.Head(r.Context(), key)
	if errors.Is(err, blob.ErrNotFound) {
		http.Error(w, "Task logs not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Could not read task logs", http.StatusInternalServerError)
		return
	}

	url, err := provider.SignedGetURL(r.Context(), key, runnerLogSignedURLTTL)
	if err == nil {
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return
	}
	if !errors.Is(err, blob.ErrSignedURLUnsupported) {
		http.Error(w, "Could not create task log URL", http.StatusInternalServerError)
		return
	}

	reader, err := provider.Get(r.Context(), key)
	if err != nil {
		http.Error(w, "Could not read task logs", http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	_, _ = io.Copy(w, reader)
}
