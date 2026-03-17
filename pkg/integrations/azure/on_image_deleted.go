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

type OnImageDeleted struct {
	integration *AzureIntegration
}

type OnImageDeletedConfiguration struct {
	ResourceGroup    string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Registry         string `json:"registry" mapstructure:"registry"`
	RepositoryFilter string `json:"repositoryFilter" mapstructure:"repositoryFilter"`
}

func (t *OnImageDeleted) Name() string {
	return "azure.onContainerImageDeleted"
}

func (t *OnImageDeleted) Label() string {
	return "On Image Deleted"
}

func (t *OnImageDeleted) Description() string {
	return "Listen to Azure Container Registry image deletion events"
}

func (t *OnImageDeleted) Documentation() string {
	return `
The On Image Deleted trigger starts a workflow execution when a container image is deleted from an Azure Container Registry.

## Use Cases

- **Cleanup workflows**: Remove associated resources when an image is deleted
- **Audit trails**: Record image deletions for compliance purposes
- **Notification workflows**: Alert teams when images are removed from the registry

## How It Works

This trigger listens to Azure Event Grid events from an ACR registry. When an image or manifest is deleted,
the ` + "`Microsoft.ContainerRegistry.ImageDeleted`" + ` event is delivered and the trigger fires with the full event payload.

Note: Image deletions reference manifests by digest. Tags may be empty if the manifest itself was deleted.

## Configuration

- **Resource Group** (required): The resource group containing the ACR registry.
- **Registry** (required): The ACR registry to watch.
- **Repository Filter** (optional): A regex pattern to filter by repository name.

## Event Data

Each delete event includes:

- **target.repository**: The repository name
- **target.digest**: The manifest digest that was deleted
- **target.tag**: The tag (may be empty for manifest deletes)
- **actor.name**: The user or service principal that deleted the image
`
}

func (t *OnImageDeleted) Icon() string {
	return "azure"
}

func (t *OnImageDeleted) Color() string {
	return "blue"
}

func (t *OnImageDeleted) ExampleData() map[string]any {
	return map[string]any{
		"id":              "afc359b4-001e-001b-66ab-eeb76e069631",
		"topic":           "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.ContainerRegistry/registries/myregistry",
		"subject":         "myregistry.azurecr.io/myrepository@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"eventType":       "Microsoft.ContainerRegistry.ImageDeleted",
		"eventTime":       "2026-03-16T11:00:00Z",
		"dataVersion":     "1.0",
		"metadataVersion": "1",
		"data": map[string]any{
			"id":        "afc359b4-001e-001b-66ab-eeb76e069631",
			"timestamp": "2026-03-16T11:00:00Z",
			"action":    "delete",
			"target": map[string]any{
				"mediaType":  "application/vnd.docker.distribution.manifest.v2+json",
				"digest":     "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				"repository": "myrepository",
				"tag":        "",
				"url":        "https://myregistry.azurecr.io/v2/myrepository/manifests/sha256:abcdef1234567890",
			},
			"request": map[string]any{
				"id":        "6d6cef9a-a602-4a23-bc26-91bb68a2bf74",
				"addr":      "203.0.113.0:49926",
				"host":      "myregistry.azurecr.io",
				"method":    "DELETE",
				"useragent": "docker/20.10.7",
			},
			"actor": map[string]any{
				"name": "myuser",
			},
			"source": map[string]any{
				"addr":       "myregistry.azurecr.io",
				"instanceID": "a29a591f-f89c-4f8d-b061-3c5d73d4756c",
			},
		},
	}
}

func (t *OnImageDeleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceGroup",
			Label:       "Resource Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The resource group containing the Azure Container Registry",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeResourceGroupDropdown,
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "registry",
			Label:       "Registry",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Azure Container Registry to watch for image deletion events",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeContainerRegistryDropdown,
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
			Name:        "repositoryFilter",
			Label:       "Repository Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., myapp/.*",
			Description: "Optional regex pattern to filter by repository name",
		},
	}
}

func (t *OnImageDeleted) Setup(ctx core.TriggerContext) error {
	config := OnImageDeletedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Registry == "" {
		return fmt.Errorf("registry is required")
	}

	scope := config.Registry

	err := ctx.Integration.RequestWebhook(AzureWebhookConfiguration{
		EventTypes: []string{EventTypeContainerImageDeleted},
		Scope:      scope,
	})
	if err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	ctx.Logger.Info("Azure On Image Deleted trigger configured successfully")
	return nil
}

func (t *OnImageDeleted) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := t.authenticateWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, nil, err
	}

	config := OnImageDeletedConfiguration{}
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

		if event.EventType == EventTypeContainerImageDeleted {
			if err := t.handleImageDeletedEvent(ctx, event, rawEvents[i], config); err != nil {
				ctx.Logger.Errorf("Failed to process image deleted event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnImageDeleted) handleSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) (*core.WebhookResponseBody, error) {
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

func (t *OnImageDeleted) handleImageDeletedEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	rawEvent map[string]any,
	config OnImageDeletedConfiguration,
) error {
	var eventData ACREventData
	if err := mapstructure.Decode(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to parse ACR event data: %w", err)
	}

	repository := extractACRRepository(event.Subject)
	if eventData.Target != nil && eventData.Target.Repository != "" {
		repository = eventData.Target.Repository
	}

	digest := ""
	if eventData.Target != nil {
		digest = eventData.Target.Digest
	}

	if config.RepositoryFilter != "" {
		matched, err := regexp.MatchString(config.RepositoryFilter, repository)
		if err != nil {
			return fmt.Errorf("invalid repositoryFilter regex: %w", err)
		}
		if !matched {
			ctx.Logger.Debugf("Skipping image event for repository %s (filter: %s)", repository, config.RepositoryFilter)
			return nil
		}
	}

	ctx.Logger.Infof("Image deleted: %s@%s", repository, digest)

	if err := ctx.Events.Emit("azure.image.deleted", rawEvent); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	ctx.Logger.Infof("Successfully emitted azure.image.deleted event for %s@%s", repository, digest)
	return nil
}

func (t *OnImageDeleted) authenticateWebhook(ctx core.WebhookRequestContext) error {
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

func (t *OnImageDeleted) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnImageDeleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnImageDeleted) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure On Image Deleted trigger")
	return nil
}
