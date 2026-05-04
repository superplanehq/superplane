package claude

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
	integrationsetup "github.com/superplanehq/superplane/pkg/integrations/setup"
)

func newSetupProvider() core.IntegrationSetupProvider {
	return &integrationsetup.TokenProvider{
		IntegrationLabel:       "Claude",
		CapabilityGroupLabel:   "Models",
		CredentialStepLabel:    "Enter Claude API key",
		CredentialInstructions: "Create an API key in Anthropic Console, then paste it below.",
		Actions: []core.Action{
			&TextPrompt{},
			&runagent.RunAgent{},
		},
		Secrets: []integrationsetup.Secret{
			{
				Name:        "apiKey",
				Label:       "API Key",
				Description: "Claude API key",
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
