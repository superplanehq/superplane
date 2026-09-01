package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FactoryHostedBudgetSummary is remaining hosted credit for one workspace.
type FactoryHostedBudgetSummary struct {
	BudgetCents     *int64
	BilledMicros    int64
	RemainingMicros int64
	Warning         bool
	Capped          bool
}

func SumFactoryHostedBilledMicros(tx *gorm.DB, orgID, factoryID uuid.UUID) (int64, error) {
	var billedMicros int64
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("COALESCE(SUM(cost_micros), 0)").
		Where("organization_id = ? AND factory_id = ? AND funding_source = ? AND usage_kind = ?", orgID, factoryID, UsageFundingSourceHosted, UsageKindModel).
		Scan(&billedMicros).Error
	return billedMicros, err
}

func DescribeFactoryHostedBudget(tx *gorm.DB, factory *Factory) (FactoryHostedBudgetSummary, error) {
	orgCredit, err := DescribeOrganizationLLMCredit(tx, factory.OrganizationID)
	if err != nil {
		return FactoryHostedBudgetSummary{}, err
	}

	billedMicros, err := SumFactoryHostedBilledMicros(tx, factory.OrganizationID, factory.ID)
	if err != nil {
		return FactoryHostedBudgetSummary{}, err
	}

	if factory.HostedSpendBudgetCents == nil {
		return FactoryHostedBudgetSummary{
			BilledMicros:    billedMicros,
			RemainingMicros: orgCredit.RemainingMicros,
			Warning:         orgCredit.Warning,
			Capped:          false,
		}, nil
	}

	capMicros := CentsToMicros(*factory.HostedSpendBudgetCents)
	factoryRemaining := capMicros - billedMicros
	if factoryRemaining < 0 {
		factoryRemaining = 0
	}

	remaining := factoryRemaining
	if orgCredit.RemainingMicros < remaining {
		remaining = orgCredit.RemainingMicros
	}

	installation, err := GetInstallationLLMSettings(tx)
	if err != nil {
		return FactoryHostedBudgetSummary{}, err
	}

	warning := remaining <= 0
	if capMicros > 0 {
		threshold := capMicros * int64(installation.WarningThresholdBPS) / int64(MarkupBaseBPS)
		warning = remaining <= threshold
	}

	budgetCents := *factory.HostedSpendBudgetCents
	return FactoryHostedBudgetSummary{
		BudgetCents:     &budgetCents,
		BilledMicros:    billedMicros,
		RemainingMicros: remaining,
		Warning:         warning,
		Capped:          true,
	}, nil
}
