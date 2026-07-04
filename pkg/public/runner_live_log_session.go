package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

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
		if errors.Is(err, runneraction.ErrLiveLogNotConfigured) {
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
