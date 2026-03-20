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

type OnBlobCreated struct {
	integration *AzureIntegration
}

type OnBlobCreatedConfiguration struct {
	ResourceGroup   string `json:"resourceGroup" mapstructure:"resourceGroup"`
	StorageAccount  string `json:"storageAccount" mapstructure:"storageAccount"`
	ContainerFilter string `json:"containerFilter" mapstructure:"containerFilter"`
	BlobFilter      string `json:"blobFilter" mapstructure:"blobFilter"`
}

func (t *OnBlobCreated) Name() string {
	return "azure.onBlobCreated"
}

func (t *OnBlobCreated) Label() string {
	return "On Blob Created"
}

func (t *OnBlobCreated) Description() string {
	return "Listen to Azure Blob Storage blob creation events"
}

func (t *OnBlobCreated) Documentation() string {
	return `
The On Blob Created trigger starts a workflow execution when a blob is created or replaced in an Azure Storage Account.

## Use Cases

- **Data pipelines**: Trigger processing when new files arrive in a storage container
- **Image processing**: React to new images or media uploaded to blob storage
- **Audit and compliance**: Record blob creation events for traceability
- **ETL workflows**: Kick off data transformation when input files are uploaded

## How It Works

This trigger listens to Azure Event Grid events from a Storage Account. When a blob is created or replaced,
the ` + "`Microsoft.Storage.BlobCreated`" + ` event is delivered and the trigger fires with the full event payload.

## Configuration

- **Resource Group** (required): The resource group containing the Storage Account.
- **Storage Account** (required): The Storage Account to watch.
- **Container Filter** (optional): A regex pattern to filter by container name.
- **Blob Filter** (optional): A regex pattern to filter by blob path.

## Event Data

Each blob created event includes:

- **subject**: The full blob path in the format /blobServices/default/containers/{container}/blobs/{blob}
- **data.api**: The operation that triggered the event (e.g., PutBlob, CopyBlob)
- **data.contentType**: The content type of the blob
- **data.contentLength**: The size of the blob in bytes
- **data.blobType**: The blob type (BlockBlob, PageBlob, AppendBlob)
- **data.url**: The URL of the blob
`
}

func (t *OnBlobCreated) Icon() string {
	return "azure"
}

func (t *OnBlobCreated) Color() string {
	return "blue"
}

func (t *OnBlobCreated) ExampleData() map[string]any {
	return map[string]any{
		"id":              "831e1650-001e-001b-66ab-eeb76e069631",
		"topic":           "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.Storage/storageAccounts/mystorageaccount",
		"subject":         "/blobServices/default/containers/mycontainer/blobs/path/to/myfile.csv",
		"eventType":       "Microsoft.Storage.BlobCreated",
		"eventTime":       "2026-03-16T10:00:00Z",
		"dataVersion":     "",
		"metadataVersion": "1",
		"data": map[string]any{
			"api":             "PutBlob",
			"clientRequestId": "6d6cef9a-a602-4a23-bc26-91bb68a2bf74",
			"requestId":       "d1e6b5a4-0001-0035-4a7b-2e5c4f000000",
			"eTag":            "0x8D4BCC2E4835CD0",
			"contentType":     "text/csv",
			"contentLength":   524288,
			"blobType":        "BlockBlob",
			"url":             "https://mystorageaccount.blob.core.windows.net/mycontainer/path/to/myfile.csv",
			"sequencer":       "00000000000004420000000000028963",
		},
	}
}

func (t *OnBlobCreated) Configuration() []configuration.Field {
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
			Description: "The Azure Storage Account to watch for blob creation events",
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

func (t *OnBlobCreated) Setup(ctx core.TriggerContext) error {
	config := OnBlobCreatedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.StorageAccount == "" {
		return fmt.Errorf("storageAccount is required")
	}

	err := ctx.Integration.RequestWebhook(AzureWebhookConfiguration{
		EventTypes: []string{EventTypeBlobCreated},
		Scope:      config.StorageAccount,
	})
	if err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	ctx.Logger.Info("Azure On Blob Created trigger configured successfully")
	return nil
}

func (t *OnBlobCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := t.authenticateWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, nil, err
	}

	config := OnBlobCreatedConfiguration{}
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

		if event.EventType == EventTypeBlobCreated {
			if err := t.handleBlobCreatedEvent(ctx, event, rawEvents[i], config); err != nil {
				ctx.Logger.Errorf("Failed to process blob created event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnBlobCreated) handleSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) (*core.WebhookResponseBody, error) {
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

func (t *OnBlobCreated) handleBlobCreatedEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	rawEvent map[string]any,
	config OnBlobCreatedConfiguration,
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

	ctx.Logger.Infof("Blob created: %s/%s", container, blobName)

	if err := ctx.Events.Emit("azure.blob.created", rawEvent); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	ctx.Logger.Infof("Successfully emitted azure.blob.created event for %s/%s", container, blobName)
	return nil
}

func (t *OnBlobCreated) authenticateWebhook(ctx core.WebhookRequestContext) error {
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

func (t *OnBlobCreated) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnBlobCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnBlobCreated) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure On Blob Created trigger")
	return nil
}
