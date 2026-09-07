package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

// SyncFactoryVelocity asks the velocity sync worker to read a workspace now and
// returns before the read finishes.
//
// Velocity reads stored merges rather than calling GitHub while rendering, which
// is what keeps the page fast and its data durable. The cost is that a merge is
// only visible once a sync collected it, so a user who just merged something
// needs a way to ask for a fresh read.
//
// The work is handed to the worker over the message broker instead of running
// here. The API process does not run the sync worker in production, and a read
// of sixty days of history must not depend on the request that started it.
func SyncFactoryVelocity(
	ctx context.Context,
	organizationID string,
	req *pb.SyncFactoryVelocityRequest,
) (*pb.SyncFactoryVelocityResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to sync factory velocity")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to sync factory velocity")
	}

	factory, err := models.FindFactory(database.DB(ctx), orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to sync factory velocity")
	}

	// A workspace with no repository has nothing to read, and the UI explains
	// that instead of showing progress that never finishes.
	if !factoryHasVelocityRepository(factory) {
		return &pb.SyncFactoryVelocityResponse{Started: false}, nil
	}

	if err := messages.PublishFactoryVelocitySyncRequested(factoryID.String()); err != nil {
		return nil, factoryErrorToStatus(err, "failed to sync factory velocity")
	}

	return &pb.SyncFactoryVelocityResponse{Started: true}, nil
}

func factoryHasVelocityRepository(factory *models.Factory) bool {
	config := factory.OnboardingConfigValue()
	return config.VCSIntegrationID != "" && config.AppRepository != ""
}
