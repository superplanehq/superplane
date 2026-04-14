package hashicorp_vault

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("hashicorp_vault", &HashicorpVault{})
}

type HashicorpVault struct{}

type vaultSettings struct {
	BaseURL   string `mapstructure:"baseURL"`
	Namespace string `mapstructure:"namespace"`
	Token     string `mapstructure:"token"`
}

func (v *HashicorpVault) Name() string         { return "hashicorp_vault" }
func (v *HashicorpVault) Label() string        { return "HashiCorp Vault" }
func (v *HashicorpVault) Icon() string         { return "vault" }
func (v *HashicorpVault) Description() string  { return "Securely read secrets from HashiCorp Vault" }
func (v *HashicorpVault) Instructions() string { return "" }

func (v *HashicorpVault) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Vault server URL, e.g. https://vault.example.com",
		},
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Vault Enterprise namespace. Leave empty for community edition.",
		},
		{
			Name:        "token",
			Label:       "Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Vault token (hvs.… or s.…)",
		},
	}
}

func (v *HashicorpVault) Components() []core.Component {
	return []core.Component{
		&getSecret{},
	}
}
func (v *HashicorpVault) Triggers() []core.Trigger      { return []core.Trigger{} }
func (v *HashicorpVault) Actions() []core.Action        { return []core.Action{} }
func (v *HashicorpVault) HandleRequest(ctx core.HTTPRequestContext) {}

func (v *HashicorpVault) Cleanup(ctx core.IntegrationCleanupContext) error    { return nil }
func (v *HashicorpVault) HandleAction(ctx core.IntegrationActionContext) error { return nil }

func (v *HashicorpVault) Sync(ctx core.SyncContext) error {
	cfg := vaultSettings{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if cfg.BaseURL == "" {
		return fmt.Errorf("baseURL is required")
	}

	if cfg.Token == "" {
		return fmt.Errorf("token is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.LookupSelf(); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (v *HashicorpVault) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}
