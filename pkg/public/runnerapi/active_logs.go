package runnerapi

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
)

func (s *Server) readActiveTaskLogs(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(mux.Vars(r)["task_id"])
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}
	cursor := r.URL.Query().Get("after")
	if err := runnerlogs.VerifyActiveLogRead(
		s.signer.Secret,
		r.Header.Get("Authorization"),
		taskID,
		cursor,
		time.Now(),
	); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	store := runnerlogs.Current()
	if store == nil || store.Name() != s.activeLogStoreName {
		http.Error(w, "Active log storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	result, err := store.ReadAfter(r.Context(), taskID, cursor)
	if errors.Is(err, runnerlogs.ErrNotFound) {
		http.Error(w, "active runner log not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, runnerlogs.ErrInvalidCursor) {
		http.Error(w, "Invalid log cursor", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Could not read task logs", http.StatusInternalServerError)
		return
	}
	defer result.Content.Close()

	content, err := io.ReadAll(result.Content)
	if err != nil {
		http.Error(w, "Could not read task logs", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set(runnerlogs.HeaderCursor, result.Cursor)
	w.Header().Set("Content-Type", "application/x-ndjson")
	if len(content) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	_, _ = w.Write(content)
}
