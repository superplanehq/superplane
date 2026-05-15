package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// DeleteMonitorPayloadType is the emitted execution payload type (dash0-style: integration.resource.operation).
const DeleteMonitorPayloadType = "cloudflare.monitor.deleted"

type DeleteMonitor struct{}

type DeleteMonitorSpec struct {
	Monitor string `json:"monitor"`
	Force   bool   `json:"force"`
}

// MonitorNodeMetadata stores the resolved monitor identity for workflow canvas UI.
type MonitorNodeMetadata struct {
	MonitorID          string `json:"monitorId" mapstructure:"monitorId"`
	MonitorDescription string `json:"monitorDescription" mapstructure:"monitorDescription"`
}

func (c *DeleteMonitor) Name() string {
	return "cloudflare.deleteMonitor"
}

func (c *DeleteMonitor) Label() string {
	return "Delete Monitor"
}

func (c *DeleteMonitor) Description() string {
	return "Delete a Cloudflare load balancing health monitor"
}

func (c *DeleteMonitor) Documentation() string {
	return `The Delete Monitor component removes a Cloudflare Load Balancing health monitor.

## Use Cases

- **Cleanup**: Delete monitor definitions when load balancing resources are decommissioned
- **Rollback**: Remove monitors created during test or migration workflows

## Configuration

- **Monitor**: The monitor to delete.
- **Force**: Delete even when Cloudflare reports that pools or other resources reference the monitor.

## Output

Emits the deleted monitor ID and any references Cloudflare reported before deletion.`
}

func (c *DeleteMonitor) Icon() string {
	return "activity"
}

func (c *DeleteMonitor) Color() string {
	return "orange"
}

func (c *DeleteMonitor) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteMonitor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "monitor",
			Label:       "Monitor",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The load balancing health monitor to delete",
			Placeholder: "Select a monitor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "monitor",
				},
			},
		},
		{
			Name:        "force",
			Label:       "Force",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Delete even when Cloudflare reports references to this monitor",
		},
	}
}

func (c *DeleteMonitor) Setup(ctx core.SetupContext) error {
	spec := DeleteMonitorSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	monitorID := strings.TrimSpace(spec.Monitor)
	if monitorID == "" {
		return errors.New("monitor is required")
	}

	return resolveMonitorMetadata(ctx, monitorID, nil)
}

// preloaded, when non-nil, is used for node metadata instead of fetching again (e.g. Update Monitor
// already retrieved the monitor for validation).
func resolveMonitorMetadata(ctx core.SetupContext, monitorID string, preloaded *Monitor) error {
	if ctx.Metadata == nil || ctx.Integration == nil || ctx.HTTP == nil {
		return nil
	}

	id := strings.TrimSpace(monitorID)
	if strings.Contains(id, "{{") {
		return ctx.Metadata.Set(MonitorNodeMetadata{
			MonitorID:          id,
			MonitorDescription: id,
		})
	}

	var existing MonitorNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil && existing.MonitorID == id && existing.MonitorDescription != "" {
		return nil
	}

	var monitor *Monitor
	if preloaded != nil {
		monitor = preloaded
	} else {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create client for monitor metadata: %w", err)
		}

		accountID, err := accountIDForIntegration(ctx.Integration)
		if err != nil {
			return err
		}

		monitor, err = client.GetMonitor(accountID, id)
		if err != nil {
			return fmt.Errorf("failed to fetch monitor %s for metadata: %w", id, err)
		}
	}

	desc := strings.TrimSpace(monitor.Description)
	if desc == "" {
		desc = id
	}

	return ctx.Metadata.Set(MonitorNodeMetadata{
		MonitorID:          id,
		MonitorDescription: desc,
	})
}

func (c *DeleteMonitor) Execute(ctx core.ExecutionContext) error {
	spec := DeleteMonitorSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	monitorID := strings.TrimSpace(spec.Monitor)
	if monitorID == "" {
		return errors.New("monitor is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	accountID, err := accountIDForIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	references, err := client.ListMonitorReferences(accountID, monitorID)
	if err != nil {
		return fmt.Errorf("failed to list monitor references: %w", err)
	}

	if len(references) > 0 && !spec.Force {
		return fmt.Errorf("monitor %s is still referenced by %d resource(s); set force to delete anyway", monitorID, len(references))
	}

	deleted, err := client.DeleteMonitor(accountID, monitorID)
	if err != nil {
		return fmt.Errorf("failed to delete monitor: %w", err)
	}

	deletedID := deleted.ID
	if deletedID == "" {
		deletedID = monitorID
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteMonitorPayloadType, []any{
		map[string]any{
			"accountId":  accountID,
			"monitorId":  deletedID,
			"deleted":    true,
			"references": references,
		},
	})
}

func (c *DeleteMonitor) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteMonitor) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteMonitor) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteMonitor) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteMonitor) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteMonitor) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
