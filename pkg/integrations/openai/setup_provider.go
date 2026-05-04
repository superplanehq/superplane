package openai

import (
	"github.com/superplanehq/superplane/pkg/core"
	integrationsetup "github.com/superplanehq/superplane/pkg/integrations/setup"
)

func newSetupProvider() core.IntegrationSetupProvider {
	return &integrationsetup.TokenProvider{
		IntegrationLabel:       "OpenAI",
		CapabilityGroupLabel:   "Models",
		CredentialStepLabel:    "Enter OpenAI credentials",
		CredentialInstructions: "Paste your OpenAI API key. Leave Base URL empty unless you use an OpenAI-compatible provider.",
		Actions: []core.Action{
			&CreateResponse{},
		},
		Properties: []integrationsetup.Property{
			{
				Name:        "baseURL",
				Label:       "Base URL",
				Description: "Custom API base URL for OpenAI-compatible providers",
				Placeholder: defaultBaseURL,
				Required:    false,
			},
		},
		Secrets: []integrationsetup.Secret{
			{
				Name:        "apiKey",
				Label:       "API Key",
				Description: "OpenAI API key",
			},
		},
		Validate: func(ctx core.SetupStepContext, values map[string]string) error {
			return NewClientWithAPIKey(ctx.HTTP, values["apiKey"], values["baseURL"]).Verify()
		},
		ValidateSecret: func(ctx core.SecretUpdateContext, value string) error {
			baseURL, _ := ctx.Properties.GetString("baseURL")
			return NewClientWithAPIKey(ctx.HTTP, value, baseURL).Verify()
		},
	}
}
