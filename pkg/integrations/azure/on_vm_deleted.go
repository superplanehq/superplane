package azure

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnVMDeleted struct {
}

type OnVMDeletedConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NameFilter    string `json:"nameFilter" mapstructure:"nameFilter"`
}

func (t *OnVMDeleted) Name() string {
	return "azure.onVirtualMachineDeleted"
}

func (t *OnVMDeleted) Label() string {
	return "On VM Deleted"
}

func (t *OnVMDeleted) Description() string {
	return "Listen to Azure VM deletion events"
}

func (t *OnVMDeleted) Documentation() string {
	return `
The On VM Deleted trigger starts a workflow execution when an Azure Virtual Machine is deleted.

## Use Cases

- **Cleanup workflows**: Remove DNS records, monitoring agents, or other dependent resources when a VM is deleted
- **Inventory tracking**: Update external inventory systems when VMs are removed
- **Notification workflows**: Send notifications to teams when VMs are deleted
- **Cost tracking**: Log VM deletion events for cost analysis and reporting
- **Compliance auditing**: Track and audit VM deletions for security and compliance

## How It Works

This trigger listens to Azure Event Grid events for Virtual Machine resource delete operations.
When a VM delete operation succeeds (` + "`status: Succeeded`" + `), the trigger fires and
provides the full Azure Event Grid event payload.

Azure fires ` + "`Microsoft.Resources.ResourceDeleteSuccess`" + ` when a VM is successfully deleted.
This is a distinct event from write operations — it only fires when the VM is actually removed,
not during creation or updates.

## Configuration

- **Resource Group** (optional): Filter events to only trigger for VMs in a specific
  resource group. Leave empty to trigger for all resource groups in the subscription.
- **VM Name Filter** (optional): A regex pattern to filter VMs by name. Only VMs whose name
  matches the pattern will trigger the workflow. Leave empty to trigger for all VM names.

## Event Data

Each VM delete event includes the full Azure Event Grid event:

- **id**: Unique event ID
- **topic**: The Azure subscription topic
- **subject**: The full ARM resource ID of the VM
- **eventType**: The event type (` + "`Microsoft.Resources.ResourceDeleteSuccess`" + `)
- **eventTime**: The timestamp when the event occurred
- **data**: The event data including operationName, status, resourceProvider, resourceUri, subscriptionId, tenantId

## Azure Event Grid Setup

Event Grid subscriptions are created automatically when the trigger is set up. SuperPlane will:

1. Create an Event Grid subscription at the Azure subscription scope
2. Configure it to forward ` + "`Microsoft.Resources.ResourceDeleteSuccess`" + ` events to the trigger webhook
3. Apply subject filters based on the configured resource group and resource type
4. Handle the Event Grid validation handshake automatically

No manual setup is required.

## Notes

- The trigger fires only when a VM is successfully deleted
- Failed delete operations do not trigger the workflow
- The trigger processes events from Azure Event Grid in real-time
- Multiple triggers can share the same Event Grid subscription if configured correctly
`
}

func (t *OnVMDeleted) Icon() string {
	return "azure"
}

func (t *OnVMDeleted) Color() string {
	return "blue"
}

func (t *OnVMDeleted) ExampleData() map[string]any {
	return map[string]any{
		"id":              "96257b6d-17d3-49e2-8369-fb185b29e1b5",
		"topic":           "/subscriptions/12345678-1234-1234-1234-123456789abc",
		"subject":         "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm-01",
		"eventType":       "Microsoft.Resources.ResourceDeleteSuccess",
		"eventTime":       "2026-02-11T10:30:00Z",
		"dataVersion":     "2",
		"metadataVersion": "1",
		"data": map[string]any{
			"operationName":    "Microsoft.Compute/virtualMachines/delete",
			"status":           "Succeeded",
			"resourceProvider": "Microsoft.Compute",
			"resourceUri":      "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm-01",
			"subscriptionId":   "12345678-1234-1234-1234-123456789abc",
			"tenantId":         "12345678-1234-1234-1234-123456789abc",
		},
	}
}

func (t *OnVMDeleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter events to a specific resource group (optional - leave empty for all resource groups)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeResourceGroupDropdown,
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "nameFilter",
			Label:       "VM Name Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., prod-.*",
			Description: "Optional regex pattern to filter VMs by name",
		},
	}
}

// Setup configures trigger webhooks.
func (t *OnVMDeleted) Setup(ctx core.TriggerContext) error {
	config := OnVMDeletedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	err := ctx.Integration.RequestWebhook(AzureWebhookConfiguration{
		EventTypes:    []string{EventTypeResourceDeleteSuccess},
		ResourceType:  ResourceTypeVirtualMachine,
		ResourceGroup: config.ResourceGroup,
	})
	if err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	ctx.Logger.Info("Azure On VM Deleted trigger configured successfully")
	if config.ResourceGroup != "" {
		ctx.Logger.Infof("Filtering events for resource group: %s", config.ResourceGroup)
	} else {
		ctx.Logger.Info("Listening for VM delete events in all resource groups")
	}

	return nil
}

// HandleWebhook processes Event Grid webhook requests.
func (t *OnVMDeleted) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := t.authenticateWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, nil, err
	}

	config := OnVMDeletedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Parse as typed structs for filtering logic.
	var events []EventGridEvent
	if err := json.Unmarshal(ctx.Body, &events); err != nil {
		ctx.Logger.Errorf("Failed to parse Event Grid events: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse events: %w", err)
	}

	// Also parse as raw maps so we can emit the full, unmodified event data.
	var rawEvents []map[string]any
	if err := json.Unmarshal(ctx.Body, &rawEvents); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse raw events: %w", err)
	}

	ctx.Logger.Infof("Received %d Event Grid event(s), raw body length: %d bytes", len(events), len(ctx.Body))

	for i, event := range events {
		ctx.Logger.Infof("Event[%d]: id=%s type=%s subject=%s", i, event.ID, event.EventType, event.Subject)

		if event.EventType == EventTypeSubscriptionValidation {
			resp, err := t.handleSubscriptionValidation(ctx, event)
			if err != nil {
				return http.StatusInternalServerError, nil, err
			}
			return http.StatusOK, resp, nil
		}

		if event.EventType == EventTypeResourceDeleteSuccess {
			if err := t.handleVMDeletedEvent(ctx, event, rawEvents[i], config); err != nil {
				ctx.Logger.Errorf("Failed to process VM deleted event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil, nil
}

// handleSubscriptionValidation validates Event Grid subscription using the
// synchronous handshake: return the validationCode in the HTTP response body.
func (t *OnVMDeleted) handleSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) (*core.WebhookResponseBody, error) {
	var validationData SubscriptionValidationEventData
	if err := mapstructure.Decode(event.Data, &validationData); err != nil {
		return nil, fmt.Errorf("failed to parse validation data: %w", err)
	}

	if validationData.ValidationCode == "" {
		return nil, fmt.Errorf("validation code is empty")
	}

	ctx.Logger.Infof("Event Grid subscription validation received, responding with validation code")

	body, err := json.Marshal(map[string]string{
		"validationResponse": validationData.ValidationCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation response: %w", err)
	}

	ctx.Logger.Info("Event Grid subscription validation response set successfully")
	return &core.WebhookResponseBody{Body: body, ContentType: "application/json"}, nil
}

// handleVMDeletedEvent processes VM delete events.
func (t *OnVMDeleted) handleVMDeletedEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	rawEvent map[string]any,
	config OnVMDeletedConfiguration,
) error {
	ctx.Logger.Infof("Processing event: type=%s subject=%s", event.EventType, event.Subject)

	if !isVirtualMachineEvent(event.Subject) {
		ctx.Logger.Infof("Skipping non-VM event with subject: %s", event.Subject)
		return nil
	}

	var eventData ResourceDeleteSuccessData
	if err := mapstructure.Decode(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	ctx.Logger.Infof("VM event data: status=%s operationName=%s resourceURI=%s", eventData.Status, eventData.OperationName, eventData.ResourceURI)

	if !isSuccessfulStatus(eventData.Status) {
		ctx.Logger.Infof("Skipping VM event with non-successful status: %s", eventData.Status)
		return nil
	}

	resourceGroup := extractResourceGroup(event.Subject)
	if resourceGroup == "" {
		ctx.Logger.Warnf("Could not extract resource group from subject: %s", event.Subject)
	}

	if config.ResourceGroup != "" && !strings.EqualFold(resourceGroup, config.ResourceGroup) {
		ctx.Logger.Debugf("Skipping VM event for resource group %s (filter: %s)", resourceGroup, config.ResourceGroup)
		return nil
	}

	vmName := extractVMName(event.Subject)

	// Apply name filter if configured
	if config.NameFilter != "" {
		matched, err := regexp.MatchString(config.NameFilter, vmName)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}

		if !matched {
			ctx.Logger.Debugf("Skipping VM %s (name filter: %s)", vmName, config.NameFilter)
			return nil
		}
	}

	ctx.Logger.Infof("VM deleted event: %s in resource group %s", vmName, resourceGroup)

	// Emit the full, unmodified Azure Event Grid event — same pattern as GitHub, GitLab, etc.
	if err := ctx.Events.Emit("azure.vm.deleted", rawEvent); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	ctx.Logger.Infof("Successfully emitted azure.vm.deleted event for VM: %s", vmName)
	return nil
}

// authenticateWebhook verifies the webhook secret if one is configured.
func (t *OnVMDeleted) authenticateWebhook(ctx core.WebhookRequestContext) error {
	if ctx.Webhook == nil {
		return nil
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		ctx.Logger.Debugf("Could not retrieve webhook secret: %v", err)
		return nil
	}

	if len(secret) == 0 {
		return nil
	}

	// Check Authorization header (Bearer token)
	authHeader := ctx.Headers.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		providedSecret := strings.TrimPrefix(authHeader, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(providedSecret), secret) == 1 {
			return nil
		}
		return fmt.Errorf("invalid webhook secret")
	}

	// Check custom header
	secretHeader := ctx.Headers.Get("X-Webhook-Secret")
	if secretHeader != "" {
		if subtle.ConstantTimeCompare([]byte(secretHeader), secret) == 1 {
			return nil
		}
		return fmt.Errorf("invalid webhook secret")
	}

	return fmt.Errorf("webhook secret required but not provided in Authorization or X-Webhook-Secret header")
}

func (t *OnVMDeleted) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnVMDeleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

// Cleanup is called when the trigger is removed.
func (t *OnVMDeleted) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure On VM Deleted trigger")
	return nil
}
