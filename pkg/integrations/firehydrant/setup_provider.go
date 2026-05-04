package firehydrant

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	integrationsetup "github.com/superplanehq/superplane/pkg/integrations/setup"
)

func newSetupProvider() core.IntegrationSetupProvider {
	return &integrationsetup.TokenProvider{
		IntegrationLabel:       "FireHydrant",
		CapabilityGroupLabel:   "Incidents",
		CredentialStepLabel:    "Enter FireHydrant API key",
		CredentialInstructions: "Create a FireHydrant API key with write access, then paste it below.",
		Actions: []core.Action{
			&CreateIncident{},
		},
		Triggers: []core.Trigger{
			&OnIncident{},
		},
		Secrets: []integrationsetup.Secret{
			{
				Name:        "apiKey",
				Label:       "API Key",
				Description: "API key from FireHydrant",
			},
		},
		Validate: func(ctx core.SetupStepContext, values map[string]string) error {
			return validateFireHydrantToken(ctx.HTTP, values["apiKey"])
		},
		ValidateSecret: func(ctx core.SecretUpdateContext, value string) error {
			return validateFireHydrantToken(ctx.HTTP, value)
		},
	}
}

func validateFireHydrantToken(http core.HTTPContext, apiKey string) error {
	client := NewClientWithAPIKey(http, apiKey)
	if _, err := client.ListSeverities(); err != nil {
		return fmt.Errorf("error listing severities: %v", err)
	}

	return nil
}
