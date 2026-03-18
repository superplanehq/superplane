package usage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestApplyOverrides(t *testing.T) {
	t.Run("applies all override fields", func(t *testing.T) {
		limits := &EffectiveLimits{
			ProfileName:           "basic",
			MaxOrgsPerAccount:     1,
			MaxCanvasesPerOrg:     3,
			MaxNodesPerCanvas:     50,
			MaxUsersPerOrg:        3,
			MaxIntegrationsPerOrg: 5,
			MaxEventsPerMonth:     10000,
			RetentionDays:         14,
		}

		maxOrgs := 5
		maxCanvases := 10
		maxNodes := 100
		maxUsers := 20
		maxIntegrations := 15
		maxEvents := 50000
		retention := 30

		override := &models.OrgUsageOverride{
			MaxOrgsPerAccount:     &maxOrgs,
			MaxCanvasesPerOrg:     &maxCanvases,
			MaxNodesPerCanvas:     &maxNodes,
			MaxUsersPerOrg:        &maxUsers,
			MaxIntegrationsPerOrg: &maxIntegrations,
			MaxEventsPerMonth:     &maxEvents,
			RetentionDays:         &retention,
		}

		applyOverrides(limits, override)

		assert.Equal(t, 5, limits.MaxOrgsPerAccount)
		assert.Equal(t, 10, limits.MaxCanvasesPerOrg)
		assert.Equal(t, 100, limits.MaxNodesPerCanvas)
		assert.Equal(t, 20, limits.MaxUsersPerOrg)
		assert.Equal(t, 15, limits.MaxIntegrationsPerOrg)
		assert.Equal(t, 50000, limits.MaxEventsPerMonth)
		assert.Equal(t, 30, limits.RetentionDays)
	})

	t.Run("applies partial overrides", func(t *testing.T) {
		limits := &EffectiveLimits{
			ProfileName:           "basic",
			MaxOrgsPerAccount:     1,
			MaxCanvasesPerOrg:     3,
			MaxNodesPerCanvas:     50,
			MaxUsersPerOrg:        3,
			MaxIntegrationsPerOrg: 5,
			MaxEventsPerMonth:     10000,
			RetentionDays:         14,
		}

		maxCanvases := 10

		override := &models.OrgUsageOverride{
			MaxCanvasesPerOrg: &maxCanvases,
		}

		applyOverrides(limits, override)

		assert.Equal(t, 1, limits.MaxOrgsPerAccount)
		assert.Equal(t, 10, limits.MaxCanvasesPerOrg)
		assert.Equal(t, 50, limits.MaxNodesPerCanvas)
		assert.Equal(t, 3, limits.MaxUsersPerOrg)
		assert.Equal(t, 5, limits.MaxIntegrationsPerOrg)
		assert.Equal(t, 10000, limits.MaxEventsPerMonth)
		assert.Equal(t, 14, limits.RetentionDays)
	})

	t.Run("no-op with nil override fields", func(t *testing.T) {
		limits := &EffectiveLimits{
			MaxOrgsPerAccount:     1,
			MaxCanvasesPerOrg:     3,
			MaxNodesPerCanvas:     50,
			MaxUsersPerOrg:        3,
			MaxIntegrationsPerOrg: 5,
			MaxEventsPerMonth:     10000,
			RetentionDays:         14,
		}

		override := &models.OrgUsageOverride{}

		applyOverrides(limits, override)

		assert.Equal(t, 1, limits.MaxOrgsPerAccount)
		assert.Equal(t, 3, limits.MaxCanvasesPerOrg)
		assert.Equal(t, 50, limits.MaxNodesPerCanvas)
		assert.Equal(t, 3, limits.MaxUsersPerOrg)
		assert.Equal(t, 5, limits.MaxIntegrationsPerOrg)
		assert.Equal(t, 10000, limits.MaxEventsPerMonth)
		assert.Equal(t, 14, limits.RetentionDays)
	})
}

func TestIsEnforcementDisabled(t *testing.T) {
	t.Run("disabled in dev without env var", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv("USAGE_LIMITS_ENABLED", "")
		assert.True(t, isEnforcementDisabled())
	})

	t.Run("disabled in test without env var", func(t *testing.T) {
		t.Setenv("APP_ENV", "test")
		t.Setenv("USAGE_LIMITS_ENABLED", "")
		assert.True(t, isEnforcementDisabled())
	})

	t.Run("enabled in dev with USAGE_LIMITS_ENABLED=true", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv("USAGE_LIMITS_ENABLED", "true")
		assert.False(t, isEnforcementDisabled())
	})

	t.Run("enabled in production", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("USAGE_LIMITS_ENABLED", "")
		assert.False(t, isEnforcementDisabled())
	})

	t.Run("enabled with no APP_ENV set", func(t *testing.T) {
		t.Setenv("APP_ENV", "")
		t.Setenv("USAGE_LIMITS_ENABLED", "")
		assert.False(t, isEnforcementDisabled())
	})
}

func TestGetDefaultProfileName(t *testing.T) {
	t.Run("returns basic by default", func(t *testing.T) {
		t.Setenv("USAGE_DEFAULT_PROFILE", "")
		assert.Equal(t, DefaultProfileName, getDefaultProfileName())
	})

	t.Run("returns env var value if set", func(t *testing.T) {
		t.Setenv("USAGE_DEFAULT_PROFILE", "enterprise")
		assert.Equal(t, "enterprise", getDefaultProfileName())
	})
}
