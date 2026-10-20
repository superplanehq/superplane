package entitlements

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"slices"
	"time"
)

type CreditPolicy struct {
	AmountCents  int64  `json:"amount_cents"`
	Frequency    string `json:"frequency"`
	Expiration   string `json:"expiration"`
	ValidityDays *int   `json:"validity_days"`
}

type Entitlements struct {
	PlanID                 uuid.UUID       `json:"plan_id"`
	PlanKey                string          `json:"plan_key"`
	PlanName               string          `json:"plan_name"`
	SeatLimit              *int            `json:"seat_limit"`
	ParallelTaskLimit      *int            `json:"parallel_task_limit"`
	Features               map[string]bool `json:"features"`
	CreditPolicy           CreditPolicy    `json:"credit_policy"`
	TrialCreditCents       int64           `json:"trial_credit_cents"`
	TrialLengthDays        int             `json:"trial_length_days"`
	MarkupBPS              int             `json:"markup_bps"`
	HostedExecutionAllowed bool            `json:"hosted_execution_allowed"`
	CreditPurchaseAllowed  bool            `json:"credit_purchase_allowed"`
}

func Resolve(tx *gorm.DB, orgID uuid.UUID) (*Entitlements, error) {
	plan, err := models.ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	catalog := plan.Catalog
	markup, err := models.ResolveOrganizationMarkupBPS(tx, orgID)
	if err != nil {
		return nil, err
	}
	grants, err := models.ListOrganizationFeatureGrants(tx, orgID)
	if err != nil {
		return nil, err
	}
	enabled := map[string]bool{}
	for _, grant := range grants {
		if grant.Enabled {
			enabled[grant.FeatureKey] = true
		}
	}
	result := &Entitlements{
		PlanID: catalog.ID, PlanKey: catalog.Key, PlanName: catalog.Name,
		SeatLimit: catalog.SeatLimit, ParallelTaskLimit: catalog.ParallelTaskLimit,
		Features:         map[string]bool{},
		CreditPolicy:     CreditPolicy{AmountCents: catalog.CreditAmountCents, Frequency: catalog.CreditFrequency, Expiration: catalog.CreditExpiration, ValidityDays: catalog.CreditValidityDays},
		TrialCreditCents: catalog.TrialCreditCents, TrialLengthDays: catalog.TrialLengthDays, MarkupBPS: markup,
		HostedExecutionAllowed: plan.AllowsHostedExecution(time.Now()), CreditPurchaseAllowed: plan.AllowsCreditPurchase(),
	}
	for _, feature := range Definitions {
		result.Features[feature.Key] = !models.BillingEnabled() || !feature.Gated || slices.Contains(catalog.Features, feature.Key) || enabled[feature.Key]
	}
	if !models.BillingEnabled() {
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
