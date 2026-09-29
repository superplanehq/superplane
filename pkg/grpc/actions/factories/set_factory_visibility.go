package factories

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
)

func SetFactoryVisibility(ctx context.Context, organizationID string, req *pb.SetFactoryVisibilityRequest) (*pb.SetFactoryVisibilityResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to set factory visibility")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to set factory visibility")
	}

	if err := factory.SetPublic(db, req.GetPublic()); err != nil {
		return nil, factoryErrorToStatus(err, "failed to set factory visibility")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(factory.ID.String(), "", "visibility"); err != nil {
		log.WithError(err).Warnf("Failed to publish factory visibility change for %s", factory.ID)
	}
	eventdistributer.NotifyPublicBoard(factory.ID.String())

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to set factory visibility")
	}

	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to set factory visibility")
	}

	return &pb.SetFactoryVisibilityResponse{Factory: serialized}, nil
}
