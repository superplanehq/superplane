package canvases

import (
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

func ListNodeExecutionLogs(orgID uuid.UUID, canvasID string, executionID string, limit uint32, afterSequence int64) (*pb.ListNodeExecutionLogsResponse, error) {
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

	pageLimit := executionLogLimit(limit)
	var after *int64
	if afterSequence > 0 {
		after = &afterSequence
	}

	logs, err := models.ListNodeExecutionLogs(workflowID, execID, pageLimit+1, after)
	if err != nil {
		return nil, err
	}

	hasNextPage := len(logs) > pageLimit
	if hasNextPage {
		logs = logs[:pageLimit]
	}

	return &pb.ListNodeExecutionLogsResponse{
		Logs:         serializeNodeExecutionLogs(logs),
		HasNextPage:  hasNextPage,
		LastSequence: lastExecutionLogSequence(logs),
	}, nil
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

func serializeNodeExecutionLogs(logs []models.CanvasNodeExecutionLog) []*pb.CanvasNodeExecutionLog {
	result := make([]*pb.CanvasNodeExecutionLog, 0, len(logs))
	for _, log := range logs {
		pbLog := &pb.CanvasNodeExecutionLog{
			Id:          log.ID.String(),
			CanvasId:    log.WorkflowID.String(),
			NodeId:      log.NodeID,
			ExecutionId: log.ExecutionID.String(),
			RunId:       log.RunID.String(),
			Sequence:    log.Sequence,
			Type:        log.Type,
			CreatedAt:   timestamppb.New(*log.CreatedAt),
		}
		if log.Text != nil {
			pbLog.Text = *log.Text
		}
		if log.Message != nil {
			pbLog.Message = *log.Message
		}
		if log.CommandIndex != nil {
			pbLog.CommandIndex = int32(*log.CommandIndex)
		}
		if log.Status != nil {
			pbLog.Status = *log.Status
		}
		if log.DurationMs != nil {
			pbLog.DurationMs = *log.DurationMs
		}
		result = append(result, pbLog)
	}
	return result
}

func lastExecutionLogSequence(logs []models.CanvasNodeExecutionLog) int64 {
	if len(logs) == 0 {
		return 0
	}
	return logs[len(logs)-1].Sequence
}
