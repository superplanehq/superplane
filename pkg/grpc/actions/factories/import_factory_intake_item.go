package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
)

func ImportFactoryIntakeItem(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.ImportFactoryIntakeItemRequest,
) (*pb.ImportFactoryIntakeItemResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	intakeID, err := parseIntakeID(req.GetIntakeId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	itemID := strings.TrimSpace(req.GetItemId())
	if itemID == "" {
		return nil, factoryErrorToStatus(invalidArgument("item id is required"), "failed to import factory intake item")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to import factory intake item")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	intake, err := factory.FindIntake(db, intakeID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	source, err := deps.itemSource(ctx, db, intake)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	item, err := source.Get(ctx, itemID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	origin := models.WorkOrderOrigin{URL: item.URL, Label: models.OriginLabelFromURL(item.URL)}
	order, err := factory.CreateWorkOrderWithOrigin(
		db,
		item.Title,
		item.Body,
		&createdByID,
		[]uuid.UUID{createdByID},
		nil,
		origin,
	)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
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
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	return &pb.ImportFactoryIntakeItemResponse{Order: serialized}, nil
}
