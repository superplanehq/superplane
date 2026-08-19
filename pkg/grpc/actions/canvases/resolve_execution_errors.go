package canvases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func ResolveExecutionErrors(ctx context.Context, db *gorm.DB, canvas *models.Canvas, executionIDs []uuid.UUID) (*pb.ResolveExecutionErrorsResponse, error) {
	uniqueExecutionIDs := uniqueExecutionIDs(executionIDs)
	if len(uniqueExecutionIDs) == 0 {
		return nil, grpcerrors.InvalidArgument(nil, "execution_ids are required")
	}

	executions, err := models.FindNodeExecutionsByIDs(canvas.ID, uniqueExecutionIDs)
	if err != nil {
		return nil, err
	}

	if len(executions) != len(uniqueExecutionIDs) {
		return nil, grpcerrors.NotFound(err, "execution not found")
	}

	invalidIDs := invalidErrorResolutionIDs(executions)
	if len(invalidIDs) > 0 {
		return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("executions not in error state: %s", strings.Join(invalidIDs, ", ")))
	}

	if err := models.ResolveExecutionErrors(canvas.ID, uniqueExecutionIDs); err != nil {
		return nil, err
	}

	return &pb.ResolveExecutionErrorsResponse{}, nil
}

func uniqueExecutionIDs(executionIDs []uuid.UUID) []uuid.UUID {
	unique := make([]uuid.UUID, 0, len(executionIDs))
	seen := make(map[uuid.UUID]struct{}, len(executionIDs))
	for _, id := range executionIDs {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func invalidErrorResolutionIDs(executions []models.CanvasNodeExecution) []string {
	invalidIDs := make([]string, 0)
	for _, execution := range executions {
		if execution.ResultReason == models.CanvasNodeExecutionResultReasonError ||
			execution.ResultReason == models.CanvasNodeExecutionResultReasonErrorResolved {
			continue
		}
		invalidIDs = append(invalidIDs, execution.ID.String())
	}
	return invalidIDs
}
