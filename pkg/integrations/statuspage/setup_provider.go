package statuspage

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	integrationsetup "github.com/superplanehq/superplane/pkg/integrations/setup"
)

func newSetupProvider() core.IntegrationSetupProvider {
	return &integrationsetup.TokenProvider{
		IntegrationLabel:       "Statuspage",
		CapabilityGroupLabel:   "Incidents",
		CredentialStepLabel:    "Enter Statuspage credentials",
		CredentialInstructions: "Paste your Statuspage API key. Leave Base URL empty for Atlassian Statuspage.",
		Actions: []core.Action{
			&CreateIncident{},
			&UpdateIncident{},
			&GetIncident{},
		},
		Properties: []integrationsetup.Property{
			{
				Name:        "baseURL",
				Label:       "API Base URL",
				Description: "Statuspage API base URL",
				Placeholder: defaultBaseURL,
				Required:    false,
			},
		},
		Secrets: []integrationsetup.Secret{
			{
				Name:        "apiKey",
				Label:       "API Key",
				Description: "Statuspage OAuth API key",
			},
		},
		Validate: func(ctx core.SetupStepContext, values map[string]string) error {
			return validateStatuspageCredentials(ctx.HTTP, values["apiKey"], values["baseURL"])
		},
		ValidateSecret: func(ctx core.SecretUpdateContext, value string) error {
			baseURL, _ := ctx.Properties.GetString("baseURL")
			return validateStatuspageCredentials(ctx.HTTP, value, baseURL)
		},
	}
}

func validateStatuspageCredentials(http core.HTTPContext, apiKey, baseURL string) error {
	client, err := NewClientWithAPIKey(http, apiKey, baseURL)
	if err != nil {
		return err
	}

	if _, err := client.ListPages(); err != nil {
		return fmt.Errorf("error verifying connection: %v", err)
	}

	return nil
}
