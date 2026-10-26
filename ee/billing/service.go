// Package billing implements accounting independently of its HTTP and RPC
// transports. The server supplies the provider implementation at startup.
package billing

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/pricebook"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type Service struct{ provider billing.Provider }

func New(provider billing.Provider) *Service { return &Service{provider: provider} }

var _ billing.Runtime = (*Service)(nil)

func (*Service) Initialize(db *gorm.DB) error {
	if err := billingmodels.BindLegacyBillingProduct(db, os.Getenv("POLAR_BUSINESS_PRODUCT_ID")); err != nil {
		return err
	}
	if err := billingmodels.LoadCurrentPriceBook(db); err != nil {
		log.WithError(err).Warn("failed to load usage price book; using in-memory rates")
	}
	return nil
}

func (*Service) InitializeOrganization(tx *gorm.DB, organizationID uuid.UUID) error {
	if !billing.Enabled() {
		return nil
	}
	_, err := billingmodels.EnsureOrganizationBillingPlan(tx, organizationID)
	return err
}

func (*Service) GrantWelcomeCredit(tx *gorm.DB, organizationID, accountID uuid.UUID) error {
	if !billing.Enabled() {
		return nil
	}
	return billingmodels.GrantWelcomeCredit(tx, organizationID, accountID)
}

func (*Service) LocalDeletionGuard(tx *gorm.DB, organizationID uuid.UUID) (billing.DeletionGuard, error) {
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, organizationID)
	if err != nil {
		return nil, err
	}
	return billingmodels.NewLocalDeletionClearance(organizationID, plan)
}

func (s *Service) PrepareDeletion(ctx context.Context, organizationID uuid.UUID) (billing.DeletionGuard, error) {
	if s.provider == nil {
		return nil, billing.ErrOrganizationSubscriptionUnverified
	}
	return s.provider.PrepareDeletion(ctx, organizationID)
}
func (s *Service) PrepareAdmission(ctx context.Context, organizationID uuid.UUID) error {
	if s.provider == nil {
		return billing.ErrImplementationMissing
	}
	return s.provider.PrepareAdmission(ctx, organizationID)
}
func (s *Service) ValidateMemberAdmission(tx *gorm.DB, organizationID uuid.UUID) error {
	if s.provider == nil {
		return billing.ErrImplementationMissing
	}
	return s.provider.ValidateMemberAdmission(tx, organizationID)
}
func (s *Service) ValidateInvitationCapacity(tx *gorm.DB, organizationID uuid.UUID, alreadyPending bool, pendingLimit int) error {
	if s.provider == nil {
		return billing.ErrImplementationMissing
	}
	return s.provider.ValidateInvitationCapacity(tx, organizationID, alreadyPending, pendingLimit)
}
func (*Service) ResolveSeats(tx *gorm.DB, organizationID uuid.UUID) (*billing.SeatUsage, error) {
	return entitlements.ResolveSeats(tx, organizationID)
}
func (s *Service) RecheckSeatCapacity(tx *gorm.DB, organizationID uuid.UUID, cause string) (billing.CapacityAlert, error) {
	if s.provider == nil {
		return nil, billing.ErrImplementationMissing
	}
	return s.provider.RecheckSeatCapacity(tx, organizationID, cause)
}
func (s *Service) AssertOrganizationPaymentMethod(tx *gorm.DB, organizationID uuid.UUID) error {
	if s.provider == nil {
		return billing.ErrImplementationMissing
	}
	return s.provider.AssertOrganizationPaymentMethod(tx, organizationID)
}
func (*Service) AssertOrganizationPlanAllowsRuns(tx *gorm.DB, organizationID uuid.UUID) error {
	return billingmodels.AssertOrganizationPlanAllowsRuns(tx, organizationID)
}
func (*Service) AssertHostedRunAllowed(tx *gorm.DB, organizationID uuid.UUID, factoryID *uuid.UUID) error {
	return billingmodels.AssertHostedRunAllowed(tx, organizationID, factoryID)
}
func (*Service) ResolveParallelTaskLimit(tx *gorm.DB, organizationID uuid.UUID) (*int, error) {
	plan, err := billingmodels.ResolveOrganizationBillingPlan(tx, organizationID)
	if err != nil {
		return nil, err
	}
	return plan.Catalog.ParallelTaskLimit, nil
}
func (*Service) ResolveProductFeatures(tx *gorm.DB, organizationID uuid.UUID) (map[string]bool, error) {
	return billingmodels.ResolveOrganizationProductFeatures(tx, organizationID)
}
func (*Service) ResolveMarkupBPS(tx *gorm.DB, organizationID uuid.UUID) (int, error) {
	return billingmodels.ResolveOrganizationMarkupBPS(tx, organizationID)
}
func (*Service) RemainingCreditMicros(tx *gorm.DB, organizationIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	return billingmodels.RemainingHostedCreditMicros(tx, organizationIDs)
}
func (*Service) DescribeCredit(tx *gorm.DB, organizationID uuid.UUID) (billing.CreditSummary, error) {
	summary, err := billingmodels.DescribeOrganizationLLMCredit(tx, organizationID)
	return billing.CreditSummary(summary), err
}
func (*Service) DescribeFactoryBudget(tx *gorm.DB, organizationID, factoryID uuid.UUID) (billing.FactoryBudget, error) {
	factory, err := models.FindFactory(tx, organizationID, factoryID)
	if err != nil {
		return billing.FactoryBudget{}, err
	}
	summary, err := billingmodels.DescribeFactoryHostedBudget(tx, factory)
	return billing.FactoryBudget(summary), err
}
func (*Service) LookupRate(provider, model string) (billing.Rate, bool) {
	return pricebook.Lookup(provider, model)
}
func (*Service) PriceUsage(tx *gorm.DB, organizationID uuid.UUID, in billing.Usage) (billing.UsagePrice, error) {
	version := pricebook.Version
	providerMicros := int64(0)
	if in.CostMicros != nil {
		providerMicros = *in.CostMicros
		version += "+provider"
	} else {
		if strings.TrimSpace(in.FundingSource) == models.UsageFundingSourceHosted && !pricebook.IsPriced(in.Provider, in.Model) {
			log.WithFields(log.Fields{"provider": in.Provider, "model": in.Model}).Warn("hosted LLM usage has no price for this model; recording tokens at zero cost")
		}
		providerMicros = pricebook.EstimateMicros(in.Provider, in.Model, in.InputTokens, in.OutputTokens, in.CacheReadTokens, in.CacheWriteTokens, in.ReasoningTokens)
	}
	billed := providerMicros
	if in.FundingSource == models.UsageFundingSourceHosted {
		markup, err := billingmodels.ResolveOrganizationMarkupBPS(tx, organizationID)
		if err != nil {
			return billing.UsagePrice{}, err
		}
		billed = billingmodels.ApplyMarkupMicros(providerMicros, markup)
	}
	return billing.UsagePrice{ProviderCostMicros: providerMicros, CostMicros: billed, PriceBookVersion: version}, nil
}
func (*Service) PriceCompute(in billing.ComputeUsage) (billing.UsagePrice, error) {
	micros := pricebook.EstimateComputeMicros(in.MachineType, in.FleetID, in.DurationSeconds)
	return billing.UsagePrice{ProviderCostMicros: micros, CostMicros: micros, PriceBookVersion: pricebook.Version}, nil
}
func (*Service) GetInstallationSettings(tx *gorm.DB) (billing.InstallationSettings, error) {
	settings, err := billingmodels.GetInstallationLLMSettings(tx)
	if err != nil {
		return billing.InstallationSettings{}, err
	}
	trial, err := billingmodels.FindBillingPlan(tx, billingmodels.BillingPlanTrial)
	if err != nil {
		return billing.InstallationSettings{}, err
	}
	return billing.InstallationSettings{TrialCreditCents: trial.TrialCreditCents, MarkupBPS: settings.MarkupBPS,
		WarningThresholdBPS: settings.WarningThresholdBPS, CreditPurchaseFeeBPS: settings.CreditPurchaseFeeBPS,
		CreditPurchaseFixedFeeCents: settings.CreditPurchaseFixedFeeCents}, nil
}
func (*Service) UpdateInstallationSettings(tx *gorm.DB, settings billing.InstallationSettings) error {
	_, err := billingmodels.UpdateInstallationLLMSettings(tx, billingmodels.InstallationLLMSettings{
		WelcomeGrantCents: settings.TrialCreditCents, MarkupBPS: settings.MarkupBPS,
		WarningThresholdBPS: settings.WarningThresholdBPS, CreditPurchaseFeeBPS: settings.CreditPurchaseFeeBPS,
		CreditPurchaseFixedFeeCents: settings.CreditPurchaseFixedFeeCents})
	return err
}
func (*Service) GetPastDueGraceDays(tx *gorm.DB) (int, error) {
	return billingmodels.GetPastDueGraceDays(tx)
}
func (*Service) UpdatePastDueGraceDays(tx *gorm.DB, days int) error {
	return billingmodels.UpdatePastDueGraceDays(tx, days)
}
func (*Service) ValidatePastDueGraceDays(days int) error {
	return billingmodels.ValidatePastDueGraceDays(days)
}
func (*Service) SummarizeUsage(tx *gorm.DB, filter billing.UsageReportFilter) (billing.UsageTotals, []billing.UsageByModel, error) {
	return billingmodels.SummarizeUsage(tx, filter)
}
func (*Service) SummarizeComputeUsage(tx *gorm.DB, filter billing.UsageReportFilter) (billing.UsageTotals, []billing.UsageByMachineType, error) {
	return billingmodels.SummarizeComputeUsage(tx, filter)
}
func (*Service) ListWorkOrderRunUsage(tx *gorm.DB, filter billing.UsageReportFilter, limit, offset int) ([]billing.WorkOrderRunUsage, int64, error) {
	return billingmodels.ListWorkOrderRunUsage(tx, filter, limit, offset)
}
func (*Service) ListAllWorkOrderRunUsage(tx *gorm.DB, filter billing.UsageReportFilter) ([]billing.WorkOrderRunUsage, error) {
	return billingmodels.ListAllWorkOrderRunUsage(tx, filter)
}
func (*Service) SummarizeUsageByLocalDay(tx *gorm.DB, filter billing.UsageReportFilter, bounds []time.Time) ([]billing.UsageSplit, error) {
	return billingmodels.SummarizeUsageByLocalDay(tx, filter, bounds)
}

func (*Service) JoinLimit(tx *gorm.DB, organizationID uuid.UUID) (*int, error) {
	seats, err := entitlements.ResolveSeats(tx, organizationID)
	if err != nil {
		return nil, err
	}
	return entitlements.JoinLimit(seats), nil
}

func (*Service) ValidateSpendingReportWindow(since, until time.Time) error {
	return billingmodels.ValidateSpendingReportWindow(since, until)
}

func (*Service) UpdateFactoryHostedBudget(tx *gorm.DB, organizationID, factoryID uuid.UUID, budgetCents *int64) error {
	return billingmodels.UpdateFactoryHostedBudget(tx, organizationID, factoryID, budgetCents)
}
