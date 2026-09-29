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

// UseHostedInstall is true when this integration create should skip the
// SetupProvider wizard and the organization dialog, and install the public app.
func UseHostedInstall(orgID, integrationName string) bool {
	return integrationName == "github" && UseHostedApp(orgID)
}

const configPrivateApp = "privateApp"

// WantsPrivateApp is true when CreateIntegration asked for a customer GitHub
// App instead of SuperPlane's public App.
func WantsPrivateApp(integrationName string, config map[string]any) bool {
	if integrationName != "github" || config == nil {
		return false
	}
	value, ok := config[configPrivateApp].(bool)
	return ok && value
}

// PreferHostedInstall is the default GitHub create path. A privateApp
// configuration flag opts out and uses the customer GitHub App wizard.
func PreferHostedInstall(orgID, integrationName string, config map[string]any) bool {
	return UseHostedInstall(orgID, integrationName) && !WantsPrivateApp(integrationName, config)
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
