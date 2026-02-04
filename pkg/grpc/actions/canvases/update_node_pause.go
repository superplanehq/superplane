package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateNodePause(ctx context.Context, registry *registry.Registry, canvasID, nodeID string, paused bool) (*pb.UpdateNodePauseResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	var canvasNode *models.CanvasNode
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedNode, err := models.LockCanvasNodeForUpdate(tx, canvasUUID, nodeID)
		if err != nil {
			return err
		}

		if lockedNode.Type != models.NodeTypeComponent && lockedNode.Type != models.NodeTypeBlueprint {
			return status.Error(codes.InvalidArgument, "pause is only supported for component or blueprint nodes")
		}

		if paused {
			switch lockedNode.State {
			case models.CanvasNodeStateError:
				return status.Error(codes.FailedPrecondition, "node is in error state")
			case models.CanvasNodeStatePaused:
				// no-op
			default:
				lockedNode.State = models.CanvasNodeStatePaused
				if err := lockedNode.UpdateState(tx, models.CanvasNodeStatePaused); err != nil {
					return err
				}
			}
		} else if lockedNode.State == models.CanvasNodeStatePaused {
			nextState, err := models.ResumeStateForNodeInTransaction(tx, lockedNode.WorkflowID, lockedNode.NodeID)
			if err != nil {
				return err
			}
			lockedNode.State = nextState
			if err := lockedNode.UpdateState(tx, nextState); err != nil {
				return err
			}
		}

		canvasNode = lockedNode
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas node not found")
		}
		return nil, err
	}

	return &pb.UpdateNodePauseResponse{
		Node: serializeCanvasNodeState(canvasNode),
	}, nil
}
