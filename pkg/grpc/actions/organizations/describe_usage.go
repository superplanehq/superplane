package organizations

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
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
	var limits *usagepb.OrganizationLimits
	err := telemetry.RunSpan(ctx, "usage.load_limits", func(ctx context.Context) error {
		var loadErr error
		limits, loadErr = describeUsageLimits(ctx, usageService, orgID)
		return loadErr
	})
	if err != nil {
		return nil, err
	}

	if err := usage.CacheOrganizationLimits(orgID, limits, readStartedAt, time.Now()); err != nil {
		log.Warnf("Failed to persist usage limits cache for organization %s: %v", orgID, err)
	}

	var orgUsage *usagepb.OrganizationUsage
	err = telemetry.RunSpan(ctx, "usage.load_metrics", func(ctx context.Context) error {
		var loadErr error
		orgUsage, loadErr = describeUsageMetrics(ctx, usageService, orgID)
		return loadErr
	})
	if err != nil {
		return nil, err
	}

	if err := usage.MarkOrganizationSyncedIfUnset(orgID); err != nil {
		log.Warnf("Failed to persist usage sync state for organization %s: %v", orgID, err)
	}

	go usage.ReconcileCanvasCount(orgID, orgUsage.GetCanvases())

	var response *pb.DescribeUsageResponse
	_ = telemetry.RunSpan(ctx, "usage.build_response", func(ctx context.Context) error {
		response = buildDescribeUsageResponse(limits, orgUsage)
		return nil
	})

	return response, nil
}

func buildDescribeUsageResponse(limits *usagepb.OrganizationLimits, orgUsage *usagepb.OrganizationUsage) *pb.DescribeUsageResponse {
	return &pb.DescribeUsageResponse{
		Enabled:       true,
		StatusMessage: "Usage tracking is active and up to date.",
		Limits:        serializeUsageLimits(limits),
		Usage:         serializeUsage(orgUsage),
	}
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

	if grpcerrors.Code(err) != codes.NotFound {
		log.Errorf("Error describing usage limits for organization %s: %v", orgID, err)
		return nil, grpcerrors.Internal(err, "failed to describe organization usage limits")
	}

	if err := usage.SyncOrganizationForce(ctx, usageService, orgID); err != nil {
		return nil, usageSyncError(orgID, err)
	}

	response, err = usageService.DescribeOrganizationLimits(ctx, orgID)
	if err != nil {
		return nil, describeUsageAfterSyncError(orgID, "limits", err)
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

	if grpcerrors.Code(err) != codes.NotFound {
		log.Errorf("Error describing usage metrics for organization %s: %v", orgID, err)
		return nil, grpcerrors.Internal(err, "failed to describe organization usage")
	}

	if err := usage.SyncOrganizationForce(ctx, usageService, orgID); err != nil {
		return nil, usageSyncError(orgID, err)
	}

	response, err = usageService.DescribeOrganizationUsage(ctx, orgID)
	if err != nil {
		return nil, describeUsageAfterSyncError(orgID, "metrics", err)
	}

	return response.Usage, nil
}

func usageSyncError(orgID string, err error) error {
	switch {
	case errors.Is(err, usage.ErrNoBillingAccountCandidate), errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.FailedPrecondition(nil, "organization has no billing account candidate")
	case grpcerrors.Code(err) == codes.FailedPrecondition:
		return grpcerrors.FailedPrecondition(nil, "organization usage setup failed precondition")
	case grpcerrors.Code(err) == codes.ResourceExhausted:
		return grpcerrors.ResourceExhausted(nil, "organization exceeds configured account usage limits")
	default:
		log.Errorf("Error syncing usage for organization %s: %v", orgID, err)
		return grpcerrors.Internal(err, "failed to set up organization usage")
	}
}

func describeUsageAfterSyncError(orgID, resource string, err error) error {
	switch grpcerrors.Code(err) {
	case codes.NotFound, codes.FailedPrecondition:
		return grpcerrors.FailedPrecondition(nil, "organization usage is not configured")
	case codes.ResourceExhausted:
		return grpcerrors.ResourceExhausted(nil, "organization exceeds configured account usage limits")
	default:
		log.Errorf("Error describing usage %s after sync for organization %s: %v", resource, orgID, err)
		return grpcerrors.Internal(err, fmt.Sprintf("failed to describe organization usage %s", resource))
	}
}

func serializeUsageLimits(limits *usagepb.OrganizationLimits) *pb.OrganizationLimits {
	if limits == nil {
		return nil
	}

	return &pb.OrganizationLimits{
		MaxCanvases:            limits.MaxCanvases,
		MaxNodesPerCanvas:      limits.MaxNodesPerCanvas,
		MaxUsers:               limits.MaxUsers,
		RetentionWindowDays:    limits.RetentionWindowDays,
		MaxEventsPerMonth:      limits.MaxEventsPerMonth,
		MaxIntegrations:        limits.MaxIntegrations,
		MaxAgentTokensPerMonth: limits.MaxAgentTokensPerMonth,
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

	var nextEventBucketDecreaseAt *timestamppb.Timestamp
	if orgUsage.NextEventBucketLeakAtUnixSeconds > 0 {
		nextEventBucketDecreaseAt = timestamppb.New(
			time.Unix(orgUsage.NextEventBucketLeakAtUnixSeconds, 0).UTC(),
		)
	}

	var agentTokenBucketLastUpdatedAt *timestamppb.Timestamp
	if orgUsage.AgentTokenBucketLastUpdatedAtUnixSeconds > 0 {
		agentTokenBucketLastUpdatedAt = timestamppb.New(
			time.Unix(orgUsage.AgentTokenBucketLastUpdatedAtUnixSeconds, 0).UTC(),
		)
	}

	var nextAgentTokenBucketDecreaseAt *timestamppb.Timestamp
	if orgUsage.NextAgentTokenBucketLeakAtUnixSeconds > 0 {
		nextAgentTokenBucketDecreaseAt = timestamppb.New(
			time.Unix(orgUsage.NextAgentTokenBucketLeakAtUnixSeconds, 0).UTC(),
		)
	}

	return &pb.OrganizationUsage{
		Canvases:                       orgUsage.Canvases,
		EventBucketLevel:               orgUsage.EventBucketLevel,
		EventBucketCapacity:            orgUsage.EventBucketCapacity,
		EventBucketLastUpdatedAt:       eventBucketLastUpdatedAt,
		NextEventBucketDecreaseAt:      nextEventBucketDecreaseAt,
		AgentTokenBucketLevel:          orgUsage.AgentTokenBucketLevel,
		AgentTokenBucketCapacity:       orgUsage.AgentTokenBucketCapacity,
		AgentTokenBucketLastUpdatedAt:  agentTokenBucketLastUpdatedAt,
		NextAgentTokenBucketDecreaseAt: nextAgentTokenBucketDecreaseAt,
	}
}
