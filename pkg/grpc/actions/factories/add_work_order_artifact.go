package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func AddWorkOrderArtifact(
	ctx context.Context,
	organizationID string,
	req *pb.AddWorkOrderArtifactRequest,
) (*pb.AddWorkOrderArtifactResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order artifact")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order artifact")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order artifact")
	}

	artifactType, ok := artifactTypeFromProto(req.GetType())
	if !ok {
		return nil, factoryErrorToStatus(invalidArgument("artifact type is required"), "failed to add work order artifact")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	actor, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to add work order artifact")
	}

	var data map[string]any
	if req.GetData() != nil {
		data = req.GetData().AsMap()
	}

	db := database.DB(ctx)
	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order artifact")
	}

	order, err := factoryModel.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order artifact")
	}

	artifact, err := order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type:      artifactType,
		URL:       req.GetUrl(),
		Title:     req.GetTitle(),
		Body:      req.GetBody(),
		Data:      data,
		CreatedBy: &actor,
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order artifact")
	}

	//
	// Reload so the CreatedBy relationship is populated for the response.
	//
	artifacts, err := order.ListArtifacts(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order artifact")
	}

	created := artifact
	for i := range artifacts {
		if artifacts[i].ID == artifact.ID {
			created = &artifacts[i]
			break
		}
	}

	serialized, err := serializeArtifact(created)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order artifact")
	}

	return &pb.AddWorkOrderArtifactResponse{Artifact: serialized}, nil
}
