package broker

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/superplanehq/superplane/pkg/taskbroker/livelogs"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
)

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

	status := taskStatusResponse(task, s)
	taskLog := status.TaskLog
	if taskLog == nil || strings.TrimSpace(taskLog.Type) == "" {
		writeError(w, http.StatusNotFound, "Logs are not available for this execution yet. Check again shortly.")
		return
	}
	if taskLog.Type != api.TaskLogTypeCloudWatch || taskLog.CloudWatch == nil {
		writeError(w, http.StatusNotFound, "Live logs are not configured for this execution")
		return
	}

	g := strings.TrimSpace(taskLog.CloudWatch.LogGroupName)
	stName := strings.TrimSpace(taskLog.CloudWatch.LogStreamName)
	if g == "" || stName == "" {
		writeError(w, http.StatusNotFound, "Log details are incomplete for this execution")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming unsupported")
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
}
