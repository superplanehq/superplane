package render

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type PurgeCache struct{}

type PurgeCacheConfiguration struct {
	Service string `json:"service" mapstructure:"service"`
}

func (c *PurgeCache) Name() string {
	return "render.purgeCache"
}

func (c *PurgeCache) Label() string {
	return "Purge Cache"
}

func (c *PurgeCache) Description() string {
	return "Purge the build cache of a Render service"
}

func (c *PurgeCache) Documentation() string {
	return `The Purge Cache component clears the build cache for a Render service.

## Use Cases

- **Stale cache issues**: Force a clean build when cache corruption causes failures
- **Dependency updates**: Clear cache after major dependency version changes
- **Pre-deploy cleanup**: Purge cache before triggering a fresh deploy

## Configuration

- **Service**: The Render service whose cache should be purged

## Output

Returns a confirmation payload with the service ID on success.`
}

func (c *PurgeCache) Icon() string {
	return "trash-2"
}

func (c *PurgeCache) Color() string {
	return "gray"
}

func (c *PurgeCache) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PurgeCache) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "service",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
			Description: "Render service whose cache should be purged",
		},
	}
}

func (c *PurgeCache) Setup(ctx core.SetupContext) error {
	config, err := decodePurgeCacheConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if config.Service == "" {
		return fmt.Errorf("service is required")
	}

	return nil
}

func (c *PurgeCache) Execute(ctx core.ExecutionContext) error {
	config, err := decodePurgeCacheConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.PurgeCache(config.Service); err != nil {
		return fmt.Errorf("failed to purge cache: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"render.cache.purged",
		[]any{map[string]any{
			"serviceId": config.Service,
		}},
	)
}

func (c *PurgeCache) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PurgeCache) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *PurgeCache) Actions() []core.Action {
	return []core.Action{}
}

func (c *PurgeCache) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PurgeCache) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PurgeCache) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodePurgeCacheConfiguration(configuration any) (PurgeCacheConfiguration, error) {
	config := PurgeCacheConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return PurgeCacheConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Service = strings.TrimSpace(config.Service)
	return config, nil
}
