package render

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	PurgeCachePayloadType   = "render.purge.cache"
	PurgeCacheOutputChannel = "default"
)

type PurgeCache struct{}

type PurgeCacheConfiguration struct {
	ServiceID string `json:"serviceId" mapstructure:"serviceId"`
}

func (c *PurgeCache) Name() string {
	return "render.purge_cache"
}

func (c *PurgeCache) Label() string {
	return "Purge Cache"
}

func (c *PurgeCache) Description() string {
	return "Purge the build cache for a Render service"
}

func (c *PurgeCache) Documentation() string {
	return `The Purge Cache component clears the build cache for a Render service.

## Use Cases

- **Stale cache issues**: Purge cache when builds fail due to stale dependencies
- **Clean rebuilds**: Force a clean build by purging cache before deploying

## Configuration

- **Service**: The Render service whose cache to purge

## Output

Emits a confirmation payload on the default channel.`
}

func (c *PurgeCache) Icon() string {
	return "trash-2"
}

func (c *PurgeCache) Color() string {
	return "gray"
}

func (c *PurgeCache) ExampleOutput() map[string]any {
	return nil
}

func (c *PurgeCache) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: PurgeCacheOutputChannel, Label: "Default"},
	}
}

func (c *PurgeCache) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "serviceId",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
			Description: "Render service whose cache to purge",
		},
	}
}

func decodePurgeCacheConfiguration(configuration any) (PurgeCacheConfiguration, error) {
	spec := PurgeCacheConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return PurgeCacheConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ServiceID = strings.TrimSpace(spec.ServiceID)
	if spec.ServiceID == "" {
		return PurgeCacheConfiguration{}, fmt.Errorf("serviceId is required")
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

	if err := client.PurgeCache(spec.ServiceID); err != nil {
		return err
	}

	payload := map[string]any{
		"serviceId": spec.ServiceID,
		"purged":    true,
	}

	return ctx.ExecutionState.Emit(PurgeCacheOutputChannel, PurgeCachePayloadType, []any{payload})
}

func (c *PurgeCache) Actions() []core.Action {
	return []core.Action{}
}

func (c *PurgeCache) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *PurgeCache) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (c *PurgeCache) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PurgeCache) Cleanup(ctx core.SetupContext) error {
	return nil
}

// Client method for PurgeCache
func (cl *Client) PurgeCache(serviceID string) error {
	if serviceID == "" {
		return fmt.Errorf("serviceID is required")
	}

	_, _, err := cl.execRequestWithResponse(
		"POST",
		"/services/"+url.PathEscape(serviceID)+"/cache/purge",
		nil,
		nil,
	)

	return err
}
