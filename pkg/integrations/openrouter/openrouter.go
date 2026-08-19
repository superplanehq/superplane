package openrouter

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("openrouter", &OpenRouter{})
}

const (
	ResourceTypeModel    = "model"
	ResourceTypeProvider = "provider"
)

type OpenRouter struct{}

type Configuration struct {
	APIKey        string `json:"apiKey"`
	ManagementKey string `json:"managementKey"`
}

func (o *OpenRouter) Name() string {
	return "openrouter"
}

func (o *OpenRouter) Label() string {
	return "OpenRouter"
}

func (o *OpenRouter) Icon() string {
	return "openrouter"
}

func (o *OpenRouter) Description() string {
	return "Run prompts across hundreds of models through OpenRouter"
}

func (o *OpenRouter) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "OpenRouter API key (sk-or-v1-...)",
		},
		{
			Name:        "managementKey",
			Label:       "Provisioning API Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Provisioning key, used for account-wide activity reporting. Not required by the components below.",
		},
	}
}

func (o *OpenRouter) Actions() []core.Action {
	return []core.Action{
		&ChatCompletion{},
		&GetCredits{},
	}
}

func (o *OpenRouter) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (o *OpenRouter) Instructions() string {
	return `## OpenRouter API Key

Create an [OpenRouter API key](https://openrouter.ai/settings/keys) and copy it (starts with ` + "`sk-or-v1-`" + `).

- Used by every component below, including **Get Credits**.
- A credit limit can be set per key; leave it empty for no limit.

## Provisioning API Key (optional)

Not required by any component below. Create one at [Provisioning Keys](https://openrouter.ai/settings/provisioning-keys) only if you want the integration to verify account-wide activity access.

- Provisioning keys manage keys and read account activity, but cannot call model endpoints.

## Credits

Chat Completion spends credits from your OpenRouter account. Free model variants (IDs ending in ` + "`:free`" + `) draw from a shared upstream pool and are rate limited independently of your balance.

> **Note:** Keys are shown only once — store them somewhere safe before continuing.`
}

func (o *OpenRouter) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OpenRouter) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	// The provisioning key is optional and unused by the components, so a failed
	// verification must not block them from becoming ready.
	if config.ManagementKey != "" {
		if err := client.VerifyManagement(); err != nil && ctx.Logger != nil {
			ctx.Logger.Warnf("provisioning key verification failed: %v", err)
		}
	}

	ctx.Integration.Ready()
	return nil
}

func (o *OpenRouter) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (o *OpenRouter) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeModel:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		models, err := client.ListModels()
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(models))
		for _, model := range models {
			if model.ID == "" {
				continue
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: model.ID,
				ID:   model.ID,
			})
		}
		return resources, nil

	case ResourceTypeProvider:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		// Narrow the list to the providers actually serving the selected model,
		// since routing to one that does not serve it fails the request.
		if model := ctx.Parameters["model"]; model != "" {
			endpoints, err := client.ListModelEndpoints(model)
			if err != nil {
				return nil, err
			}
			return providerResourcesFromEndpoints(endpoints), nil
		}

		providers, err := client.ListProviders()
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(providers))
		for _, provider := range providers {
			if provider.Slug == "" {
				continue
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: provider.Name,
				ID:   provider.Slug,
			})
		}
		return resources, nil
	}

	return []core.IntegrationResource{}, nil
}

// providerResourcesFromEndpoints reduces a model's endpoints to the distinct
// provider slugs routing accepts. A provider can serve the same model from
// several regions, which carry a region suffix on the tag (e.g.
// "azure/swedencentral") that routing does not take.
func providerResourcesFromEndpoints(endpoints []ModelEndpoint) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(endpoints))
	seen := map[string]bool{}

	for _, endpoint := range endpoints {
		slug, _, _ := strings.Cut(endpoint.Tag, "/")
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true

		name := endpoint.ProviderName
		if name == "" {
			name = slug
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeProvider,
			Name: name,
			ID:   slug,
		})
	}

	return resources
}

func (o *OpenRouter) Hooks() []core.Hook {
	return []core.Hook{}
}

func (o *OpenRouter) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
