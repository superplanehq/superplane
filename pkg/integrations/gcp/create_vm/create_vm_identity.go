package createvm

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

type IdentityConfig struct {
	ServiceAccount      string   `mapstructure:"serviceAccount"`
	OAuthScopes         []string `mapstructure:"oauthScopes"`
	BlockProjectSSHKeys bool     `mapstructure:"blockProjectSSHKeys"`
	EnableOSLogin       bool     `mapstructure:"enableOSLogin"`
}

func CreateVMIdentityConfigFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serviceAccount",
			Label:       "Service account (VM identity)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email of the service account this VM will run as. Leave empty to use the project's default Compute Engine service account.",
			Placeholder: "e.g. my-sa@my-project.iam.gserviceaccount.com",
		},
		{
			Name:        "oauthScopes",
			Label:       "OAuth scopes",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Access scopes for the VM (which APIs the instance can call). Leave empty for default (cloud-platform).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Scope",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "blockProjectSSHKeys",
			Label:       "Block project-wide SSH keys",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "If enabled, only instance-level SSH keys or OS Login will work; project-wide SSH keys are ignored.",
			Default:     false,
		},
		{
			Name:        "enableOSLogin",
			Label:       "Enable OS Login",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Use OS Login for SSH access (IAM-based). When enabled, SSH keys are managed via IAM and OS Login.",
			Default:     false,
		},
	}
}

func NormalizeOAuthScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	result := make([]string, 0, len(scopes))
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
