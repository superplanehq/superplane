package organizations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func ListOrganizationRunnerFleets(
	ctx context.Context,
	orgID string,
) (*pb.ListOrganizationRunnerFleetsResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)
	organization, err := models.FindOrganizationByIDOrSlug(db, organizationID.String())
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load organization")
	}

	var fleets []models.RunnerFleet
	if organization.HasExperimentalFeature(features.FeatureNewRunners) {
		fleets, err = models.ListEnabledRunnerFleetsForOrganization(db, organizationID)
	} else {
		fleets = models.DefaultInstallationRunnerFleets(models.DefaultRunnerVersion)
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list organization runner fleets")
	}

	out := make([]*pb.OrganizationRunnerFleet, 0, len(fleets))
	for _, fleet := range fleets {
		out = append(out, serializeOrganizationRunnerFleet(fleet))
	}

	return &pb.ListOrganizationRunnerFleetsResponse{Fleets: out}, nil
}

func serializeOrganizationRunnerFleet(fleet models.RunnerFleet) *pb.OrganizationRunnerFleet {
	spec := fleet.Spec.Data()
	return &pb.OrganizationRunnerFleet{
		Id:    fleet.Slug,
		Scope: fleet.ScopeType,
		Spec: &pb.OrganizationRunnerFleetSpec{
			OperatingSystem: spec.OperatingSystem,
			Architecture:    spec.Architecture,
			CpuMillicores:   spec.CPUMillicores,
			MemoryMb:        spec.MemoryMB,
			DiskGb:          spec.DiskGB,
			Capabilities:    spec.Capabilities,
		},
	}
}
