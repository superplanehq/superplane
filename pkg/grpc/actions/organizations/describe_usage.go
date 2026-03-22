package organizations

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeUsage(ctx context.Context, usageService usage.Service, orgID string) (*pb.DescribeUsageResponse, error) {
	if usageService == nil || !usageService.Enabled() {
		return &pb.DescribeUsageResponse{
			Enabled:       false,
			StatusMessage: "Usage tracking is disabled for this SuperPlane instance.",
		}, nil
	}

	readStartedAt := time.Now()
	limits, err := describeUsageLimits(ctx, usageService, orgID)
	if err != nil {
		return nil, err
	}

	if err := usage.CacheOrganizationLimits(orgID, limits, readStartedAt, time.Now()); err != nil {
		log.Warnf("Failed to persist usage limits cache for organization %s: %v", orgID, err)
	}

	orgUsage, err := describeUsageMetrics(ctx, usageService, orgID)
	if err != nil {
		return nil, err
	}

	if err := usage.MarkOrganizationSyncedIfUnset(orgID); err != nil {
		log.Warnf("Failed to persist usage sync state for organization %s: %v", orgID, err)
	}

	return &pb.DescribeUsageResponse{
		Enabled:       true,
		StatusMessage: "Usage tracking is active.",
		Limits:        serializeUsageLimits(limits),
		Usage:         serializeUsage(orgUsage),
	}, nil
}

func describeUsageLimits(
	ctx context.Context,
	usageService usage.Service,
	orgID string,
) (*usagepb.OrganizationLimits, error) {
	response, err := usageService.DescribeOrganizationLimits(ctx, orgID)
	if err == nil {
		return response.Limits, nil
	}

	if status.Code(err) != codes.NotFound {
		log.Errorf("Error describing usage limits for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to describe organization usage limits")
	}

	if err := usage.SyncOrganizationForce(ctx, usageService, orgID); err != nil {
		return nil, usageSyncError(orgID, err)
	}

	response, err = usageService.DescribeOrganizationLimits(ctx, orgID)
	if err != nil {
		log.Errorf("Error describing usage limits after sync for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to describe organization usage limits")
	}

	return response.Limits, nil
}

func describeUsageMetrics(
	ctx context.Context,
	usageService usage.Service,
	orgID string,
) (*usagepb.OrganizationUsage, error) {
	response, err := usageService.DescribeOrganizationUsage(ctx, orgID)
	if err == nil {
		return response.Usage, nil
	}

	if status.Code(err) != codes.NotFound {
		log.Errorf("Error describing usage metrics for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to describe organization usage")
	}

	if err := usage.SyncOrganizationForce(ctx, usageService, orgID); err != nil {
		return nil, usageSyncError(orgID, err)
	}

	response, err = usageService.DescribeOrganizationUsage(ctx, orgID)
	if err != nil {
		log.Errorf("Error describing usage metrics after setup for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to describe organization usage")
	}

	return response.Usage, nil
}

func usageSyncError(orgID string, err error) error {
	switch {
	case errors.Is(err, usage.ErrNoBillingAccountCandidate), errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.FailedPrecondition, "organization has no billing account candidate")
	case status.Code(err) == codes.ResourceExhausted:
		return status.Error(codes.ResourceExhausted, "organization exceeds configured account usage limits")
	default:
		log.Errorf("Error syncing usage for organization %s: %v", orgID, err)
		return status.Error(codes.Internal, "failed to set up organization usage")
	}
}

func serializeUsageLimits(limits *usagepb.OrganizationLimits) *pb.OrganizationLimits {
	if limits == nil {
		return nil
	}

	return &pb.OrganizationLimits{
		MaxCanvases:         limits.MaxCanvases,
		MaxNodesPerCanvas:   limits.MaxNodesPerCanvas,
		MaxUsers:            limits.MaxUsers,
		RetentionWindowDays: limits.RetentionWindowDays,
		MaxEventsPerMonth:   limits.MaxEventsPerMonth,
		MaxIntegrations:     limits.MaxIntegrations,
	}
}

func serializeUsage(orgUsage *usagepb.OrganizationUsage) *pb.OrganizationUsage {
	if orgUsage == nil {
		return nil
	}

	var eventBucketLastUpdatedAt *timestamppb.Timestamp
	if orgUsage.EventBucketLastUpdatedAtUnixSeconds > 0 {
		eventBucketLastUpdatedAt = timestamppb.New(
			time.Unix(orgUsage.EventBucketLastUpdatedAtUnixSeconds, 0).UTC(),
		)
	}

	return &pb.OrganizationUsage{
		Canvases:                 orgUsage.Canvases,
		EventBucketLevel:         orgUsage.EventBucketLevel,
		EventBucketCapacity:      orgUsage.EventBucketCapacity,
		EventBucketLastUpdatedAt: eventBucketLastUpdatedAt,
	}
}
