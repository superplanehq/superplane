package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func CancelExecution(ctx context.Context, authService authorization.Authorization, encryptor crypto.Encryptor, organizationID string, registry *registry.Registry, workflowID, executionID uuid.UUID) (*pb.CancelExecutionResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	var user *models.User
	if userIsSet {
		var err error
		user, err = models.FindActiveUserByID(organizationID, userID)
		if err != nil {
			return nil, status.Error(codes.NotFound, "user not found")
		}
	}
	// If user is not set (like in tests), user will be nil and that's fine

	execution, err := models.FindNodeExecution(workflowID, executionID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "execution not found")
	}

	if execution.ParentExecutionID != nil {
		return nil, status.Error(codes.InvalidArgument, "cannot cancel child execution directly, cancel the parent execution instead")
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		node, err := models.FindCanvasNode(tx, workflowID, execution.NodeID)

		if err != nil {
			return status.Error(codes.NotFound, "Node not found for execution")
		}

		err = cancelExecutionInTransaction(tx, authService, encryptor, organizationID, registry, execution, node, user)

		if err != nil {
			return status.Error(codes.Internal, "It was not possible to cancel the execution")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pb.CancelExecutionResponse{}, nil
}

func cancelExecutionInTransaction(tx *gorm.DB, authService authorization.Authorization, encryptor crypto.Encryptor, organizationID string, registry *registry.Registry, execution *models.CanvasNodeExecution, node *models.CanvasNode, user *models.User) error {
	if node.Type == models.NodeTypeBlueprint {
		err := cancelChildExecutions(tx, authService, organizationID, encryptor, registry, execution, user)
		if err != nil {
			log.Errorf("failed to cancel child executions for %s: %v", execution.ID.String(), err)
			return err
		}
	}

	if node.Type == models.NodeTypeComponent {
		ref := node.Ref.Data()
		if ref.Component != nil {
			component, err := registry.GetComponent(ref.Component.Name)
			if err != nil {
				log.Errorf("component %s not found: %v", ref.Component.Name, err)
				return err
			}

			logger := logging.ForExecution(execution, nil)
			orgUUID := uuid.MustParse(organizationID)
			ctx := core.ExecutionContext{
				ID:             execution.ID,
				WorkflowID:     execution.WorkflowID.String(),
				Configuration:  execution.Configuration.Data(),
				HTTP:           registry.HTTPContext(),
				Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
				ExecutionState: contexts.NewExecutionStateContext(tx, execution),
				Requests:       contexts.NewExecutionRequestContext(tx, execution),
				Auth:           contexts.NewAuthContext(tx, orgUUID, authService, user),
				Notifications:  contexts.NewNotificationContext(tx, orgUUID, execution.WorkflowID),
				CanvasMemory:   contexts.NewCanvasMemoryContext(tx, execution.WorkflowID),
			}

			if node.AppInstallationID != nil {
				integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
				if err != nil {
					logger.Errorf("error finding app installation: %v", err)
					return status.Error(codes.Internal, "error building context")
				}

				logger = logging.WithIntegration(logger, *integration)
				ctx.Integration = contexts.NewIntegrationContext(tx, node, integration, encryptor, registry)
			}

			ctx.Logger = logger
			if err := component.Cancel(ctx); err != nil {
				log.Errorf("failed to cancel component execution %s: %v", execution.ID.String(), err)
			}
		}
	}

	var cancelledBy *uuid.UUID
	if user != nil {
		cancelledBy = &user.ID
	}

	return execution.CancelInTransaction(tx, cancelledBy)
}

func cancelChildExecutions(
	tx *gorm.DB,
	authService authorization.Authorization,
	organizationID string,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	parentExecution *models.CanvasNodeExecution,
	user *models.User,
) error {
	childExecutions, err := models.FindChildExecutionsInTransaction(
		tx,
		parentExecution.ID,
		[]string{models.CanvasNodeExecutionStatePending, models.CanvasNodeExecutionStateStarted},
	)

	if err != nil {
		return err
	}

	if len(childExecutions) == 0 {
		return nil
	}

	nodeIDMap := make(map[string]bool)
	for _, execution := range childExecutions {
		nodeIDMap[execution.NodeID] = true
	}

	nodeIDs := make([]string, 0, len(nodeIDMap))
	for nodeID := range nodeIDMap {
		nodeIDs = append(nodeIDs, nodeID)
	}

	nodes, err := models.FindCanvasNodesByIDs(tx, parentExecution.WorkflowID, nodeIDs)
	if err != nil {
		return err
	}

	nodeMap := make(map[string]*models.CanvasNode)
	for i := range nodes {
		nodeMap[nodes[i].NodeID] = &nodes[i]
	}

	for _, childExecution := range childExecutions {
		childNode, exists := nodeMap[childExecution.NodeID]
		if !exists {
			log.Errorf("failed to find child node %s in fetched nodes", childExecution.NodeID)
			return err
		}

		err = cancelExecutionInTransaction(tx, authService, encryptor, organizationID, registry, &childExecution, childNode, user)
		if err != nil {
			log.Errorf("failed to cancel child execution %s: %v", childExecution.ID.String(), err)
			return err
		}
	}

	return nil
}
