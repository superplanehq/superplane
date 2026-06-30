package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

const defaultRunnerLiveLogRequestTimeout = 30 * time.Second

type runnerLiveLogsResponse struct {
	CanvasID     string                       `json:"canvas_id"`
	ExecutionID  string                       `json:"execution_id"`
	BrokerTaskID string                       `json:"broker_task_id"`
	Count        int                          `json:"count"`
	Truncated    bool                         `json:"truncated,omitempty"`
	Records      []runneraction.LiveLogRecord `json:"records"`
}

func (s *Server) handleRunnerLiveLogSession(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	allowed, err := s.authService.CheckOrganizationPermission(r.Context(),
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

	session, err := runneraction.NewLiveLogSession(access.BrokerTaskID, time.Now())
	if err != nil {
		if strings.Contains(err.Error(), "is not set") {
			http.Error(w, "Live logs are not configured on this installation", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "Could not create live log session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(session); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleRunnerLiveLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	allowed, err := s.authService.CheckOrganizationPermission(r.Context(),
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

	limit, err := parseRunnerLiveLogLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultRunnerLiveLogRequestTimeout)
	defer cancel()
	result, err := runneraction.FetchLiveLogRecords(ctx, access.BrokerTaskID, runneraction.LiveLogFetchOptions{Limit: limit})
	if err != nil {
		if strings.Contains(err.Error(), "is not set") {
			http.Error(w, "Live logs are not configured on this installation", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "Could not fetch live logs", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(runnerLiveLogsResponse{
		CanvasID:     canvasID.String(),
		ExecutionID:  executionID.String(),
		BrokerTaskID: access.BrokerTaskID,
		Count:        len(result.Records),
		Truncated:    result.Truncated,
		Records:      result.Records,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func parseRunnerLiveLogLimit(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return runneraction.DefaultLiveLogRecordLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 0, errors.New("Invalid limit")
	}
	return limit, nil
}

func writeRunnerLiveLogSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, runneraction.ErrLiveLogCanvasNotFound):
		http.Error(w, "Canvas not found", http.StatusNotFound)
	case errors.Is(err, runneraction.ErrLiveLogExecutionNotFound):
		http.Error(w, "Execution not found", http.StatusNotFound)
	case errors.Is(err, runneraction.ErrLiveLogNodeNotFound):
		http.Error(w, "Node not found", http.StatusNotFound)
	case errors.Is(err, runneraction.ErrLiveLogNotRunner):
		http.Error(w, "Live logs are only available for Runner components", http.StatusBadRequest)
	case errors.Is(err, runneraction.ErrLiveLogBrokerTaskMissing):
		http.Error(
			w,
			"Logs are not available for this execution yet. Check again shortly.",
			http.StatusNotFound,
		)
	default:
		http.Error(w, "Lookup failed", http.StatusInternalServerError)
	}
}
