package broker

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/task-broker/internal/livelogs"
)

// getBrokerTaskLiveLogs streams task stdout/stderr from CloudWatch using credentials on the broker host.
// Caller must send Authorization: Bearer <AUTH_TOKEN>. Response is application/x-ndjson (same contract as SuperPlane UI).
func (s *Server) getBrokerTaskLiveLogs(w http.ResponseWriter, r *http.Request) {
	brokerID := strings.TrimSpace(chi.URLParam(r, "id"))
	if brokerID == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}

	row, err := s.Store.GetBrokerTask(r.Context(), brokerID)
	if err != nil {
		s.logErr("get broker task for live logs", err)
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if row == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if row.FleetTaskID == "" {
		writeError(w, http.StatusNotFound, "Logs are not available for this execution yet. Check again shortly.")
		return
	}

	fleet, err := s.Store.GetFleet(r.Context(), row.FleetID)
	if err != nil {
		s.logErr("get fleet for live logs", err)
		writeError(w, http.StatusInternalServerError, "could not load fleet")
		return
	}
	if fleet == nil {
		writeError(w, http.StatusInternalServerError, "fleet missing")
		return
	}

	st, upstream := s.forwardGetTask(r.Context(), fleet, row.FleetTaskID)
	if st == http.StatusNotFound {
		writeError(w, http.StatusBadGateway, "upstream task missing")
		return
	}
	if st != http.StatusOK {
		s.warn("upstream get task failed for live logs", slog.Int("status", st), slog.String("fleet", fleet.ID))
		writeError(w, http.StatusBadGateway, "fleet-manager rejected status request")
		return
	}

	_, taskLog, err := parseUpstreamTaskLog(upstream)
	if err != nil {
		writeError(w, http.StatusBadGateway, "invalid upstream response")
		return
	}
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
	_ = livelogs.StreamCloudWatchLogToNDJSON(r.Context(), w, flusher, g, stName, region)
}
