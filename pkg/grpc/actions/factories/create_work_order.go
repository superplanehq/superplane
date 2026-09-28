package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func CreateWorkOrder(ctx context.Context, organizationID string, req *pb.CreateWorkOrderRequest) (*pb.CreateWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, factoryErrorToStatus(invalidArgument("title is required"), "failed to create work order")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to create work order")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	autoStartLineID, err := optionalAutoStartLineID(req.GetAutoStartLineId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}
	if err := rejectAutoStartWhenUnavailable(db, factory, autoStartLineID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	assigneeIDs := []uuid.UUID{createdByID}
	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	err = db.Transaction(func(tx *gorm.DB) error {
		created, err := factory.CreateWorkOrderWithAutoStart(tx, title, req.GetDescription(), &createdByID, assigneeIDs, nil, autoStartLineID)
		if err != nil {
			return err
		}
		order = created
		result, bindErr := storedfiles.BindDescriptionFiles(
			ctx,
			tx,
			blob.Current(),
			orgID,
			factory.ID,
			order.ID,
			order.Description,
		)
		bound = result
		return bindErr
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factory.ID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after bind")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	workersctx.EmitWorkOrderCreated(db, factory, order)

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	return &pb.CreateWorkOrderResponse{
		Order: serialized,
	}, nil
}

func optionalAutoStartLineID(raw string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, invalidArgument("auto_start_line_id must be a UUID")
	}
	return &parsed, nil
}

func rejectAutoStartWhenUnavailable(db *gorm.DB, factory *models.Factory, lineID *uuid.UUID) error {
	if lineID == nil {
		return nil
	}
	if !factory.PlanningEnabled || !factory.PlanningConfidence {
		return invalidArgument("auto-start requires planning and confidence")
	}
	if _, err := factory.FindLine(db, *lineID); err != nil {
		if errors.Is(err, models.ErrFactoryLineNotFound) {
			return invalidArgument("auto_start_line_id is not a line in this workspace")
		}
		return err
	}
	return nil
}
