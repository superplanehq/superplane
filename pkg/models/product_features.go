package models

import (
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/gorm"
)

var ErrBYOKAccessDenied = errors.New("your organization does not have access to provider credentials; select a SuperPlane-hosted model or contact your administrator")

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

// OrganizationBYOKAccess requires both the rollout and product permission.
// Custom providers also require their own rollout.
func OrganizationBYOKAccess(tx *gorm.DB, orgID uuid.UUID, provider string) (bool, error) {
	required := []string{features.FeatureOrganizationBYOK}
	if provider == UsageProviderCustom {
		required = append(required, features.FeatureOrganizationBYOKCustomProvider)
	}
	enabled, err := OrganizationHasExperimentalFeatures(tx, orgID, required...)
	if err != nil || !enabled {
		return false, err
	}
	productFeatures, err := ResolveOrganizationProductFeatures(tx, orgID)
	if err != nil {
		return false, err
	}
	return productFeatures["byo_key"], nil
}

func RequireOrganizationBYOKAccess(tx *gorm.DB, orgID uuid.UUID, provider string) error {
	allowed, err := OrganizationBYOKAccess(tx, orgID, provider)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrBYOKAccessDenied
	}
	return nil
}
