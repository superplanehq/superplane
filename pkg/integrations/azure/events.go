package azure

import (
	"time"
)

// EventGridEvent represents the top-level structure of an Azure Event Grid event
// See: https://learn.microsoft.com/en-us/azure/event-grid/event-schema
type EventGridEvent struct {
	// ID is the unique identifier for the event
	ID string `json:"id"`

	// Topic is the full resource path to the event source
	Topic string `json:"topic"`

	// Subject is the publisher-defined path to the event subject
	Subject string `json:"subject"`

	// EventType is the one of the registered event types for this event source
	EventType string `json:"eventType"`

	// EventTime is the time the event is generated based on the provider's UTC time
	EventTime time.Time `json:"eventTime"`

	// Data contains the event data specific to the event type
	Data map[string]any `json:"data"`

	// DataVersion is the schema version of the data object
	DataVersion string `json:"dataVersion"`

	// MetadataVersion is the schema version of the event metadata
	MetadataVersion string `json:"metadataVersion"`
}

// SubscriptionValidationEventData contains the validation code for Event Grid subscription validation
// This is sent when Azure Event Grid first subscribes to a webhook endpoint
type SubscriptionValidationEventData struct {
	// ValidationCode is the code that must be returned to complete the handshake
	ValidationCode string `json:"validationCode"`

	// ValidationURL is an optional URL that can be used instead of returning the code
	ValidationURL string `json:"validationUrl,omitempty"`
}

// SubscriptionValidationResponse is the response sent back to Azure Event Grid
// to complete the subscription validation handshake
type SubscriptionValidationResponse struct {
	// ValidationResponse contains the validation code from the subscription validation event
	ValidationResponse string `json:"validationResponse"`
}

// ResourceWriteSuccessData contains the data for a successful resource write operation
// This is used for events like VM creation, update, or deletion
type ResourceWriteSuccessData struct {
	// ProvisioningState indicates the current state of the resource operation
	// Common values: "Succeeded", "Failed", "Canceled", "Creating", "Updating", "Deleting"
	ProvisioningState string `json:"provisioningState"`

	// ResourceProvider is the Azure resource provider (e.g., "Microsoft.Compute")
	ResourceProvider string `json:"resourceProvider"`

	// ResourceURI is the full resource ID
	ResourceURI string `json:"resourceUri"`

	// OperationName is the name of the operation that was performed
	OperationName string `json:"operationName"`

	// Status indicates the HTTP status of the operation
	Status string `json:"status"`

	// SubscriptionID is the Azure subscription ID
	SubscriptionID string `json:"subscriptionId"`

	// TenantID is the Azure tenant ID
	TenantID string `json:"tenantId"`

	// Authorization contains information about the authorization for the operation
	Authorization *AuthorizationInfo `json:"authorization,omitempty"`

	// Claims contains the JWT claims from the request
	Claims map[string]any `json:"claims,omitempty"`

	// CorrelationID is the correlation ID for the operation
	CorrelationID string `json:"correlationId"`

	// HTTPRequest contains information about the HTTP request that triggered the operation
	HTTPRequest *HTTPRequestInfo `json:"httpRequest,omitempty"`
}

// AuthorizationInfo contains authorization information for a resource operation
type AuthorizationInfo struct {
	// Scope is the scope of the authorization
	Scope string `json:"scope"`

	// Action is the action that was authorized
	Action string `json:"action"`

	// Evidence contains additional evidence for the authorization
	Evidence map[string]any `json:"evidence,omitempty"`
}

// HTTPRequestInfo contains information about the HTTP request that triggered an event
type HTTPRequestInfo struct {
	// ClientRequestID is the client request ID
	ClientRequestID string `json:"clientRequestId"`

	// ClientIPAddress is the IP address of the client
	ClientIPAddress string `json:"clientIpAddress"`

	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string `json:"method"`

	// URL is the request URL
	URL string `json:"url"`
}

// Event type constants for Azure Event Grid
const (
	// EventTypeSubscriptionValidation is sent when Event Grid first subscribes to a webhook
	EventTypeSubscriptionValidation = "Microsoft.EventGrid.SubscriptionValidationEvent"

	// EventTypeResourceWriteSuccess is sent when a resource write operation succeeds
	EventTypeResourceWriteSuccess = "Microsoft.Resources.ResourceWriteSuccess"

	// EventTypeResourceWriteFailure is sent when a resource write operation fails
	EventTypeResourceWriteFailure = "Microsoft.Resources.ResourceWriteFailure"

	// EventTypeResourceWriteCancel is sent when a resource write operation is canceled
	EventTypeResourceWriteCancel = "Microsoft.Resources.ResourceWriteCancel"

	// EventTypeResourceDeleteSuccess is sent when a resource delete operation succeeds
	EventTypeResourceDeleteSuccess = "Microsoft.Resources.ResourceDeleteSuccess"

	// EventTypeResourceDeleteFailure is sent when a resource delete operation fails
	EventTypeResourceDeleteFailure = "Microsoft.Resources.ResourceDeleteFailure"

	// EventTypeResourceDeleteCancel is sent when a resource delete operation is canceled
	EventTypeResourceDeleteCancel = "Microsoft.Resources.ResourceDeleteCancel"

	// EventTypeResourceActionSuccess is sent when a resource action succeeds
	EventTypeResourceActionSuccess = "Microsoft.Resources.ResourceActionSuccess"

	// EventTypeResourceActionFailure is sent when a resource action fails
	EventTypeResourceActionFailure = "Microsoft.Resources.ResourceActionFailure"

	// EventTypeResourceActionCancel is sent when a resource action is canceled
	EventTypeResourceActionCancel = "Microsoft.Resources.ResourceActionCancel"
)

// Resource type constants
const (
	// ResourceTypeVirtualMachine is the resource type for Azure Virtual Machines
	ResourceTypeVirtualMachine = "Microsoft.Compute/virtualMachines"

	// ResourceTypeStorageAccount is the resource type for Azure Storage Accounts
	ResourceTypeStorageAccount = "Microsoft.Storage/storageAccounts"

	// ResourceTypeNetworkInterface is the resource type for Azure Network Interfaces
	ResourceTypeNetworkInterface = "Microsoft.Network/networkInterfaces"

	// ResourceTypeVirtualNetwork is the resource type for Azure Virtual Networks
	ResourceTypeVirtualNetwork = "Microsoft.Network/virtualNetworks"

	// ResourceTypePublicIPAddress is the resource type for Azure Public IP Addresses
	ResourceTypePublicIPAddress = "Microsoft.Network/publicIPAddresses"
)

// Provisioning state constants
const (
	// ProvisioningStateSucceeded indicates the resource operation completed successfully
	ProvisioningStateSucceeded = "Succeeded"

	// ProvisioningStateFailed indicates the resource operation failed
	ProvisioningStateFailed = "Failed"

	// ProvisioningStateCanceled indicates the resource operation was canceled
	ProvisioningStateCanceled = "Canceled"

	// ProvisioningStateCreating indicates the resource is being created
	ProvisioningStateCreating = "Creating"

	// ProvisioningStateUpdating indicates the resource is being updated
	ProvisioningStateUpdating = "Updating"

	// ProvisioningStateDeleting indicates the resource is being deleted
	ProvisioningStateDeleting = "Deleting"
)
