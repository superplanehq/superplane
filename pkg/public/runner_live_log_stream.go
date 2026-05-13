package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/runnerlive"
)

func decodeTaskLogSink(v any) (*runneraction.TaskLogSink, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var sink runneraction.TaskLogSink
	if err := json.Unmarshal(b, &sink); err != nil {
		return nil, err
	}
	if strings.TrimSpace(sink.Type) == "" {
		return nil, nil
	}
	return &sink, nil
}

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
	rawSink, ok := meta[runneraction.ExecutionMetadataTaskLog]
	if !ok || rawSink == nil {
		http.Error(
			w,
			"No live log sink recorded for this execution yet. When the task broker and fleet manager expose CloudWatch task logs, they appear here after the next poll.",
			http.StatusNotFound,
		)
		return
	}

	sink, err := decodeTaskLogSink(rawSink)
	if err != nil || sink == nil || sink.Type != "cloudwatch" || sink.CloudWatch == nil {
		http.Error(w, "Live logs are not configured for this execution", http.StatusNotFound)
		return
	}

	g := strings.TrimSpace(sink.CloudWatch.LogGroupName)
	st := strings.TrimSpace(sink.CloudWatch.LogStreamName)
	if g == "" || st == "" {
		http.Error(w, "Incomplete CloudWatch log descriptor", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	region := strings.TrimSpace(sink.CloudWatch.Region)
	if err := runnerlive.StreamCloudWatchLogToNDJSON(r.Context(), w, flusher, g, st, region); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "context canceled") {
			return
		}
		// Response may already be partially written; best-effort JSON line.
		_, _ = fmt.Fprintf(w, "{\"type\":\"error\",\"message\":%q}\n", err.Error())
		flusher.Flush()
	}
}
