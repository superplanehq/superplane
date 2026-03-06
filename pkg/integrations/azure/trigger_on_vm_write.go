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

type OnVMWriteTrigger struct {
	integration *AzureIntegration
}

type OnVMWriteConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NameFilter    string `json:"nameFilter" mapstructure:"nameFilter"`
}

func (t *OnVMWriteTrigger) Name() string {
	return "azure.onVirtualMachineWrite"
}

func (t *OnVMWriteTrigger) Label() string {
	return "Azure • On VM Write"
}

func (t *OnVMWriteTrigger) Description() string {
	return "Triggers when a Virtual Machine write operation succeeds in Azure (create, update, or delete)"
}

func (t *OnVMWriteTrigger) Documentation() string {
	return `
The On VM Write trigger starts a workflow execution when an Azure Virtual Machine write operation succeeds.
This includes VM creation, updates, and state changes during deletion.

## Use Cases

- **Automated configuration**: Run configuration scripts on newly created VMs
- **Compliance checks**: Verify that VMs meet security and compliance requirements after changes
- **Inventory tracking**: Update external inventory systems when VMs are created, modified, or deleted
- **Notification workflows**: Send notifications to teams when VM changes occur
- **Cost tracking**: Log VM write events for cost analysis and reporting
- **Cleanup workflows**: Trigger cleanup tasks when VMs are deleted

## How It Works

This trigger listens to Azure Event Grid events for Virtual Machine resource write operations.
When a VM write operation succeeds (` + "`status: Succeeded`" + `), the trigger fires and
provides the full Azure Event Grid event payload.

Azure fires ` + "`Microsoft.Resources.ResourceWriteSuccess`" + ` for all successful ARM write
operations on a VM, including creation, updates, and state changes during deletion.

## Configuration

- **Resource Group** (optional): Filter events to only trigger for VMs in a specific
  resource group. Leave empty to trigger for all resource groups in the subscription.
- **VM Name Filter** (optional): A regex pattern to filter VMs by name. Only VMs whose name
  matches the pattern will trigger the workflow. Leave empty to trigger for all VM names.

## Event Data

Each VM write event includes the full Azure Event Grid event:

- **id**: Unique event ID
- **topic**: The Azure subscription topic
- **subject**: The full ARM resource ID of the VM
- **eventType**: The event type (typically ` + "`Microsoft.Resources.ResourceWriteSuccess`" + `)
- **eventTime**: The timestamp when the event occurred
- **data**: The event data including operationName, status, resourceProvider, resourceUri, subscriptionId, tenantId

## Azure Event Grid Setup

Event Grid subscriptions are created automatically when the trigger is set up. SuperPlane will:

1. Create an Event Grid subscription at the Azure subscription scope
2. Configure it to forward ` + "`Microsoft.Resources.ResourceWriteSuccess`" + ` events to the trigger webhook
3. Apply subject filters based on the configured resource group and resource type
4. Handle the Event Grid validation handshake automatically

No manual setup is required.

## Notes

- The trigger fires for all successful VM write operations (create, update, delete)
- Failed operations do not trigger the workflow
- The trigger processes events from Azure Event Grid in real-time
- Multiple triggers can share the same Event Grid subscription if configured correctly
`
}

func (t *OnVMWriteTrigger) Icon() string {
	return "azure"
}

func (t *OnVMWriteTrigger) Color() string {
	return "blue"
}

func (t *OnVMWriteTrigger) ExampleData() map[string]any {
	return map[string]any{
		"id":              "96257b6d-17d3-49e2-8369-fb185b29e1b5",
		"topic":           "/subscriptions/12345678-1234-1234-1234-123456789abc",
		"subject":         "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm-01",
		"eventType":       "Microsoft.Resources.ResourceWriteSuccess",
		"eventTime":       "2026-02-11T10:30:00Z",
		"dataVersion":     "2",
		"metadataVersion": "1",
		"data": map[string]any{
			"operationName":    "Microsoft.Compute/virtualMachines/write",
			"status":           "Succeeded",
			"resourceProvider": "Microsoft.Compute",
			"resourceUri":      "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm-01",
			"subscriptionId":   "12345678-1234-1234-1234-123456789abc",
			"tenantId":         "12345678-1234-1234-1234-123456789abc",
		},
	}
}

func (t *OnVMWriteTrigger) Configuration() []configuration.Field {
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
func (t *OnVMWriteTrigger) Setup(ctx core.TriggerContext) error {
	config := OnVMWriteConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if ctx.Integration != nil {
		err := ctx.Integration.RequestWebhook(AzureWebhookConfiguration{
			EventTypes:    []string{EventTypeResourceWriteSuccess},
			ResourceType:  ResourceTypeVirtualMachine,
			ResourceGroup: config.ResourceGroup,
		})
		if err != nil {
			return fmt.Errorf("failed to request webhook: %w", err)
		}
	} else {
		ctx.Logger.Warn("Integration context missing; skipping webhook request")
	}

	ctx.Logger.Info("Azure VM Write trigger configured successfully")
	if config.ResourceGroup != "" {
		ctx.Logger.Infof("Filtering events for resource group: %s", config.ResourceGroup)
	} else {
		ctx.Logger.Info("Listening for VM write events in all resource groups")
	}

	return nil
}

// HandleWebhook processes Event Grid webhook requests.
func (t *OnVMWriteTrigger) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := t.authenticateWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, err
	}

	config := OnVMWriteConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Parse as typed structs for filtering logic.
	var events []EventGridEvent
	if err := json.Unmarshal(ctx.Body, &events); err != nil {
		ctx.Logger.Errorf("Failed to parse Event Grid events: %v", err)
		return http.StatusBadRequest, fmt.Errorf("failed to parse events: %w", err)
	}

	// Also parse as raw maps so we can emit the full, unmodified event data.
	var rawEvents []map[string]any
	if err := json.Unmarshal(ctx.Body, &rawEvents); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse raw events: %w", err)
	}

	ctx.Logger.Infof("Received %d Event Grid event(s), raw body length: %d bytes", len(events), len(ctx.Body))

	for i, event := range events {
		ctx.Logger.Infof("Event[%d]: id=%s type=%s subject=%s", i, event.ID, event.EventType, event.Subject)

		if event.EventType == EventTypeSubscriptionValidation {
			if err := t.handleSubscriptionValidation(ctx, event); err != nil {
				return http.StatusInternalServerError, err
			}
			return http.StatusOK, nil
		}

		if event.EventType == EventTypeResourceWriteSuccess {
			if err := t.handleVMWriteEvent(ctx, event, rawEvents[i], config); err != nil {
				ctx.Logger.Errorf("Failed to process VM write event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil
}

// handleSubscriptionValidation validates Event Grid subscription using the
// synchronous handshake: return the validationCode in the HTTP response body.
func (t *OnVMWriteTrigger) handleSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) error {
	var validationData SubscriptionValidationEventData
	if err := mapstructure.Decode(event.Data, &validationData); err != nil {
		return fmt.Errorf("failed to parse validation data: %w", err)
	}

	if validationData.ValidationCode == "" {
		return fmt.Errorf("validation code is empty")
	}

	ctx.Logger.Infof("Event Grid subscription validation received, responding with validation code")

	if ctx.Response != nil {
		body, err := json.Marshal(map[string]string{
			"validationResponse": validationData.ValidationCode,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal validation response: %w", err)
		}

		ctx.Response.Body = body
		ctx.Response.ContentType = "application/json"
	}

	ctx.Logger.Info("Event Grid subscription validation response set successfully")
	return nil
}

// handleVMWriteEvent processes VM write events.
func (t *OnVMWriteTrigger) handleVMWriteEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	rawEvent map[string]any,
	config OnVMWriteConfiguration,
) error {
	ctx.Logger.Infof("Processing event: type=%s subject=%s", event.EventType, event.Subject)

	if !isVirtualMachineEvent(event.Subject) {
		ctx.Logger.Infof("Skipping non-VM event with subject: %s", event.Subject)
		return nil
	}

	var eventData ResourceWriteSuccessData
	if err := mapstructure.Decode(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	ctx.Logger.Infof("VM event data: status=%s operationName=%s resourceURI=%s", eventData.Status, eventData.OperationName, eventData.ResourceURI)

	// Azure Event Grid ResourceWriteSuccess events use the "status" field
	// (not "provisioningState") to indicate the outcome of the operation.
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

	ctx.Logger.Infof("VM write event: %s in resource group %s", vmName, resourceGroup)

	// Emit the full, unmodified Azure Event Grid event — same pattern as GitHub, GitLab, etc.
	if err := ctx.Events.Emit("azure.vm.write", rawEvent); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	ctx.Logger.Infof("Successfully emitted azure.vm.write event for VM: %s", vmName)
	return nil
}

// authenticateWebhook verifies the webhook secret if one is configured.
func (t *OnVMWriteTrigger) authenticateWebhook(ctx core.WebhookRequestContext) error {
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

func (t *OnVMWriteTrigger) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnVMWriteTrigger) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

// Cleanup is called when the trigger is removed.
func (t *OnVMWriteTrigger) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure VM Write trigger")
	return nil
}

// extractVMName returns VM name from ARM resource ID.
func extractVMName(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// extractResourceGroup returns resource group from ARM resource ID.
func extractResourceGroup(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "resourceGroups") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractSubscriptionID returns subscription ID from ARM resource ID.
func extractSubscriptionID(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "subscriptions") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// isVirtualMachineEvent reports whether an event subject targets a VM.
func isVirtualMachineEvent(subject string) bool {
	return strings.Contains(strings.ToLower(subject), strings.ToLower(ResourceTypeVirtualMachine))
}

// isSuccessfulStatus reports whether the event status indicates success.
func isSuccessfulStatus(status string) bool {
	return status == ProvisioningStateSucceeded
}
