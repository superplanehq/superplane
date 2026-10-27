package models

import (
	"slices"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResolveOrganizationProductFeatures resolves plan and organization grants.
// A disabled grant cannot remove access included in a plan.
func ResolveOrganizationProductFeatures(tx *gorm.DB, orgID uuid.UUID) (map[string]bool, error) {
	plan, err := ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	return plan.ResolveProductFeatures(tx)
}

// ResolveProductFeatures uses the loaded plan to resolve organization grants.
func (p *OrganizationBillingPlan) ResolveProductFeatures(tx *gorm.DB) (map[string]bool, error) {
	definitions, err := ListPlanFeatureDefinitions(tx)
	if err != nil {
		return nil, err
	}
	grants, err := ListOrganizationFeatureGrants(tx, p.OrganizationID)
	if err != nil {
		return nil, err
	}
	enabled := map[string]bool{}
	for _, grant := range grants {
		if grant.Enabled {
			enabled[grant.FeatureKey] = true
		}
	}
	resolved := make(map[string]bool, len(definitions))
	for _, feature := range definitions {
		resolved[feature.Key] = !BillingEnabled() || !feature.Gated || slices.Contains(p.Catalog.Features, feature.Key) || enabled[feature.Key]
	}
	return resolved, nil
}
