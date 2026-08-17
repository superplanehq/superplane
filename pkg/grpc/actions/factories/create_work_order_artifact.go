package factories

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func CreateWorkOrderArtifact(
	ctx context.Context,
	organizationID string,
	req *pb.CreateWorkOrderArtifactRequest,
) (*pb.CreateWorkOrderArtifactResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	artifactType, ok := artifactTypeFromProto(req.GetType())
	if !ok {
		return nil, factoryErrorToStatus(invalidArgument("artifact type is required"), "failed to create work order artifact")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to create work order artifact")
	}

	data := map[string]any{}
	if req.GetData() != nil {
		data = req.GetData().AsMap()
	}

	db := database.DB(ctx)
	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	order, err := factoryModel.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	artifact, err := order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type:      artifactType,
		Data:      data,
		CreatedBy: &userID,
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	if err := db.Preload("CreatedBy").First(artifact, "id = ?", artifact.ID).Error; err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factory.EventTypeOrderArtifactAdded,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	notification := messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: orgID.String(),
		FactoryID:      factoryID.String(),
		OrderID:        orderID.String(),
		EventType:      factory.EventTypeOrderArtifactAdded,
		ActorUserID:    userIDStr,
		ArtifactType:   artifactType,
	}
	if err := notification.Publish(); err != nil {
		log.WithError(err).Warnf("Failed to publish work order notification for order %s", orderID)
	}

	serialized, err := serializeArtifact(artifact)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order artifact")
	}

	return &pb.CreateWorkOrderArtifactResponse{Artifact: serialized}, nil
}
