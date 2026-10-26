package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetPastDueGraceDays(tx *gorm.DB) (int, error) {
	implementation, err := Current()
	if err != nil {
		return 0, err
	}
	return implementation.GetPastDueGraceDays(tx)
}

func UpdatePastDueGraceDays(tx *gorm.DB, days int) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.UpdatePastDueGraceDays(tx, days)
}

func ValidatePastDueGraceDays(days int) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.ValidatePastDueGraceDays(days)
}

func Initialize(tx *gorm.DB) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.Initialize(tx)
}

func InitializeOrganization(tx *gorm.DB, organizationID uuid.UUID) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.InitializeOrganization(tx, organizationID)
}

func GrantWelcomeCredit(tx *gorm.DB, organizationID, accountID uuid.UUID) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.GrantWelcomeCredit(tx, organizationID, accountID)
}

func PrepareDeletion(ctx context.Context, organizationID uuid.UUID) (DeletionGuard, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.PrepareDeletion(ctx, organizationID)
}

func LocalDeletionGuard(tx *gorm.DB, organizationID uuid.UUID) (DeletionGuard, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.LocalDeletionGuard(tx, organizationID)
}

func PrepareAdmission(ctx context.Context, organizationID uuid.UUID) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.PrepareAdmission(ctx, organizationID)
}

func ValidateMemberAdmission(tx *gorm.DB, organizationID uuid.UUID) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.ValidateMemberAdmission(tx, organizationID)
}

func ValidateInvitationCapacity(tx *gorm.DB, organizationID uuid.UUID, alreadyPending bool, pendingLimit int) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.ValidateInvitationCapacity(tx, organizationID, alreadyPending, pendingLimit)
}

func ResolveSeats(tx *gorm.DB, organizationID uuid.UUID) (*SeatUsage, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.ResolveSeats(tx, organizationID)
}

func RecheckSeatCapacity(tx *gorm.DB, organizationID uuid.UUID, cause string) (CapacityAlert, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.RecheckSeatCapacity(tx, organizationID, cause)
}

func AssertOrganizationPaymentMethod(tx *gorm.DB, organizationID uuid.UUID) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.AssertOrganizationPaymentMethod(tx, organizationID)
}

func AssertOrganizationPlanAllowsRuns(tx *gorm.DB, organizationID uuid.UUID) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.AssertOrganizationPlanAllowsRuns(tx, organizationID)
}

func AssertHostedRunAllowed(tx *gorm.DB, organizationID uuid.UUID, factoryID *uuid.UUID) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.AssertHostedRunAllowed(tx, organizationID, factoryID)
}

// ResolveParallelTaskLimit returns nil when the effective plan has no runner limit.
func ResolveParallelTaskLimit(tx *gorm.DB, organizationID uuid.UUID) (*int, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.ResolveParallelTaskLimit(tx, organizationID)
}

func ResolveProductFeatures(tx *gorm.DB, organizationID uuid.UUID) (map[string]bool, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.ResolveProductFeatures(tx, organizationID)
}

func ResolveMarkupBPS(tx *gorm.DB, organizationID uuid.UUID) (int, error) {
	implementation, err := Current()
	if err != nil {
		return 0, err
	}
	return implementation.ResolveMarkupBPS(tx, organizationID)
}

func RemainingCreditMicros(tx *gorm.DB, organizationIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.RemainingCreditMicros(tx, organizationIDs)
}

func DescribeCredit(tx *gorm.DB, organizationID uuid.UUID) (CreditSummary, error) {
	implementation, err := Current()
	if err != nil {
		return CreditSummary{}, err
	}
	return implementation.DescribeCredit(tx, organizationID)
}

func DescribeFactoryBudget(tx *gorm.DB, organizationID, factoryID uuid.UUID) (FactoryBudget, error) {
	implementation, err := Current()
	if err != nil {
		return FactoryBudget{}, err
	}
	return implementation.DescribeFactoryBudget(tx, organizationID, factoryID)
}

func PriceUsage(tx *gorm.DB, organizationID uuid.UUID, usage Usage) (UsagePrice, error) {
	implementation, err := Current()
	if err != nil {
		return UsagePrice{}, err
	}
	return implementation.PriceUsage(tx, organizationID, usage)
}

func PriceCompute(usage ComputeUsage) (UsagePrice, error) {
	implementation, err := Current()
	if err != nil {
		return UsagePrice{}, err
	}
	return implementation.PriceCompute(usage)
}

func GetInstallationSettings(tx *gorm.DB) (InstallationSettings, error) {
	implementation, err := Current()
	if err != nil {
		return InstallationSettings{}, err
	}
	return implementation.GetInstallationSettings(tx)
}

func UpdateInstallationSettings(tx *gorm.DB, settings InstallationSettings) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.UpdateInstallationSettings(tx, settings)
}

func SummarizeUsage(tx *gorm.DB, filter UsageReportFilter) (UsageTotals, []UsageByModel, error) {
	implementation, err := Current()
	if err != nil {
		return UsageTotals{}, nil, err
	}
	return implementation.SummarizeUsage(tx, filter)
}

func SummarizeComputeUsage(tx *gorm.DB, filter UsageReportFilter) (UsageTotals, []UsageByMachineType, error) {
	implementation, err := Current()
	if err != nil {
		return UsageTotals{}, nil, err
	}
	return implementation.SummarizeComputeUsage(tx, filter)
}

func ListWorkOrderRunUsage(tx *gorm.DB, filter UsageReportFilter, limit, offset int) ([]WorkOrderRunUsage, int64, error) {
	implementation, err := Current()
	if err != nil {
		return nil, 0, err
	}
	return implementation.ListWorkOrderRunUsage(tx, filter, limit, offset)
}

func ListAllWorkOrderRunUsage(tx *gorm.DB, filter UsageReportFilter) ([]WorkOrderRunUsage, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.ListAllWorkOrderRunUsage(tx, filter)
}

func SummarizeUsageByLocalDay(tx *gorm.DB, filter UsageReportFilter, dayBounds []time.Time) ([]UsageSplit, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.SummarizeUsageByLocalDay(tx, filter, dayBounds)
}

func JoinLimit(tx *gorm.DB, organizationID uuid.UUID) (*int, error) {
	implementation, err := Current()
	if err != nil {
		return nil, err
	}
	return implementation.JoinLimit(tx, organizationID)
}

func ValidateSpendingReportWindow(since, until time.Time) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.ValidateSpendingReportWindow(since, until)
}

func LookupRate(provider, model string) (Rate, bool) {
	return Require().LookupRate(provider, model)
}

func UpdateFactoryHostedBudget(tx *gorm.DB, organizationID, factoryID uuid.UUID, budgetCents *int64) error {
	implementation, err := Current()
	if err != nil {
		return err
	}
	return implementation.UpdateFactoryHostedBudget(tx, organizationID, factoryID, budgetCents)
}
