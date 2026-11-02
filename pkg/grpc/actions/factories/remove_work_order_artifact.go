package factories

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/datatypes"
)

func RemoveWorkOrderArtifact(
	ctx context.Context,
	organizationID string,
	req *pb.RemoveWorkOrderArtifactRequest,
) (*pb.RemoveWorkOrderArtifactResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order artifact")
	}

	rawUserID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to remove work order artifact")
	}

	db := database.DB(ctx)
	factoryModel, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order artifact")
	}

	order, err := findWorkOrder(db, factoryModel, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order artifact")
	}
	if order.IsClosed() {
		return nil, grpcerrors.Conflict(nil, "Task is closed or archived")
	}

	removed, err := order.RemoveArtifact(db, req.GetKey(), req.GetTitle(), &userID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order artifact")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryModel.ID.String(),
		order.ID.String(),
		factory.EventTypeOrderArtifactsCleared,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	return &pb.RemoveWorkOrderArtifactResponse{
		Id:    removed.ID.String(),
		Title: artifactDataTitle(removed.Data),
	}, nil
}

func artifactDataTitle(data datatypes.JSON) string {
	fields := map[string]any{}
	if err := json.Unmarshal(json.RawMessage(data), &fields); err != nil {
		return ""
	}
	title, _ := fields["title"].(string)
	return strings.TrimSpace(title)
}
