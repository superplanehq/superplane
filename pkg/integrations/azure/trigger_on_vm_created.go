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

type OnVMCreatedTrigger struct{}

type OnVMCreatedConfiguration struct {
	ResourceGroup string `json:"resourceGroup" mapstructure:"resourceGroup"`
	NameFilter    string `json:"nameFilter" mapstructure:"nameFilter"`
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
- **VM Name Filter** (optional): A regex pattern to filter VMs by name. Only VMs whose name
  matches the pattern will trigger the workflow. Leave empty to trigger for all VM names.

## Event Data

Each VM creation event includes:

- **vmName**: The name of the created virtual machine
- **vmId**: The full Azure resource ID of the VM
- **resourceGroup**: The resource group containing the VM
- **subscriptionId**: The Azure subscription ID
- **location**: The Azure region where the VM was created
- **status**: The status of the operation (typically "Succeeded")
- **timestamp**: The timestamp when the event occurred

## Azure Event Grid Setup

Event Grid subscriptions are created automatically when the trigger is set up. SuperPlane will:

1. Create an Event Grid subscription at the Azure subscription scope
2. Configure it to forward ` + "`Microsoft.Resources.ResourceWriteSuccess`" + ` events to the trigger webhook
3. Apply subject filters based on the configured resource group and resource type
4. Handle the Event Grid validation handshake automatically

No manual setup is required.

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
		"vmName":         "my-vm-01",
		"vmId":           "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm-01",
		"resourceGroup":  "my-rg",
		"subscriptionId": "12345678-1234-1234-1234-123456789abc",
		"location":       "eastus",
		"timestamp":      "2026-02-11T10:30:00Z",
		"operationName":  "Microsoft.Compute/virtualMachines/write",
		"status":         "Succeeded",
	}
}

func (t *OnVMCreatedTrigger) Configuration() []configuration.Field {
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
func (t *OnVMCreatedTrigger) Setup(ctx core.TriggerContext) error {
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
	if err := t.authenticateWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, err
	}

	config := OnVMCreatedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

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

// handleSubscriptionValidation validates Event Grid subscription via the validationUrl GET approach.
func (t *OnVMCreatedTrigger) handleSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) error {
	var validationData SubscriptionValidationEventData
	if err := mapstructure.Decode(event.Data, &validationData); err != nil {
		return fmt.Errorf("failed to parse validation data: %w", err)
	}

	if validationData.ValidationCode == "" {
		return fmt.Errorf("validation code is empty")
	}

	ctx.Logger.Infof("Event Grid subscription validation received with code: %s", validationData.ValidationCode)

	if validationData.ValidationURL == "" {
		return fmt.Errorf("validation URL is empty; synchronous validation is not supported by the framework")
	}

	// Use the validationUrl GET approach since the framework's HandleWebhook
	// interface cannot return a custom response body for the synchronous handshake.
	ctx.Logger.Infof("Validating Event Grid subscription via validation URL")
	resp, err := http.Get(validationData.ValidationURL)
	if err != nil {
		return fmt.Errorf("failed to validate subscription via validation URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("validation URL returned status %d", resp.StatusCode)
	}

	ctx.Logger.Info("Event Grid subscription validated successfully via validation URL")
	return nil
}

// handleVMCreationEvent processes VM creation events.
func (t *OnVMCreatedTrigger) handleVMCreationEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	config OnVMCreatedConfiguration,
) error {
	if !isVirtualMachineEvent(event.Subject) {
		return nil
	}

	var eventData ResourceWriteSuccessData
	if err := mapstructure.Decode(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	// Azure Event Grid ResourceWriteSuccess events use the "status" field
	// (not "provisioningState") to indicate the outcome of the operation.
	if !isSuccessfulStatus(eventData.Status) {
		ctx.Logger.Infof("Skipping VM event with status: %s", eventData.Status)
		return nil
	}

	resourceGroup := extractResourceGroup(event.Subject)
	if resourceGroup == "" {
		ctx.Logger.Warnf("Could not extract resource group from subject: %s", event.Subject)
	}

	if config.ResourceGroup != "" && resourceGroup != config.ResourceGroup {
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

	payload := map[string]any{
		"vmName":         vmName,
		"vmId":           event.Subject,
		"resourceGroup":  resourceGroup,
		"subscriptionId": extractSubscriptionID(event.Subject),
		"location":       "",
		"timestamp":      event.EventTime,
		"operationName":  eventData.OperationName,
		"status":         eventData.Status,
	}

	ctx.Logger.Infof("VM created: %s in resource group %s", vmName, resourceGroup)

	if err := ctx.Events.Emit("azure.vm.created", payload); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	return nil
}

// authenticateWebhook verifies the webhook secret if one is configured.
func (t *OnVMCreatedTrigger) authenticateWebhook(ctx core.WebhookRequestContext) error {
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
		if part == "resourceGroups" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractSubscriptionID returns subscription ID from ARM resource ID.
func extractSubscriptionID(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if part == "subscriptions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// isVirtualMachineEvent reports whether an event subject targets a VM.
func isVirtualMachineEvent(subject string) bool {
	return strings.Contains(subject, ResourceTypeVirtualMachine)
}

// isSuccessfulStatus reports whether the event status indicates success.
func isSuccessfulStatus(status string) bool {
	return status == ProvisioningStateSucceeded
}
