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

type OnBlobDeleted struct {
	integration *AzureIntegration
}

type OnBlobDeletedConfiguration struct {
	ResourceGroup   string `json:"resourceGroup" mapstructure:"resourceGroup"`
	StorageAccount  string `json:"storageAccount" mapstructure:"storageAccount"`
	ContainerFilter string `json:"containerFilter" mapstructure:"containerFilter"`
	BlobFilter      string `json:"blobFilter" mapstructure:"blobFilter"`
}

func (t *OnBlobDeleted) Name() string {
	return "azure.onBlobDeleted"
}

func (t *OnBlobDeleted) Label() string {
	return "On Blob Deleted"
}

func (t *OnBlobDeleted) Description() string {
	return "Listen to Azure Blob Storage blob deletion events"
}

func (t *OnBlobDeleted) Documentation() string {
	return `
The On Blob Deleted trigger starts a workflow execution when a blob is deleted from an Azure Storage Account.

## Use Cases

- **Cleanup workflows**: Remove associated resources or records when a blob is deleted
- **Audit and compliance**: Record blob deletions for traceability
- **Notification workflows**: Alert teams when important files are removed from storage

## How It Works

This trigger listens to Azure Event Grid events from a Storage Account. When a blob is deleted,
the ` + "`Microsoft.Storage.BlobDeleted`" + ` event is delivered and the trigger fires with the full event payload.

## Configuration

- **Resource Group** (required): The resource group containing the Storage Account.
- **Storage Account** (required): The Storage Account to watch.
- **Container Filter** (optional): A regex pattern to filter by container name.
- **Blob Filter** (optional): A regex pattern to filter by blob path.

## Event Data

Each blob deleted event includes:

- **subject**: The full blob path in the format /blobServices/default/containers/{container}/blobs/{blob}
- **data.api**: The operation that triggered the event (e.g., DeleteBlob)
- **data.blobType**: The blob type (BlockBlob, PageBlob, AppendBlob)
- **data.url**: The URL of the deleted blob
`
}

func (t *OnBlobDeleted) Icon() string {
	return "azure"
}

func (t *OnBlobDeleted) Color() string {
	return "blue"
}

func (t *OnBlobDeleted) ExampleData() map[string]any {
	return map[string]any{
		"id":              "afc359b4-001e-001b-66ab-eeb76e069631",
		"topic":           "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Storage/storageAccounts/mystorageaccount",
		"subject":         "/blobServices/default/containers/mycontainer/blobs/path/to/myfile.csv",
		"eventType":       "Microsoft.Storage.BlobDeleted",
		"eventTime":       "2026-03-16T11:00:00Z",
		"dataVersion":     "",
		"metadataVersion": "1",
		"data": map[string]any{
			"api":             "DeleteBlob",
			"clientRequestId": "6d6cef9a-a602-4a23-bc26-91bb68a2bf74",
			"requestId":       "d1e6b5a4-0001-0035-4a7b-2e5c4f000000",
			"eTag":            "0x8D4BCC2E4835CD0",
			"contentType":     "text/csv",
			"contentLength":   0,
			"blobType":        "BlockBlob",
			"url":             "https://mystorageaccount.blob.core.windows.net/mycontainer/path/to/myfile.csv",
			"sequencer":       "00000000000004420000000000028964",
		},
	}
}

func (t *OnBlobDeleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The resource group containing the Azure Storage Account",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeResourceGroupDropdown,
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "storageAccount",
			Label:       "Storage Account",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Azure Storage Account to watch for blob deletion events",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeStorageAccountDropdown,
					UseNameAsValue: false,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "resourceGroup",
							ValueFrom: &configuration.ParameterValueFrom{Field: "resourceGroup"},
						},
					},
				},
			},
		},
		{
			Name:        "containerFilter",
			Label:       "Container Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., uploads",
			Description: "Optional regex pattern to filter by container name",
		},
		{
			Name:        "blobFilter",
			Label:       "Blob Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., data/.*\\.csv",
			Description: "Optional regex pattern to filter by blob path",
		},
	}
}

func (t *OnBlobDeleted) Setup(ctx core.TriggerContext) error {
	config := OnBlobDeletedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.StorageAccount == "" {
		return fmt.Errorf("storageAccount is required")
	}

	err := ctx.Integration.RequestWebhook(AzureWebhookConfiguration{
		EventTypes: []string{EventTypeBlobDeleted},
		Scope:      config.StorageAccount,
	})
	if err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	ctx.Logger.Info("Azure On Blob Deleted trigger configured successfully")
	return nil
}

func (t *OnBlobDeleted) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := t.authenticateWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, nil, err
	}

	config := OnBlobDeletedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var events []EventGridEvent
	if err := json.Unmarshal(ctx.Body, &events); err != nil {
		ctx.Logger.Errorf("Failed to parse Event Grid events: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse events: %w", err)
	}

	var rawEvents []map[string]any
	if err := json.Unmarshal(ctx.Body, &rawEvents); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse raw events: %w", err)
	}

	ctx.Logger.Infof("Received %d Event Grid event(s)", len(events))

	for i, event := range events {
		ctx.Logger.Infof("Event[%d]: id=%s type=%s subject=%s", i, event.ID, event.EventType, event.Subject)

		if event.EventType == EventTypeSubscriptionValidation {
			resp, err := t.handleSubscriptionValidation(ctx, event)
			if err != nil {
				return http.StatusInternalServerError, nil, err
			}
			return http.StatusOK, resp, nil
		}

		if event.EventType == EventTypeBlobDeleted {
			if err := t.handleBlobDeletedEvent(ctx, event, rawEvents[i], config); err != nil {
				ctx.Logger.Errorf("Failed to process blob deleted event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnBlobDeleted) handleSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) (*core.WebhookResponseBody, error) {
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

	return &core.WebhookResponseBody{Body: body, ContentType: "application/json"}, nil
}

func (t *OnBlobDeleted) handleBlobDeletedEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	rawEvent map[string]any,
	config OnBlobDeletedConfiguration,
) error {
	container := extractBlobContainer(event.Subject)
	blobName := extractBlobName(event.Subject)

	if config.ContainerFilter != "" {
		matched, err := regexp.MatchString(config.ContainerFilter, container)
		if err != nil {
			return fmt.Errorf("invalid containerFilter regex: %w", err)
		}
		if !matched {
			ctx.Logger.Debugf("Skipping blob event for container %s (filter: %s)", container, config.ContainerFilter)
			return nil
		}
	}

	if config.BlobFilter != "" {
		matched, err := regexp.MatchString(config.BlobFilter, blobName)
		if err != nil {
			return fmt.Errorf("invalid blobFilter regex: %w", err)
		}
		if !matched {
			ctx.Logger.Debugf("Skipping blob event for blob %s (filter: %s)", blobName, config.BlobFilter)
			return nil
		}
	}

	ctx.Logger.Infof("Blob deleted: %s/%s", container, blobName)

	if err := ctx.Events.Emit("azure.blob.deleted", rawEvent); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	ctx.Logger.Infof("Successfully emitted azure.blob.deleted event for %s/%s", container, blobName)
	return nil
}

func (t *OnBlobDeleted) authenticateWebhook(ctx core.WebhookRequestContext) error {
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

	authHeader := ctx.Headers.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		providedSecret := strings.TrimPrefix(authHeader, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(providedSecret), secret) == 1 {
			return nil
		}
		return fmt.Errorf("invalid webhook secret")
	}

	secretHeader := ctx.Headers.Get("X-Webhook-Secret")
	if secretHeader != "" {
		if subtle.ConstantTimeCompare([]byte(secretHeader), secret) == 1 {
			return nil
		}
		return fmt.Errorf("invalid webhook secret")
	}

	return fmt.Errorf("webhook secret required but not provided in Authorization or X-Webhook-Secret header")
}

func (t *OnBlobDeleted) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnBlobDeleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnBlobDeleted) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure On Blob Deleted trigger")
	return nil
}
