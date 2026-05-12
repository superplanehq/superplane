package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// GetRunnerExecutionLogs loads CloudWatch log lines for a Runner node execution (same AWS credentials as Runner workers).
func GetRunnerExecutionLogs(ctx context.Context, canvasID, executionID uuid.UUID, nextForwardToken string) (*pb.GetRunnerExecutionLogsResponse, error) {
	exec, err := models.FindNodeExecution(canvasID, executionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "execution not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load execution: %v", err)
	}

	node, err := models.FindCanvasNode(database.Conn(), canvasID, exec.NodeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "node not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load node: %v", err)
	}

	ref := node.Ref.Data()
	if ref.Component == nil || ref.Component.Name != "runner" {
		return nil, status.Error(codes.InvalidArgument, "execution is not a Runner component")
	}

	meta := exec.Metadata.Data()
	groupName := nestedString(meta, "logs", "groupName")
	streamName := nestedString(meta, "logs", "streamName")
	if groupName == "" || streamName == "" {
		return nil, status.Error(codes.FailedPrecondition, "CloudWatch log stream is not available for this execution yet")
	}

	lines, token, err := runner.FetchCloudWatchLogEvents(groupName, streamName, nextForwardToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch logs: %v", err)
	}

	events := make([]*pb.RunnerLogEvent, 0, len(lines))
	for _, line := range lines {
		events = append(events, &pb.RunnerLogEvent{
			TimestampMs: line.TimestampMs,
			Message:     line.Message,
		})
	}

	return &pb.GetRunnerExecutionLogsResponse{
		Events:           events,
		NextForwardToken: token,
	}, nil
}

func nestedString(m map[string]any, key1, key2 string) string {
	v1, ok := m[key1]
	if !ok || v1 == nil {
		return ""
	}
	sub, ok := v1.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := sub[key2].(string)
	return s
}
