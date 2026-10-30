package entitlements

import (
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"gorm.io/gorm"
)

type CreditPolicy struct {
	AmountCents  int64  `json:"amount_cents"`
	Frequency    string `json:"frequency"`
	Expiration   string `json:"expiration"`
	ValidityDays *int   `json:"validity_days"`
}

type Entitlements struct {
	PlanID                 uuid.UUID                                    `json:"plan_id"`
	PlanKey                string                                       `json:"plan_key"`
	PlanName               string                                       `json:"plan_name"`
	PlanSeatLimit          *int                                         `json:"plan_limit"`
	SeatLimitOverride      *billingmodels.OrganizationSeatLimitOverride `json:"override"`
	SeatLimit              *int                                         `json:"seat_limit"`
	ParallelTaskLimit      *int                                         `json:"parallel_task_limit"`
	Features               map[string]bool                              `json:"features"`
	CreditPolicy           CreditPolicy                                 `json:"credit_policy"`
	TrialCreditCents       int64                                        `json:"trial_credit_cents"`
	TrialLengthDays        int                                          `json:"trial_length_days"`
	MarkupBPS              int                                          `json:"markup_bps"`
	HostedExecutionAllowed bool                                         `json:"hosted_execution_allowed"`
	CreditPurchaseAllowed  bool                                         `json:"credit_purchase_allowed"`
}

func Resolve(tx *gorm.DB, orgID uuid.UUID) (*Entitlements, error) {
	plan, err := billingmodels.ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	catalog := plan.Catalog
	override, err := billingmodels.FindOrganizationSeatLimitOverride(tx, orgID)
	if err != nil {
		return nil, err
	}
	markup, err := billingmodels.ResolveOrganizationMarkupBPS(tx, orgID)
	if err != nil {
		return nil, err
	}
	productFeatures, err := plan.ResolveProductFeatures(tx)
	if err != nil {
		return nil, err
	}
	result := &Entitlements{
		PlanID: catalog.ID, PlanKey: catalog.Key, PlanName: catalog.Name,
		PlanSeatLimit: catalog.SeatLimit, SeatLimitOverride: override, SeatLimit: SeatMaximum(catalog.SeatLimit, override), ParallelTaskLimit: catalog.ParallelTaskLimit,
		Features:         productFeatures,
		CreditPolicy:     CreditPolicy{AmountCents: catalog.CreditAmountCents, Frequency: catalog.CreditFrequency, Expiration: catalog.CreditExpiration, ValidityDays: catalog.CreditValidityDays},
		TrialCreditCents: catalog.TrialCreditCents, TrialLengthDays: catalog.TrialLengthDays, MarkupBPS: markup,
		HostedExecutionAllowed: plan.AllowsHostedExecution(time.Now()), CreditPurchaseAllowed: plan.AllowsCreditPurchase(),
	}
	if !billingmodels.BillingEnabled() {
		result.SeatLimit = nil
		result.ParallelTaskLimit = nil
		result.HostedExecutionAllowed = true
		result.CreditPurchaseAllowed = true
	}
	return result, nil
}

func HasFeature(tx *gorm.DB, orgID uuid.UUID, key string) (bool, error) {
	resolved, err := Resolve(tx, orgID)
	if err != nil {
		return false, err
	}
	return resolved.Features[key], nil
}
