package perplexity

import (
	"github.com/superplanehq/superplane/pkg/core"
	integrationsetup "github.com/superplanehq/superplane/pkg/integrations/setup"
)

func newSetupProvider() core.IntegrationSetupProvider {
	return &integrationsetup.TokenProvider{
		IntegrationLabel:       "Perplexity",
		CapabilityGroupLabel:   "Agents",
		CredentialStepLabel:    "Enter Perplexity API key",
		CredentialInstructions: "Create a Perplexity API key, then paste it below.",
		Actions: []core.Action{
			&runAgent{},
		},
		Secrets: []integrationsetup.Secret{
			{
				Name:        "apiKey",
				Label:       "API Key",
				Description: "Perplexity API key (pplx-...)",
			},
		},
		Validate: func(ctx core.SetupStepContext, values map[string]string) error {
			return NewClientWithAPIKey(ctx.HTTP, values["apiKey"]).Verify()
		},
		ValidateSecret: func(ctx core.SecretUpdateContext, value string) error {
			return NewClientWithAPIKey(ctx.HTTP, value).Verify()
		},
	}
}
