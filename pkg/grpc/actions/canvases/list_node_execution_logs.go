package canvases

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	DefaultExecutionLogLimit = 500
	MaxExecutionLogLimit     = 1000
)

// ListNodeExecutionLogs fetches logs for a runner node execution by proxying
// the task-broker's CloudWatch history endpoint.
func ListNodeExecutionLogs(orgID uuid.UUID, canvasID string, executionID string, brokerBaseURL string, brokerAuthToken string, limit uint32, afterSequence int64) (*pb.ListNodeExecutionLogsResponse, error) {
	workflowID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	execID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid execution id: %v", err)
	}

	if _, err := models.FindCanvas(orgID, workflowID); err != nil {
		return nil, status.Error(codes.NotFound, "canvas not found")
	}

	execution, err := models.FindNodeExecution(workflowID, execID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "execution not found")
		}
		return nil, err
	}

	node, err := models.FindCanvasNode(database.Conn(), workflowID, execution.NodeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "node not found")
		}
		return nil, err
	}

	ref := node.Ref.Data()
	if ref.Component == nil || !runneraction.IsRunnerComponent(ref.Component.Name) {
		return nil, status.Error(codes.InvalidArgument, "logs are only available for runner executions")
	}

	brokerTaskID := runneraction.BrokerTaskIDFromExecutionMetadata(execution.Metadata.Data())
	if strings.TrimSpace(brokerTaskID) == "" {
		return nil, status.Error(codes.NotFound, "no task associated with this execution")
	}

	logs, err := fetchLogsFromBroker(brokerBaseURL, brokerAuthToken, brokerTaskID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch logs: %v", err)
	}

	// Apply afterSequence filter and limit.
	pageLimit := executionLogLimit(limit)
	filtered := make([]*pb.CanvasNodeExecutionLog, 0, len(logs))
	for _, l := range logs {
		if afterSequence > 0 && l.Sequence <= afterSequence {
			continue
		}
		filtered = append(filtered, l)
	}

	hasNextPage := len(filtered) > pageLimit
	if hasNextPage {
		filtered = filtered[:pageLimit]
	}

	var lastSeq int64
	if len(filtered) > 0 {
		lastSeq = filtered[len(filtered)-1].Sequence
	}

	return &pb.ListNodeExecutionLogsResponse{
		Logs:         filtered,
		HasNextPage:  hasNextPage,
		LastSequence: lastSeq,
	}, nil
}

// fetchLogsFromBroker calls GET /v1/tasks/{id}/logs on the broker and parses
// the NDJSON response into proto log records.
func fetchLogsFromBroker(brokerBaseURL, authToken, taskID string) ([]*pb.CanvasNodeExecutionLog, error) {
	url := fmt.Sprintf("%s/v1/tasks/%s/logs", strings.TrimRight(brokerBaseURL, "/"), taskID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("broker returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return parseBrokerLogNDJSON(resp.Body)
}

// parseBrokerLogNDJSON parses the NDJSON stream from the broker into proto records.
func parseBrokerLogNDJSON(r io.Reader) ([]*pb.CanvasNodeExecutionLog, error) {
	var logs []*pb.CanvasNodeExecutionLog
	var seq int64

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}

		seq++
		pbLog := &pb.CanvasNodeExecutionLog{
			Sequence:  seq,
			CreatedAt: timestamppb.Now(),
		}

		if t, ok := rec["type"].(string); ok {
			pbLog.Type = t
		}
		if text, ok := rec["text"].(string); ok {
			pbLog.Text = text
		}
		if msg, ok := rec["message"].(string); ok {
			pbLog.Message = msg
		}
		if s, ok := rec["status"].(string); ok {
			pbLog.Status = s
		}
		if idx, ok := rec["index"].(float64); ok {
			pbLog.CommandIndex = int32(idx)
		}
		if dur, ok := rec["duration_ms"].(float64); ok {
			pbLog.DurationMs = int64(dur)
		}

		logs = append(logs, pbLog)
	}

	return logs, scanner.Err()
}

func executionLogLimit(limit uint32) int {
	if limit == 0 {
		return DefaultExecutionLogLimit
	}
	if limit > MaxExecutionLogLimit {
		return MaxExecutionLogLimit
	}
	return int(limit)
}
