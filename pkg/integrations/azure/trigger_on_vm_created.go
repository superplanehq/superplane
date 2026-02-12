package azure

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnVMCreatedTrigger struct{}

type OnVMCreatedConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
}

func (t *OnVMCreatedTrigger) Name() string {
	return "azure.onVirtualMachineCreated"
}

func (t *OnVMCreatedTrigger) Label() string {
	return "Azure • On VM Created"
}

func (t *OnVMCreatedTrigger) Description() string {
	return "Triggers when a new Virtual Machine is successfully provisioned in Azure"
}

func (t *OnVMCreatedTrigger) Documentation() string {
	return `
The On VM Created trigger starts a workflow execution when a new Azure Virtual Machine is successfully provisioned.

## Use Cases

- **Automated configuration**: Run configuration scripts on newly created VMs
- **Compliance checks**: Verify that new VMs meet security and compliance requirements
- **Inventory tracking**: Update external inventory systems when VMs are created
- **Notification workflows**: Send notifications to teams when new VMs are provisioned
- **Cost tracking**: Log VM creation events for cost analysis and reporting

## How It Works

This trigger listens to Azure Event Grid events for Virtual Machine resource write operations.
When a VM is successfully created (` + "`provisioningState: Succeeded`" + `), the trigger fires and
provides detailed information about the new VM.

## Configuration

- **Resource Group** (optional): Filter events to only trigger for VMs created in a specific
  resource group. Leave empty to trigger for all resource groups in the subscription.

## Event Data

Each VM creation event includes:

- **vmName**: The name of the created virtual machine
- **vmId**: The full Azure resource ID of the VM
- **resourceGroup**: The resource group containing the VM
- **subscriptionId**: The Azure subscription ID
- **location**: The Azure region where the VM was created
- **provisioningState**: The provisioning state (typically "Succeeded")
- **timestamp**: The timestamp when the event occurred

## Azure Event Grid Setup

**Important**: This trigger requires manual setup of an Azure Event Grid subscription.

1. **Create an Event Grid System Topic** (if not already created):
   - Go to Azure Portal → Event Grid System Topics
   - Create a new topic for your subscription
   - Topic Type: "Azure Subscriptions"
   - Select your subscription

2. **Create an Event Subscription**:
   - In your Event Grid System Topic, create a new Event Subscription
   - **Event Types**: Select "Resource Write Success"
   - **Filters**: 
     - Subject begins with: ` + "`/subscriptions/<subscription-id>/resourceGroups/`" + `
     - Subject ends with: ` + "`/providers/Microsoft.Compute/virtualMachines/`" + `
   - **Endpoint Type**: Webhook
   - **Endpoint**: Use the webhook URL provided by SuperPlane for this trigger node

3. **Validation**: Azure Event Grid will send a validation event to verify the endpoint.
   SuperPlane will automatically respond to this validation request.

## Notes

- The trigger only fires for successfully provisioned VMs (` + "`provisioningState: Succeeded`" + `)
- Failed VM creations do not trigger the workflow
- The trigger processes events from Azure Event Grid in real-time
- Multiple triggers can share the same Event Grid subscription if configured correctly
`
}

func (t *OnVMCreatedTrigger) Icon() string {
	return "azure"
}

func (t *OnVMCreatedTrigger) Color() string {
	return "blue"
}

func (t *OnVMCreatedTrigger) ExampleData() map[string]any {
	return map[string]any{
		"vmName":            "my-vm-01",
		"vmId":              "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm-01",
		"resourceGroup":     "my-rg",
		"subscriptionId":    "12345678-1234-1234-1234-123456789abc",
		"location":          "eastus",
		"provisioningState": "Succeeded",
		"timestamp":         "2026-02-11T10:30:00Z",
		"operationName":     "Microsoft.Compute/virtualMachines/write",
	}
}

func (t *OnVMCreatedTrigger) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter events to a specific resource group (optional - leave empty for all resource groups)",
			Placeholder: "my-resource-group",
		},
	}
}

// Setup configures trigger webhooks.
func (t *OnVMCreatedTrigger) Setup(ctx core.TriggerContext) error {
	// Decode configuration
	config := OnVMCreatedConfiguration{}
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

	ctx.Logger.Info("Azure VM Created trigger configured successfully")
	if config.ResourceGroup != "" {
		ctx.Logger.Infof("Filtering events for resource group: %s", config.ResourceGroup)
	} else {
		ctx.Logger.Info("Listening for VM creation events in all resource groups")
	}

	return nil
}

// HandleWebhook processes Event Grid webhook requests.
func (t *OnVMCreatedTrigger) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// Decode configuration
	config := OnVMCreatedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Parse Event Grid events
	var events []EventGridEvent
	if err := json.Unmarshal(ctx.Body, &events); err != nil {
		ctx.Logger.Errorf("Failed to parse Event Grid events: %v", err)
		return http.StatusBadRequest, fmt.Errorf("failed to parse events: %w", err)
	}

	ctx.Logger.Infof("Received %d Event Grid event(s)", len(events))

	for _, event := range events {
		if event.EventType == EventTypeSubscriptionValidation {
			if err := t.handleSubscriptionValidation(ctx, event); err != nil {
				return http.StatusInternalServerError, err
			}
			return http.StatusOK, nil
		}

		if event.EventType == EventTypeResourceWriteSuccess {
			if err := t.handleVMCreationEvent(ctx, event, config); err != nil {
				ctx.Logger.Errorf("Failed to process VM creation event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil
}

// handleSubscriptionValidation validates Event Grid subscription setup.
func (t *OnVMCreatedTrigger) handleSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) error {
	var validationData SubscriptionValidationEventData
	if err := mapstructure.Decode(event.Data, &validationData); err != nil {
		return fmt.Errorf("failed to parse validation data: %w", err)
	}

	ctx.Logger.Infof("Responding to Event Grid subscription validation with code: %s", validationData.ValidationCode)

	return nil
}

// handleVMCreationEvent processes VM creation events.
func (t *OnVMCreatedTrigger) handleVMCreationEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	config OnVMCreatedConfiguration,
) error {
	if !strings.Contains(event.Subject, ResourceTypeVirtualMachine) {
		return nil
	}

	var eventData ResourceWriteSuccessData
	if err := mapstructure.Decode(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	if eventData.ProvisioningState != ProvisioningStateSucceeded {
		ctx.Logger.Infof("Skipping VM event with provisioning state: %s", eventData.ProvisioningState)
		return nil
	}

	resourceGroup := ""
	parts := strings.Split(event.Subject, "/")
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			resourceGroup = parts[i+1]
			break
		}
	}

	if config.ResourceGroup != "" && resourceGroup != config.ResourceGroup {
		ctx.Logger.Debugf("Skipping VM event for resource group %s (filter: %s)", resourceGroup, config.ResourceGroup)
		return nil
	}

	vmName := ""
	if len(parts) > 0 {
		vmName = parts[len(parts)-1]
	}

	payload := map[string]any{
		"vmName":            vmName,
		"vmId":              event.Subject,
		"resourceGroup":     resourceGroup,
		"subscriptionId":    eventData.SubscriptionID,
		"location":          "",
		"provisioningState": eventData.ProvisioningState,
		"timestamp":         event.EventTime,
		"operationName":     eventData.OperationName,
		"status":            eventData.Status,
	}

	ctx.Logger.Infof("VM created: %s in resource group %s", vmName, resourceGroup)

	if err := ctx.Events.Emit("azure.vm.created", payload); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	return nil
}

func (t *OnVMCreatedTrigger) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnVMCreatedTrigger) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

// Cleanup is called when the trigger is removed.
func (t *OnVMCreatedTrigger) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure VM Created trigger")
	return nil
}
