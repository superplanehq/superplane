package usage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

var ErrNoBillingAccountCandidate = errors.New("organization has no billing account candidate")

func SyncOrganization(ctx context.Context, usageService Service, orgID string) error {
	return syncOrganization(ctx, usageService, orgID, false)
}

func SyncOrganizationForce(ctx context.Context, usageService Service, orgID string) error {
	return syncOrganization(ctx, usageService, orgID, true)
}

func MarkOrganizationSyncedIfUnset(orgID string) error {
	return models.MarkOrganizationUsageSyncedIfUnset(orgID, time.Now())
}

func markOrganizationUsageSynced(orgID string, readStartedAt, syncedAt time.Time, limits *pb.OrganizationLimits) error {
	if limits == nil {
		return models.MarkOrganizationUsageSynced(orgID, syncedAt)
	}

	retentionWindowDays := limits.RetentionWindowDays
	return models.MarkOrganizationUsageSyncedWithLimitsIfNoNewerThan(
		orgID,
		syncedAt,
		&retentionWindowDays,
		readStartedAt,
		syncedAt,
	)
}

func syncOrganization(ctx context.Context, usageService Service, orgID string, force bool) error {
	if usageService == nil || !usageService.Enabled() {
		return ErrUsageDisabled
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return fmt.Errorf("find organization %s: %w", orgID, err)
	}

	if organization.UsageSyncedAt != nil && !force {
		return nil
	}

	accountID, err := resolveOrganizationBillingAccountID(orgID)
	if err != nil {
		return err
	}

	if _, err := usageService.SetupAccount(ctx, accountID); err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("set up usage account %s: %w", accountID, err)
	}

	setupStartedAt := time.Now()
	response, err := usageService.SetupOrganization(ctx, orgID, accountID)
	if err == nil {
		syncedAt := time.Now()
		if response.GetLimits() != nil {
			return markOrganizationUsageSynced(orgID, setupStartedAt, syncedAt, response.GetLimits())
		}

		describeStartedAt := time.Now()
		describeResponse, describeErr := usageService.DescribeOrganizationLimits(ctx, orgID)
		if describeErr == nil {
			return markOrganizationUsageSynced(orgID, describeStartedAt, syncedAt, describeResponse.GetLimits())
		}

		return models.MarkOrganizationUsageSynced(orgID, syncedAt)
	}

	switch status.Code(err) {
	case codes.AlreadyExists:
		describeStartedAt := time.Now()
		describeResponse, describeErr := usageService.DescribeOrganizationLimits(ctx, orgID)
		if describeErr == nil {
			return markOrganizationUsageSynced(orgID, describeStartedAt, time.Now(), describeResponse.GetLimits())
		}

		return models.MarkOrganizationUsageSynced(orgID, time.Now())
	case codes.FailedPrecondition:
		describeStartedAt := time.Now()
		describeResponse, describeErr := usageService.DescribeOrganizationLimits(ctx, orgID)
		if describeErr == nil {
			return markOrganizationUsageSynced(orgID, describeStartedAt, time.Now(), describeResponse.GetLimits())
		}
	case codes.ResourceExhausted:
		describeStartedAt := time.Now()
		describeResponse, describeErr := usageService.DescribeOrganizationLimits(ctx, orgID)
		if describeErr == nil {
			return markOrganizationUsageSynced(orgID, describeStartedAt, time.Now(), describeResponse.GetLimits())
		}

		return err
	}

	return fmt.Errorf("set up usage organization %s: %w", orgID, err)
}

func resolveOrganizationBillingAccountID(orgID string) (string, error) {
	billingUser, err := models.FindFirstHumanUserByOrganization(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNoBillingAccountCandidate
		}

		return "", fmt.Errorf("find billing account candidate for organization %s: %w", orgID, err)
	}

	if billingUser.AccountID == nil {
		return "", ErrNoBillingAccountCandidate
	}

	return billingUser.AccountID.String(), nil
}
