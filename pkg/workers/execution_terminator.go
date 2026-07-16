package workers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type ExecutionTerminator struct {
	encryptor   crypto.Encryptor
	registry    *registry.Registry
	authService authorization.Authorization
	semaphore   *semaphore.Weighted
	logger      *log.Entry
	rabbitMQURL string
	consumer    *tackle.Consumer
}

func NewExecutionTerminator(
	rabbitMQURL string,
	authService authorization.Authorization,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
) *ExecutionTerminator {
	return &ExecutionTerminator{
		encryptor:   encryptor,
		registry:    registry,
		authService: authService,
		semaphore:   semaphore.NewWeighted(25),
		logger:      log.WithFields(log.Fields{"worker": "ExecutionTerminator"}),
		rabbitMQURL: rabbitMQURL,
	}
}

func (w *ExecutionTerminator) Name() string {
	return "ExecutionTerminator"
}

func (w *ExecutionTerminator) Start(ctx context.Context) {
	go w.startCancellingConsumer(ctx)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			executions, err := models.ListCancellingNodeExecutions()
			if err != nil {
				w.logger.Errorf("Error finding cancelling executions: %v", err)
				continue
			}

			for _, execution := range executions {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(execution models.CanvasNodeExecution) {
					defer w.semaphore.Release(1)

					if err := w.LockAndCancelExecution(execution.ID); err != nil {
						if errors.Is(err, ErrRecordLocked) {
							return
						}
						w.logger.Errorf("Error terminating execution %s: %v", execution.ID, err)
					}
				}(execution)
			}
		}
	}
}

func (w *ExecutionTerminator) startCancellingConsumer(ctx context.Context) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name(),
		RemoteExchange: messages.ExecutionsExchange,
		Service:        messages.ExecutionsExchange + "." + messages.ExecutionCancellingRoutingKey + "." + w.Name(),
		RoutingKey:     messages.ExecutionCancellingRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))
	w.consumer = consumer

	for {
		if ctx.Err() != nil {
			return
		}

		w.logger.Infof("Connecting to RabbitMQ queue for %s events", messages.ExecutionCancellingRoutingKey)

		err := w.consumer.Start(&options, w.consumeCancelling)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.ExecutionCancellingRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.ExecutionCancellingRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *ExecutionTerminator) consumeCancelling(delivery tackle.Delivery) error {
	data := &pb.CanvasNodeExecutionMessage{}
	if err := proto.Unmarshal(delivery.Body(), data); err != nil {
		w.logger.Errorf("Error unmarshaling canvas execution message: %v", err)
		return err
	}

	executionID, err := uuid.Parse(data.Id)
	if err != nil {
		w.logger.Errorf("Error parsing execution id: %v", err)
		return err
	}

	err = w.LockAndCancelExecution(executionID)
	if err == nil {
		return nil
	}

	if errors.Is(err, ErrRecordLocked) {
		return nil
	}

	w.logger.Errorf("Error terminating execution %s: %v", executionID, err)
	return err
}

func (w *ExecutionTerminator) LockAndCancelExecution(executionID uuid.UUID) error {
	var workflowID uuid.UUID
	finished := false

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		execution, err := models.LockCancellingNodeExecutionInActiveCanvas(tx, executionID)
		if err != nil {
			return err
		}

		workflowID = execution.WorkflowID

		canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID)
		if err != nil {
			return err
		}

		if err := w.cancelComponent(tx, canvas.OrganizationID.String(), execution); err != nil {
			return err
		}

		finished = true
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRecordLocked
		}
		return err
	}

	if !finished {
		return nil
	}

	if err := messages.PublishCanvasExecutionByID(workflowID, executionID); err != nil {
		w.logger.Errorf("failed to publish execution finished RabbitMQ message: %v", err)
	}

	return nil
}

func (w *ExecutionTerminator) cancelComponent(tx *gorm.DB, organizationID string, execution *models.CanvasNodeExecution) error {
	node, err := models.FindCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return err
	}

	if node.Type != models.NodeTypeComponent {
		return nil
	}

	ref := node.Ref.Data()
	if ref.Component == nil {
		return nil
	}

	action, err := w.registry.GetAction(ref.Component.Name)
	if err != nil {
		log.Errorf("action %s not found: %v", ref.Component.Name, err)
		return err
	}

	logger := logging.ForExecution(execution)
	orgUUID := uuid.MustParse(organizationID)
	canvasName := ""
	if workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID); err == nil && workflow != nil {
		canvasName = workflow.Name
	}

	ctx := core.ExecutionContext{
		ID:             execution.ID,
		WorkflowID:     execution.WorkflowID.String(),
		OrganizationID: organizationID,
		CanvasName:     canvasName,
		NodeID:         execution.NodeID,
		NodeName:       node.Name,
		Configuration:  execution.Configuration.Data(),
		HTTP:           w.registry.HTTPContextInTransaction(tx),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution, nil),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Auth:           contexts.NewAuthReader(tx, orgUUID, w.authService, nil),
		CanvasMemory:   contexts.NewCanvasMemoryContext(tx, execution.WorkflowID),
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			logger.Errorf("error finding app installation: %v", err)
			return grpcerrors.Internal(err, "error building context")
		}

		logger = logging.WithIntegration(logger, *integration)
		ctx.Integration = contexts.NewIntegrationContext(tx, node, integration, w.encryptor, w.registry, nil)
	}

	ctx.Logger = logger
	if err := action.Cancel(ctx); err != nil {
		log.Errorf("failed to cancel component execution %s: %v", execution.ID.String(), err)
	}

	return execution.CancelInTransaction(tx, execution.CancelledBy)
}
