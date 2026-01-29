package workflows

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	compb "github.com/superplanehq/superplane/pkg/protos/components"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateNodePause(ctx context.Context, registry *registry.Registry, workflowID, nodeID string, paused bool) (*pb.UpdateNodePauseResponse, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workflow_id")
	}

	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	var workflowNode *models.WorkflowNode
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedNode, err := models.LockWorkflowNodeForUpdate(tx, workflowUUID, nodeID)
		if err != nil {
			return err
		}

		if lockedNode.Type != models.NodeTypeComponent && lockedNode.Type != models.NodeTypeBlueprint {
			return status.Error(codes.InvalidArgument, "pause is only supported for component or blueprint nodes")
		}

		if paused {
			switch lockedNode.State {
			case models.WorkflowNodeStateError:
				return status.Error(codes.FailedPrecondition, "node is in error state")
			case models.WorkflowNodeStatePaused:
				// no-op
			default:
				lockedNode.State = models.WorkflowNodeStatePaused
				if err := lockedNode.UpdateState(tx, models.WorkflowNodeStatePaused); err != nil {
					return err
				}
			}
		} else if lockedNode.State == models.WorkflowNodeStatePaused {
			nextState, err := models.ResumeStateForNodeInTransaction(tx, lockedNode.WorkflowID, lockedNode.NodeID)
			if err != nil {
				return err
			}
			lockedNode.State = nextState
			if err := lockedNode.UpdateState(tx, nextState); err != nil {
				return err
			}
		}

		workflowNode = lockedNode
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "workflow node not found")
		}
		return nil, err
	}

	serializedNode, err := serializeWorkflowNode(workflowNode)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateNodePauseResponse{
		Node: serializedNode,
	}, nil
}

func serializeWorkflowNode(node *models.WorkflowNode) (*compb.Node, error) {
	var integrationID *string
	if node.AppInstallationID != nil {
		id := node.AppInstallationID.String()
		integrationID = &id
	}

	modelNode := models.Node{
		ID:            node.NodeID,
		Name:          node.Name,
		Type:          node.Type,
		Ref:           node.Ref.Data(),
		Configuration: node.Configuration.Data(),
		Metadata:      node.Metadata.Data(),
		Position:      node.Position.Data(),
		IsCollapsed:   node.IsCollapsed,
		IntegrationID: integrationID,
	}

	serialized := actions.NodesToProto([]models.Node{modelNode})
	if len(serialized) == 0 {
		return nil, status.Error(codes.Internal, "failed to serialize node")
	}

	serialized[0].Paused = node.State == models.WorkflowNodeStatePaused
	return serialized[0], nil
}
