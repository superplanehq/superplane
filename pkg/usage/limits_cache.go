package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CacheOrganizationLimits(orgID string, limits *pb.OrganizationLimits, readStartedAt time.Time, syncedAt time.Time) error {
	var retentionWindowDays *int32
	if limits != nil {
		retentionWindowDays = &limits.RetentionWindowDays
	}

	applied, err := models.MarkOrganizationUsageLimitsSyncedIfNoNewerThan(
		orgID,
		retentionWindowDays,
		readStartedAt,
		syncedAt,
	)
	if err != nil {
		return fmt.Errorf("persist usage limits cache for organization %s: %w", orgID, err)
	}

	if !applied {
		return nil
	}

	return nil
}

func RefreshOrganizationLimitsCache(ctx context.Context, usageService Service, orgID string) (*pb.OrganizationLimits, error) {
	if usageService == nil || !usageService.Enabled() {
		return nil, ErrUsageDisabled
	}

	readStartedAt := time.Now()
	response, err := usageService.DescribeOrganizationLimits(ctx, orgID)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return nil, err
		}

		if err := SyncOrganizationForce(ctx, usageService, orgID); err != nil {
			return nil, err
		}

		response, err = usageService.DescribeOrganizationLimits(ctx, orgID)
		if err != nil {
			return nil, err
		}
	}

	var limits *pb.OrganizationLimits
	if response != nil {
		limits = response.Limits
	}

	if err := CacheOrganizationLimits(orgID, limits, readStartedAt, time.Now()); err != nil {
		return nil, err
	}

	return limits, nil
}
