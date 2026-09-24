package broker

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/task-broker/internal/livelogs"
)

const maxLiveLogEventBytes = 1 << 20

func (s *Server) liveLogs() *livelogs.Hub {
	if s.LiveLogs == nil {
		s.LiveLogs = livelogs.NewHub()
	}
	return s.LiveLogs
}

func (s *Server) closeLiveLogs(taskID string) {
	s.liveLogs().Close(taskID)
}

func (s *Server) appendTaskLiveLogs(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	identity, ok := runnerIdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	task, err := s.Store.GetTask(r.Context(), id)
	if err != nil {
		s.logErr("get task for live log ingest", err)
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if strings.TrimSpace(task.RunnerID) != identity.RunnerID {
		writeError(w, http.StatusForbidden, "runner identity mismatch")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxLiveLogEventBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body) > maxLiveLogEventBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "payload too large")
		return
	}
	s.liveLogs().Append(id, body)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getTaskLiveLogs(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}

	task, err := s.Store.GetTask(r.Context(), id)
	if err != nil {
		s.logErr("get task for live logs", err)
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	status := taskStatusResponse(task, s)
	taskLog := status.TaskLog
	if taskLog != nil && strings.TrimSpace(taskLog.Type) == api.TaskLogTypeCloudWatch && taskLog.CloudWatch != nil {
		g := strings.TrimSpace(taskLog.CloudWatch.LogGroupName)
		stName := strings.TrimSpace(taskLog.CloudWatch.LogStreamName)
		if g == "" || stName == "" {
			writeError(w, http.StatusNotFound, "Log details are incomplete for this execution")
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		region := strings.TrimSpace(taskLog.CloudWatch.Region)
		if err := livelogs.StreamCloudWatchLogToNDJSON(r.Context(), w, flusher, g, stName, region); err != nil && s.Log != nil {
			s.Log.Warn("live logs stream", slog.Any("err", err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	if err := streamLocalLiveLogs(r.Context(), w, flusher, s.liveLogs(), id); err != nil && s.Log != nil {
		s.Log.Warn("live logs stream", slog.Any("err", err))
	}
}

func streamLocalLiveLogs(ctx context.Context, w io.Writer, flusher http.Flusher, hub *livelogs.Hub, taskID string) error {
	from := 0
	for {
		records, next, closed, err := hub.Wait(ctx, taskID, from)
		if err != nil {
			return err
		}
		for _, rec := range records {
			if _, err := w.Write(append(append([]byte{}, rec...), '\n')); err != nil {
				return err
			}
		}
		if len(records) > 0 && flusher != nil {
			flusher.Flush()
		}
		from = next
		if closed {
			return nil
		}
	}
}
