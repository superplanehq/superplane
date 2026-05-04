package launchdarkly

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	integrationsetup "github.com/superplanehq/superplane/pkg/integrations/setup"
)

func newSetupProvider() core.IntegrationSetupProvider {
	return &integrationsetup.TokenProvider{
		IntegrationLabel:       "LaunchDarkly",
		CapabilityGroupLabel:   "Feature flags",
		CredentialStepLabel:    "Enter LaunchDarkly API access token",
		CredentialInstructions: "Create a LaunchDarkly API access token with the required feature flag permissions, then paste it below.",
		Actions: []core.Action{
			&GetFeatureFlag{},
			&DeleteFeatureFlag{},
		},
		Triggers: []core.Trigger{
			&OnFeatureFlagChange{},
		},
		Secrets: []integrationsetup.Secret{
			{
				Name:        "apiKey",
				Label:       "API Access Token",
				Description: "LaunchDarkly API access token",
			},
		},
		Validate: func(ctx core.SetupStepContext, values map[string]string) error {
			return validateLaunchDarklyToken(ctx.HTTP, values["apiKey"])
		},
		ValidateSecret: func(ctx core.SecretUpdateContext, value string) error {
			return validateLaunchDarklyToken(ctx.HTTP, value)
		},
	}
}

func validateLaunchDarklyToken(http core.HTTPContext, apiKey string) error {
	client := NewClientWithAPIKey(http, apiKey)
	if _, err := client.ListProjects(); err != nil {
		return fmt.Errorf("error listing projects: %v", err)
	}

	return nil
}
