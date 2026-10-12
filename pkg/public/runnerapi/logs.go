package runnerapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerclientapi "github.com/superplanehq/superplane/pkg/runners/api"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
)

func (s *Server) uploadTaskLogChunk(w http.ResponseWriter, r *http.Request) {
	runner, ok := runnerFromContext(r.Context())
	if !ok {
		http.Error(w, "runner authentication is required", http.StatusUnauthorized)
		return
	}

	taskID, err := uuid.Parse(mux.Vars(r)["task_id"])
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}
	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil || sequence < 0 {
		http.Error(w, "invalid chunk sequence", http.StatusBadRequest)
		return
	}

	maxUploadBytes := runnerclientapi.DefaultLogUploadPolicy().MaxUploadBytes()
	content, err := io.ReadAll(io.LimitReader(r.Body, maxUploadBytes+1))
	if err != nil {
		http.Error(w, "failed to read log chunk", http.StatusBadRequest)
		return
	}
	if int64(len(content)) > maxUploadBytes {
		http.Error(w, "log chunk exceeds the upload limit", http.StatusRequestEntityTooLarge)
		return
	}

	store := runnerlogs.Current()
	if store == nil {
		http.Error(w, "log storage is unavailable", http.StatusServiceUnavailable)
		return
	}

	tx := database.DB(r.Context())
	task, err := models.FindRunnerTask(tx, taskID)
	if errors.Is(err, models.ErrRunnerTaskNotFound) ||
		(err == nil && (task.RunnerID == nil || *task.RunnerID != runner.ID)) {
		err = models.ErrTaskLogLifecycleNotFound
	}
	var lifecycle *models.RunnerTaskLogLifecycle
	if err == nil {
		lifecycle, err = task.FindLifecycle(tx)
	}
	if err == nil && lifecycle.ActiveStore != store.Name() {
		err = models.ErrTaskLogLifecycleNotFound
	}
	if err == nil && (task.IsTerminal() || lifecycle.State != models.RunnerTaskLogStateActive) {
		err = models.ErrTaskLogLifecycleClosed
	}
	var appendResult runnerlogs.AppendResult
	if err == nil {
		appendResult, err = store.Append(r.Context(), taskID, sequence, content)
	}
	switch {
	case err == nil:
		writeLogUploadResponse(w, appendResult.Truncated)
	case errors.Is(err, models.ErrTaskLogLifecycleNotFound):
		http.Error(w, "task log lifecycle not found", http.StatusNotFound)
	case errors.Is(err, runnerlogs.ErrSequenceConflict):
		http.Error(w, "log chunk sequence is not next", http.StatusConflict)
	case errors.Is(err, models.ErrTaskLogLifecycleClosed):
		http.Error(w, "task log lifecycle is closed", http.StatusConflict)
	default:
		http.Error(w, "failed to upload log chunk", http.StatusInternalServerError)
	}
}

func writeLogUploadResponse(w http.ResponseWriter, stop bool) {
	action := runnerclientapi.LogUploadActionContinue
	if stop {
		action = runnerclientapi.LogUploadActionStop
	}
	policy := runnerclientapi.DefaultLogUploadPolicy()
	w.Header().Set(runnerclientapi.HeaderLogUploadAction, action)
	w.Header().Set(
		runnerclientapi.HeaderLogTargetChunkBytes,
		strconv.FormatInt(policy.TargetChunkBytes, 10),
	)
	w.Header().Set(
		runnerclientapi.HeaderLogFlushMinimumMS,
		strconv.FormatInt(policy.PartialFlushMinimumMS, 10),
	)
	w.Header().Set(
		runnerclientapi.HeaderLogFlushMaximumMS,
		strconv.FormatInt(policy.PartialFlushMaximumMS, 10),
	)
	w.WriteHeader(http.StatusNoContent)
}
