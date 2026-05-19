package public

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/runners"
	"github.com/superplanehq/superplane/pkg/runners/livelogs"
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

	group, stream, region, ok := resolveCloudWatchLogSink(execution.Metadata.Data())
	if !ok {
		http.Error(
			w,
			"Logs are not available for this execution yet. Check again shortly.",
			http.StatusNotFound,
		)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	_ = livelogs.StreamCloudWatchLogToNDJSON(r.Context(), w, flusher, group, stream, region)
}

func resolveCloudWatchLogSink(meta any) (group, stream, region string, ok bool) {
	m, _ := meta.(map[string]any)
	if m == nil {
		return "", "", "", false
	}

	if v, exists := m[runneraction.ExecutionMetadataTaskLog]; exists && v != nil {
		if sink := taskLogMapToSink(v); sink != nil && sink.CloudWatch != nil {
			g := strings.TrimSpace(sink.CloudWatch.LogGroupName)
			s := strings.TrimSpace(sink.CloudWatch.LogStreamName)
			if g != "" && s != "" {
				return g, s, strings.TrimSpace(sink.CloudWatch.Region), true
			}
		}
	}

	store := runners.NewPostgresStore()
	if brokerID, ok2 := m[runneraction.ExecutionMetadataBrokerTaskID].(string); ok2 && strings.TrimSpace(brokerID) != "" {
		if taskID, err := uuid.Parse(strings.TrimSpace(brokerID)); err == nil {
			if task, err := store.FindTask(taskID); err == nil {
				if sink := task.TaskLog.Data(); sink != nil && sink.CloudWatch != nil {
					g := strings.TrimSpace(sink.CloudWatch.LogGroupName)
					s := strings.TrimSpace(sink.CloudWatch.LogStreamName)
					if g != "" && s != "" {
						return g, s, strings.TrimSpace(sink.CloudWatch.Region), true
					}
				}
			}
		}
	}

	return "", "", "", false
}

func taskLogMapToSink(v any) *runners.FleetTaskLog {
	switch t := v.(type) {
	case runners.FleetTaskLog:
		return &t
	case *runners.FleetTaskLog:
		return t
	case map[string]any:
		ft := &runners.FleetTaskLog{}
		if typ, ok := t["type"].(string); ok {
			ft.Type = typ
		}
		if cw, ok := t["cloudwatch"].(map[string]any); ok {
			ft.CloudWatch = &struct {
				LogGroupName  string `json:"log_group_name"`
				LogStreamName string `json:"log_stream_name"`
				Region        string `json:"region,omitempty"`
			}{}
			if g, ok := cw["log_group_name"].(string); ok {
				ft.CloudWatch.LogGroupName = g
			}
			if s, ok := cw["log_stream_name"].(string); ok {
				ft.CloudWatch.LogStreamName = s
			}
			if r, ok := cw["region"].(string); ok {
				ft.CloudWatch.Region = r
			}
		}
		if ft.Type != "" {
			return ft
		}
	}
	return nil
}
