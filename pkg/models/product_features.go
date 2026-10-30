package models

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/gorm"
)

var ErrBYOKAccessDenied = errors.New("your organization does not have access to provider credentials; select a SuperPlane-hosted model or contact your administrator")

// OrganizationBYOKAccess allows standard providers when no hosted default exists.
// Otherwise, rollout and product permissions apply. Custom providers need both rollouts.
func OrganizationBYOKAccess(tx *gorm.DB, orgID uuid.UUID, provider string) (bool, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != UsageProviderCustom {
		defaultModel, err := GetInstallationDefaultHostedLLMModel(tx)
		if err != nil {
			return false, err
		}
		if !defaultModel.IsSet() {
			return true, nil
		}
	}

	required := []string{features.FeatureOrganizationBYOK}
	if provider == UsageProviderCustom {
		required = append(required, features.FeatureOrganizationBYOKCustomProvider)
	}
	enabled, err := OrganizationHasExperimentalFeatures(tx, orgID, required...)
	if err != nil || !enabled {
		return false, err
	}
	productFeatures, err := billing.ResolveProductFeatures(tx, orgID)
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
