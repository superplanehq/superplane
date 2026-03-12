package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PurgeCachePayloadType = "render.cache.purge.requested"

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
	return "Request a purge of the build cache for a Render service"
}

func (c *PurgeCache) Documentation() string {
	return `The Purge Cache component requests a build cache purge for a Render service.

## Use Cases

- **Cache reset**: Force a clean rebuild when you suspect stale dependencies or build artifacts
- **Operational tooling**: Provide a one-click cache purge in incident response workflows

## Configuration

- **Service**: Render service whose build cache should be purged

## Output

Emits a ` + "`render.cache.purge.requested`" + ` payload with ` + "`serviceId`" + ` and a ` + "`status`" + ` field indicating the request was accepted.`
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
			Description: "Render service whose build cache should be purged",
		},
	}
}

func decodePurgeCacheConfiguration(configuration any) (PurgeCacheConfiguration, error) {
	spec := PurgeCacheConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return PurgeCacheConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	if spec.Service == "" {
		return PurgeCacheConfiguration{}, fmt.Errorf("service is required")
	}

	return spec, nil
}

func (c *PurgeCache) Setup(ctx core.SetupContext) error {
	_, err := decodePurgeCacheConfiguration(ctx.Configuration)
	return err
}

func (c *PurgeCache) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PurgeCache) Execute(ctx core.ExecutionContext) error {
	spec, err := decodePurgeCacheConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.PurgeCache(spec.Service); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PurgeCachePayloadType,
		[]any{
			map[string]any{
				"serviceId": spec.Service,
				"status":    "accepted",
			},
		},
	)
}

func (c *PurgeCache) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
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
