package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/registry"
)

func SwitchFactoryModelSource(
	ctx context.Context,
	reg *registry.Registry,
	organizationID string,
	req *pb.SwitchFactoryModelSourceRequest,
) (*pb.SwitchFactoryModelSourceResponse, error) {
	if req == nil {
		return nil, grpcerrors.InvalidArgument(nil, "request is required")
	}

	changed, integrationID, err := SwitchFactoryModelSourceInTransaction(
		ctx,
		reg,
		organizationID,
		req.GetId(),
		req.GetSource(),
		req.GetApiKey(),
	)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to switch model source")
	}

	for _, canvasID := range changed {
		publishCanvasUpdated(canvasID, organizationID)
	}

	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to switch model source")
	}
	factory, err := findFactory(database.DB(ctx), orgID, req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to switch model source")
	}

	return &pb.SwitchFactoryModelSourceResponse{
		Factory:       serializeFactory(factory),
		IntegrationId: integrationID,
	}, nil
}

func publishCanvasUpdated(canvasID uuid.UUID, organizationID string) {
	if err := messages.NewCanvasUpdatedMessage(canvasID.String(), organizationID).PublishUpdated(); err != nil {
		return
	}
}
