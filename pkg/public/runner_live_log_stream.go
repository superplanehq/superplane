package public

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/runners"
)

func (s *Server) handleRunnerLiveLogStream(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	allowed, err := s.authService.CheckOrganizationPermission(
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

	if _, err := models.FindCanvas(user.OrganizationID, canvasID); err != nil {
		http.Error(w, "Canvas not found", http.StatusNotFound)
		return
	}

	execution, err := models.FindNodeExecution(canvasID, executionID)
	if err != nil {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	node, err := models.FindCanvasNode(database.Conn(), canvasID, execution.NodeID)
	if err != nil {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	ref := node.Ref.Data()
	if ref.Component == nil || ref.Component.Name != "runner" {
		http.Error(w, "Live logs are only available for Runner components", http.StatusBadRequest)
		return
	}

	meta := execution.Metadata.Data()
	var brokerTaskID string
	if v, ok := meta[runneraction.ExecutionMetadataBrokerTaskID]; ok && v != nil {
		if s, ok2 := v.(string); ok2 {
			brokerTaskID = strings.TrimSpace(s)
		} else {
			brokerTaskID = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	if brokerTaskID == "" {
		http.Error(
			w,
			"Logs are not available for this execution yet. Check again shortly.",
			http.StatusNotFound,
		)
		return
	}

	fleetURL, authToken, err := resolveFleetForExecution(executionID)
	if err != nil || fleetURL == "" {
		http.Error(w, "Live logs are not configured on this installation", http.StatusServiceUnavailable)
		return
	}

	upstream := strings.TrimRight(fleetURL, "/") + "/v1/tasks/" + url.PathEscape(brokerTaskID) + "/live-logs"

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream, nil)
	if err != nil {
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	req.Header.Set("Accept", "application/x-ndjson")
	// Avoid upstream gzip; it adds latency and can buffer small NDJSON chunks.
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if ct := resp.Header.Get("Content-Type"); ct != "" {
			w.Header().Set("Content-Type", ct)
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		return
	}

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	flusher, flusherOK := w.(http.Flusher)
	if flusherOK {
		flusher.Flush()
	}

	buf := make([]byte, 16*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return
			}
			if flusherOK {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return
		}
	}
}

// resolveFleetForExecution looks up the fleet URL and auth token for a given execution.
// It first tries to find a runner_task record linked to the execution (new architecture).
// If none exists it falls back to the legacy TASK_BROKER_* env vars.
func resolveFleetForExecution(executionID uuid.UUID) (fleetURL, authToken string, err error) {
	store := runners.NewPostgresStore()

	task, taskErr := store.FindTaskByExecutionID(executionID)
	if taskErr == nil {
		fleet, fleetErr := store.FindFleet(task.FleetID)
		if fleetErr == nil {
			return fleet.FleetURL, fleet.AuthToken, nil
		}
	}

	// Fall back to legacy env vars.
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_BROKER_BASE_URL")), "/")
	token := strings.TrimSpace(os.Getenv("TASK_BROKER_AUTH_TOKEN"))
	return base, token, nil
}
