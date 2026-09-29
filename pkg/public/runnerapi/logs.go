package runnerapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"gorm.io/gorm"
)

const maxLogChunkSize = 4 << 20

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

	content, err := io.ReadAll(io.LimitReader(r.Body, maxLogChunkSize+1))
	if err != nil {
		http.Error(w, "failed to read log chunk", http.StatusBadRequest)
		return
	}
	if len(content) > maxLogChunkSize {
		http.Error(w, "log chunk exceeds 4 MiB", http.StatusRequestEntityTooLarge)
		return
	}

	provider := blob.Current()
	if provider == nil {
		http.Error(w, "log storage is unavailable", http.StatusServiceUnavailable)
		return
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		upload, task, err := runner.FindTaskLogUpload(tx, taskID)
		if err != nil {
			return err
		}
		if task.IsTerminal() {
			return models.ErrTaskLogUploadClosed
		}
		if sequence < upload.NextChunkSequence {
			return nil
		}
		if sequence > upload.NextChunkSequence {
			return models.ErrTaskLogChunkSequenceConflict
		}
		if upload.FinalizingAt != nil {
			return models.ErrTaskLogUploadFinalizing
		}

		key := runnerlogs.ChunkKey(task.OrganizationID, task.ID, sequence)
		if err := provider.Put(r.Context(), key, bytes.NewReader(content), blob.PutOptions{
			ContentType: "application/x-ndjson",
		}); err != nil {
			return fmt.Errorf("store log chunk: %w", err)
		}
		return upload.Advance(tx, sequence, int64(len(content)), time.Now())
	})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, models.ErrTaskLogUploadNotFound):
		http.Error(w, "task log upload not found", http.StatusNotFound)
	case errors.Is(err, models.ErrTaskLogChunkSequenceConflict):
		http.Error(w, "log chunk sequence is not next", http.StatusConflict)
	case errors.Is(err, models.ErrTaskLogUploadFinalizing):
		http.Error(w, "task log upload is finalizing", http.StatusConflict)
	case errors.Is(err, models.ErrTaskLogUploadClosed):
		http.Error(w, "task log upload is closed", http.StatusConflict)
	default:
		http.Error(w, "failed to upload log chunk", http.StatusInternalServerError)
	}
}
