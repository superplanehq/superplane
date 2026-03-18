package organizations

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetUsage(_ context.Context, orgID string) (*pb.GetUsageResponse, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization ID")
	}

	limits, err := usage.ResolveEffectiveLimits(orgUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve usage limits: %v", err)
	}

	counters, err := usage.GetUsageCounters(orgUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get usage counters: %v", err)
	}

	return &pb.GetUsageResponse{
		Usage: &pb.UsageInfo{
			ProfileName:  limits.ProfileName,
			HasOverrides: limits.HasOverrides,
			IsUnlimited:  limits.IsUnlimited,
			EffectiveLimits: &pb.UsageLimits{
				MaxOrgsPerAccount:     int32(limits.MaxOrgsPerAccount),
				MaxCanvasesPerOrg:     int32(limits.MaxCanvasesPerOrg),
				MaxNodesPerCanvas:     int32(limits.MaxNodesPerCanvas),
				MaxUsersPerOrg:        int32(limits.MaxUsersPerOrg),
				MaxIntegrationsPerOrg: int32(limits.MaxIntegrationsPerOrg),
				MaxEventsPerMonth:     int32(limits.MaxEventsPerMonth),
				RetentionDays:         int32(limits.RetentionDays),
			},
			CurrentUsage: &pb.UsageCounters{
				Canvases:        counters.Canvases,
				Users:           counters.Users,
				Integrations:    counters.Integrations,
				EventsThisMonth: counters.EventsMonth,
			},
		},
	}, nil
}
