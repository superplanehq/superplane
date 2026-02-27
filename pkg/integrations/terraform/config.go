package terraform

import (
	"github.com/superplanehq/superplane/pkg/configuration"
)

type Configuration struct {
	Address       string `json:"address"`
	APIToken      string `json:"apiToken"`
	WebhookSecret string `json:"webhookSecret"`
}

type IntegrationMetadata struct{}

func getConfigurationFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "address",
			Label:       "Terraform Address",
			Type:        configuration.FieldTypeString,
			Default:     "https://app.terraform.io",
			Required:    true,
			Description: "The URL of the HCP Terraform or Terraform Enterprise instance.",
		},
		{
			Name:        "apiToken",
			Label:       "Team API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Your HCP Terraform Team API Token. This token must belong to a team with appropriate workspace permissions.",
		},
		{
			Name:        "webhookSecret",
			Label:       "Webhook Secret",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "A mandatory secret token used to cryptographically sign and verify incoming webhooks from Terraform.",
		},
	}
}
