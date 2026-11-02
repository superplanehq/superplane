package github

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
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
const configFactoryOnboarding = "factoryOnboarding"

// WantsPrivateApp is true when CreateIntegration asked for a customer GitHub
// App instead of SuperPlane's public App.
func WantsPrivateApp(integrationName string, config map[string]any) bool {
	if integrationName != "github" {
		return false
	}
	if HostRequiresPrivateApp(config) {
		return true
	}
	if config == nil {
		return false
	}
	value, ok := config[configPrivateApp].(bool)
	return ok && value
}

// PreferHostedInstall is the default GitHub create path. A privateApp
// configuration flag or a GitHub Enterprise host opts out and uses the
// customer GitHub App wizard.
func PreferHostedInstall(orgID, integrationName string, config map[string]any) bool {
	if HostRequiresPrivateApp(config) {
		return false
	}
	return UseHostedInstall(orgID, integrationName) && !WantsPrivateApp(integrationName, config)
}

// WizardCreateConfiguration is the subset of create-time GitHub configuration
// the setup wizard persists. An empty host is github.com. Non-dotcom hosts
// require the github_enterprise license.
func WizardCreateConfiguration(orgID uuid.UUID, configMap map[string]any, httpCtx core.HTTPContext) (map[string]any, error) {
	host, err := hostFromConfig(configMap)
	if err != nil {
		return nil, err
	}
	if !host.IsDotCom() {
		if err := authorizeGitHubEnterpriseHost(host); err != nil {
			return nil, err
		}
		if !hostAllowed(host) {
			return nil, fmt.Errorf("GitHub host %s is not allowed", host.WebHostname())
		}
		if httpCtx == nil {
			return nil, fmt.Errorf("could not reach GitHub host")
		}
		if err := probeGitHubHost(httpCtx, host, orgID); err != nil {
			return nil, err
		}
	}

	out := map[string]any{}
	if host.IsDotCom() {
		out[common.ConfigHost] = ""
	} else {
		out[common.ConfigHost] = host.WebHostname()
		out[configPrivateApp] = true
	}
	if WantsPrivateApp("github", configMap) {
		out[configPrivateApp] = true
	}
	if wantsFactoryOnboarding(configMap) {
		out[configFactoryOnboarding] = true
	}
	return out, nil
}

func wantsFactoryOnboarding(config map[string]any) bool {
	if config == nil {
		return false
	}
	value, ok := config[configFactoryOnboarding].(bool)
	return ok && value
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
