package usage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
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

	_, err = usageService.SetupOrganization(ctx, orgID, accountID)
	if err == nil {
		return models.MarkOrganizationUsageSynced(orgID, time.Now())
	}

	switch status.Code(err) {
	case codes.AlreadyExists:
		return models.MarkOrganizationUsageSynced(orgID, time.Now())
	case codes.FailedPrecondition:
		if _, describeErr := usageService.DescribeOrganizationLimits(ctx, orgID); describeErr == nil {
			return models.MarkOrganizationUsageSynced(orgID, time.Now())
		}
	case codes.ResourceExhausted:
		if _, describeErr := usageService.DescribeOrganizationLimits(ctx, orgID); describeErr == nil {
			return models.MarkOrganizationUsageSynced(orgID, time.Now())
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
