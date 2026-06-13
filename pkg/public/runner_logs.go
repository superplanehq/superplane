package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type runnerLogBatchRequest struct {
	TaskID  string            `json:"task_id"`
	Records []runnerLogRecord `json:"records"`
}

type runnerLogRecord struct {
	Seq        int64  `json:"seq"`
	TaskID     string `json:"task_id"`
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	Message    string `json:"message,omitempty"`
	Index      *int   `json:"index,omitempty"`
	Status     string `json:"status,omitempty"`
	DurationMs *int64 `json:"duration_ms,omitempty"`
	Timestamp  *int64 `json:"timestamp,omitempty"`
}

func (s *Server) handleRunnerLogs(w http.ResponseWriter, r *http.Request) {
	if !validRunnerLogToken(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var batch runnerLogBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	taskID := strings.TrimSpace(batch.TaskID)
	if taskID == "" {
		http.Error(w, "task_id required", http.StatusBadRequest)
		return
	}

	execution, err := models.FirstNodeExecutionByKVValue("task_id", taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}

	if err := models.CreateNodeExecutionLogs(runnerLogRecordsToModelLogs(batch.Records, taskID, execution)); err != nil {
		http.Error(w, "persist failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validRunnerLogToken(r *http.Request) bool {
	want := strings.TrimSpace(os.Getenv("TASK_BROKER_AUTH_TOKEN"))
	if want == "" {
		return false
	}

	got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return got != "" && got == want
}

func runnerLogRecordsToModelLogs(records []runnerLogRecord, taskID string, execution *models.CanvasNodeExecution) []models.CanvasNodeExecutionLog {
	logs := make([]models.CanvasNodeExecutionLog, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.TaskID) != "" && strings.TrimSpace(record.TaskID) != taskID {
			continue
		}
		if record.Seq <= 0 {
			continue
		}

		logs = append(logs, runnerLogRecordToModelLog(record, execution))
	}
	return logs
}

func runnerLogRecordToModelLog(record runnerLogRecord, execution *models.CanvasNodeExecution) models.CanvasNodeExecutionLog {
	log := models.CanvasNodeExecutionLog{
		WorkflowID:   execution.WorkflowID,
		RunID:        execution.RunID,
		NodeID:       execution.NodeID,
		ExecutionID:  execution.ID,
		Sequence:     record.Seq,
		Type:         strings.TrimSpace(record.Type),
		Text:         stringPointer(record.Text),
		Message:      stringPointer(record.Message),
		CommandIndex: record.Index,
		Status:       stringPointer(record.Status),
		DurationMs:   record.DurationMs,
	}
	if log.Type == "" {
		log.Type = models.CanvasNodeExecutionLogTypeLine
	}
	return log
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
