package organizations

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func DescribeUsage(ctx context.Context, usageService usage.Service, orgID string) (*pb.DescribeUsageResponse, error) {
	if !usageService.Enabled() {
		return &pb.DescribeUsageResponse{
			Enabled:       false,
			StatusMessage: "Usage tracking is disabled for this SuperPlane instance.",
		}, nil
	}

	limits, err := describeUsageLimits(ctx, usageService, orgID)
	if err != nil {
		return nil, err
	}

	orgUsage, err := describeUsageMetrics(ctx, usageService, orgID)
	if err != nil {
		return nil, err
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

	return setupUsageOrganization(ctx, usageService, orgID)
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

	if _, setupErr := setupUsageOrganization(ctx, usageService, orgID); setupErr != nil {
		return nil, setupErr
	}

	response, err = usageService.DescribeOrganizationUsage(ctx, orgID)
	if err != nil {
		log.Errorf("Error describing usage metrics after setup for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to describe organization usage")
	}

	return response.Usage, nil
}

func setupUsageOrganization(
	ctx context.Context,
	usageService usage.Service,
	orgID string,
) (*usagepb.OrganizationLimits, error) {
	billingUser, err := models.FindFirstHumanUserByOrganization(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.FailedPrecondition, "organization has no billing account candidate")
		}

		log.Errorf("Error finding billing account candidate for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to determine organization billing account")
	}

	if billingUser.AccountID == nil {
		return nil, status.Error(codes.FailedPrecondition, "organization has no billing account candidate")
	}

	accountID := billingUser.AccountID.String()

	if _, err := usageService.SetupAccount(ctx, accountID); err != nil && status.Code(err) != codes.AlreadyExists {
		log.Errorf("Error setting up usage account %s: %v", accountID, err)
		return nil, status.Error(codes.Internal, "failed to set up usage account")
	}

	response, err := usageService.SetupOrganization(ctx, orgID, accountID)
	if err == nil {
		return response.Limits, nil
	}

	if status.Code(err) == codes.FailedPrecondition {
		describeResponse, describeErr := usageService.DescribeOrganizationLimits(ctx, orgID)
		if describeErr == nil {
			return describeResponse.Limits, nil
		}
	}

	if status.Code(err) == codes.ResourceExhausted {
		return nil, status.Error(codes.ResourceExhausted, "organization exceeds configured account usage limits")
	}

	log.Errorf("Error setting up usage organization %s for account %s: %v", orgID, accountID, err)
	return nil, status.Error(codes.Internal, "failed to set up organization usage")
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

	return &pb.OrganizationUsage{
		Canvases:            orgUsage.Canvases,
		EventBucketLevel:    orgUsage.EventBucketLevel,
		EventBucketCapacity: orgUsage.EventBucketCapacity,
		EventBucketLastUpdatedAt: timestamppb.New(
			time.Unix(orgUsage.EventBucketLastUpdatedAtUnixSeconds, 0).UTC(),
		),
	}
}
