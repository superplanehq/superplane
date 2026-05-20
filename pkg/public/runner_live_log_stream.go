package public

import (
	"context"
	"errors"
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
)

// statusClientClosedRequest is the non-standard 499 status code (popularized by
// nginx) used to indicate that the client closed the connection before the
// server finished responding. We use it on this streaming endpoint to avoid
// emitting a 5xx (and a corresponding Sentry event) for the very common case
// of users navigating away from the live-log dialog, which aborts the in-flight
// fetch request.
const statusClientClosedRequest = 499

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

	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_BROKER_BASE_URL")), "/")
	token := strings.TrimSpace(os.Getenv("TASK_BROKER_AUTH_TOKEN"))
	if base == "" || token == "" {
		http.Error(w, "Live logs are not configured on this installation", http.StatusServiceUnavailable)
		return
	}
	upstream := base + "/v1/tasks/" + url.PathEscape(brokerTaskID) + "/live-logs"

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream, nil)
	if err != nil {
		if isClientDisconnect(r.Context(), err) {
			w.WriteHeader(statusClientClosedRequest)
			return
		}
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/x-ndjson")
	// Avoid upstream gzip; it adds latency and can buffer small NDJSON chunks.
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// When the client cancels the request (e.g. by closing the live-log
		// dialog), the request context is cancelled and the in-flight HTTP
		// call to the broker fails. That is not a server-side problem, so
		// respond with 499 to avoid generating Sentry noise.
		if isClientDisconnect(r.Context(), err) {
			w.WriteHeader(statusClientClosedRequest)
			return
		}
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

// isClientDisconnect reports whether the given error is the result of the
// client (i.e. the HTTP request initiator) cancelling its context, rather than
// an actual upstream/server failure. It checks both the request context state
// and the error chain to cover the common scenarios surfaced by net/http.
func isClientDisconnect(ctx context.Context, err error) bool {
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
