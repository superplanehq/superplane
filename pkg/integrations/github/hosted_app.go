package github

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
)

// UseHostedApp reports whether new GitHub connections for this organization
// should install SuperPlane's public GitHub App. The org must have the
// factories feature, and the process must hold the Cloud app credentials.
func UseHostedApp(orgID string) bool {
	if !common.HostedAppConfigured() {
		return false
	}
	return factoriesEnabled(orgID)
}

var factoriesEnabled = factoriesEnabledForOrg

func factoriesEnabledForOrg(orgID string) bool {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return false
	}

	enabled, err := models.HasExperimentalFeature(id, features.FeatureFactories)
	return err == nil && enabled
}

func withFactoriesEnabledForTest(fn func(string) bool) func() {
	previous := factoriesEnabled
	factoriesEnabled = fn
	return func() { factoriesEnabled = previous }
}
