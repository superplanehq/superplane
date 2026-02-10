package digitalocean

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnDropletEvent struct{}

type OnDropletEventConfiguration struct {
	Events []string `json:"events"`
}

type OnDropletEventMetadata struct {
	LastPollTime string `json:"lastPollTime"`
}

func (t *OnDropletEvent) Name() string {
	return "digitalocean.onDropletEvent"
}

func (t *OnDropletEvent) Label() string {
	return "On Droplet Event"
}

func (t *OnDropletEvent) Description() string {
	return "Poll for DigitalOcean droplet lifecycle events"
}

func (t *OnDropletEvent) Documentation() string {
	return `The On Droplet Event trigger polls the DigitalOcean API for droplet lifecycle events.

## Use Cases

- **Infrastructure monitoring**: React to droplet creation and destruction events
- **Audit logging**: Track all droplet lifecycle changes
- **Automation**: Trigger workflows when droplets are powered on/off or resized

## Configuration

- **Events**: Select which droplet event types to listen for (create, destroy, power_on, power_off, shutdown, reboot, snapshot, rebuild, resize, rename)

## Polling

This trigger polls the DigitalOcean Actions API every 60 seconds for new completed droplet events matching the configured event types.

## Event Data

Each event includes:
- **action**: The DigitalOcean action object with id, status, type, timestamps, resource_id, and region_slug`
}

func (t *OnDropletEvent) Icon() string {
	return "server"
}

func (t *OnDropletEvent) Color() string {
	return "gray"
}

func (t *OnDropletEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"create", "destroy"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Create", Value: "create"},
						{Label: "Destroy", Value: "destroy"},
						{Label: "Power On", Value: "power_on"},
						{Label: "Power Off", Value: "power_off"},
						{Label: "Shutdown", Value: "shutdown"},
						{Label: "Reboot", Value: "reboot"},
						{Label: "Snapshot", Value: "snapshot"},
						{Label: "Rebuild", Value: "rebuild"},
						{Label: "Resize", Value: "resize"},
						{Label: "Rename", Value: "rename"},
					},
				},
			},
		},
	}
}

func (t *OnDropletEvent) Setup(ctx core.TriggerContext) error {
	config := OnDropletEventConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event type must be selected")
	}

	// Check if already set up
	var existingMetadata OnDropletEventMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &existingMetadata)
	if err == nil && existingMetadata.LastPollTime != "" {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	err = ctx.Metadata.Set(OnDropletEventMetadata{
		LastPollTime: now,
	})
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, 60*time.Second)
}

func (t *OnDropletEvent) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			Description:    "Poll for new droplet events",
			UserAccessible: false,
		},
	}
}

func (t *OnDropletEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "poll" {
		return nil, fmt.Errorf("action %s not supported", ctx.Name)
	}

	config := OnDropletEventConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var metadata OnDropletEventMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	lastPollTime, err := time.Parse(time.RFC3339, metadata.LastPollTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lastPollTime: %w", err)
	}

	// Capture the poll timestamp before the API call so events that complete
	// between the request and response are not permanently missed.
	now := time.Now().UTC().Format(time.RFC3339)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	actions, err := client.ListActions("droplet")
	if err != nil {
		// On transient API errors, reschedule the poll and return nil so the
		// framework does not roll back the scheduled call.
		_ = ctx.Requests.ScheduleActionCall("poll", map[string]any{}, 60*time.Second)
		return nil, nil
	}

	for _, action := range actions {
		if action.Status != "completed" {
			continue
		}

		completedAt, err := time.Parse(time.RFC3339, action.CompletedAt)
		if err != nil {
			continue
		}

		if !completedAt.After(lastPollTime) {
			continue
		}

		if !slices.Contains(config.Events, action.Type) {
			continue
		}

		payload := map[string]any{
			"action": map[string]any{
				"id":            action.ID,
				"status":        action.Status,
				"type":          action.Type,
				"started_at":    action.StartedAt,
				"completed_at":  action.CompletedAt,
				"resource_id":   action.ResourceID,
				"resource_type": action.ResourceType,
				"region_slug":   action.RegionSlug,
			},
		}

		// Enrich the payload with droplet details when available.
		// For destroy events the droplet may no longer exist.
		droplet, err := client.GetDroplet(action.ResourceID)
		if err == nil {
			payload["droplet"] = map[string]any{
				"id":        droplet.ID,
				"name":      droplet.Name,
				"size_slug": droplet.SizeSlug,
				"image": map[string]any{
					"name": droplet.Image.Name,
					"slug": droplet.Image.Slug,
				},
				"region": map[string]any{
					"name": droplet.Region.Name,
					"slug": droplet.Region.Slug,
				},
			}
		}

		err = ctx.Events.Emit(
			fmt.Sprintf("digitalocean.droplet.%s", action.Type),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("error emitting event: %v", err)
		}
	}

	err = ctx.Metadata.Set(OnDropletEventMetadata{
		LastPollTime: now,
	})
	if err != nil {
		return nil, fmt.Errorf("error updating metadata: %v", err)
	}

	err = ctx.Requests.ScheduleActionCall("poll", map[string]any{}, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error scheduling next poll: %v", err)
	}

	return nil, nil
}

func (t *OnDropletEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnDropletEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}
