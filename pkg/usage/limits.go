package usage

import (
	"os"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const DefaultProfileName = "basic"

type EffectiveLimits struct {
	ProfileName           string
	MaxOrgsPerAccount     int
	MaxCanvasesPerOrg     int
	MaxNodesPerCanvas     int
	MaxUsersPerOrg        int
	MaxIntegrationsPerOrg int
	MaxEventsPerMonth     int
	RetentionDays         int
	IsUnlimited           bool
	HasOverrides          bool
}

func isEnforcementDisabled() bool {
	env := os.Getenv("APP_ENV")
	if env == "development" || env == "test" {
		return os.Getenv("USAGE_LIMITS_ENABLED") != "true"
	}
	return false
}

func getDefaultProfileName() string {
	name := os.Getenv("USAGE_DEFAULT_PROFILE")
	if name != "" {
		return name
	}
	return DefaultProfileName
}

func ResolveEffectiveLimits(orgID uuid.UUID) (*EffectiveLimits, error) {
	return ResolveEffectiveLimitsInTransaction(database.Conn(), orgID)
}

func ResolveEffectiveLimitsInTransaction(tx *gorm.DB, orgID uuid.UUID) (*EffectiveLimits, error) {
	if isEnforcementDisabled() {
		return &EffectiveLimits{IsUnlimited: true}, nil
	}

	override, err := models.FindOrgUsageOverrideInTransaction(tx, orgID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if override != nil && override.IsUnlimited {
		return &EffectiveLimits{
			IsUnlimited:  true,
			HasOverrides: true,
		}, nil
	}

	org, err := models.FindOrganizationByIDInTransaction(tx, orgID.String())
	if err != nil {
		return nil, err
	}

	var profile *models.UsageProfile
	if org.UsageProfileID != nil {
		profile, err = models.FindUsageProfileByIDInTransaction(tx, *org.UsageProfileID)
		if err != nil {
			return nil, err
		}
	} else {
		profile, err = models.FindUsageProfileByNameInTransaction(tx, getDefaultProfileName())
		if err != nil {
			return nil, err
		}
	}

	limits := &EffectiveLimits{
		ProfileName:           profile.Name,
		MaxOrgsPerAccount:     profile.MaxOrgsPerAccount,
		MaxCanvasesPerOrg:     profile.MaxCanvasesPerOrg,
		MaxNodesPerCanvas:     profile.MaxNodesPerCanvas,
		MaxUsersPerOrg:        profile.MaxUsersPerOrg,
		MaxIntegrationsPerOrg: profile.MaxIntegrationsPerOrg,
		MaxEventsPerMonth:     profile.MaxEventsPerMonth,
		RetentionDays:         profile.RetentionDays,
	}

	if override != nil {
		limits.HasOverrides = true
		applyOverrides(limits, override)
	}

	return limits, nil
}

func applyOverrides(limits *EffectiveLimits, override *models.OrgUsageOverride) {
	if override.MaxOrgsPerAccount != nil {
		limits.MaxOrgsPerAccount = *override.MaxOrgsPerAccount
	}
	if override.MaxCanvasesPerOrg != nil {
		limits.MaxCanvasesPerOrg = *override.MaxCanvasesPerOrg
	}
	if override.MaxNodesPerCanvas != nil {
		limits.MaxNodesPerCanvas = *override.MaxNodesPerCanvas
	}
	if override.MaxUsersPerOrg != nil {
		limits.MaxUsersPerOrg = *override.MaxUsersPerOrg
	}
	if override.MaxIntegrationsPerOrg != nil {
		limits.MaxIntegrationsPerOrg = *override.MaxIntegrationsPerOrg
	}
	if override.MaxEventsPerMonth != nil {
		limits.MaxEventsPerMonth = *override.MaxEventsPerMonth
	}
	if override.RetentionDays != nil {
		limits.RetentionDays = *override.RetentionDays
	}
}
