package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func CancelWorkOrderQueue(ctx context.Context, organizationID string, req *pb.CancelWorkOrderQueueRequest) (*pb.CancelWorkOrderQueueResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove task from queue")
	}
	if _, ok := authentication.GetUserIdFromMetadata(ctx); !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	queueID, err := uuid.Parse(req.GetQueueId())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "queue id must be a UUID")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove task from queue")
	}
	order, err := findWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove task from queue")
	}
	var cancellation *models.RunCancellationResult
	err = db.Transaction(func(tx *gorm.DB) error {
		var err error
		cancellation, err = order.CancelQueuedTask(tx, queueID)
		return err
	})
	if errors.Is(err, models.ErrTaskNoLongerQueued) {
		return nil, grpcerrors.FailedPrecondition(err, "This task is no longer queued. Refresh and try again.")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove task from queue")
	}
	if cancellation != nil {
		messages.PublishRunCancellationDrain(cancellation.Run.WorkflowID, cancellation.Drain)
		if err := messages.NewCanvasRunMessage(cancellation.Run.WorkflowID.String(), cancellation.Run.ID.String()).Publish(); err != nil {
			log.WithError(err).Warn("Failed to publish queued run cancellation")
		}
	}
	if err := messages.PublishFactoryWorkOrderUpdated(factory.ID.String(), order.ID.String(), factoryevents.EventTypeOrderStatusUpdated); err != nil {
		log.WithError(err).WithField("order_id", order.ID).Warn("Failed to publish queue cancellation")
	}
	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load task after queue cancellation")
	}
	return &pb.CancelWorkOrderQueueResponse{Order: serialized}, nil
}
