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

type OnImagePushed struct {
	integration *AzureIntegration
}

type OnImagePushedConfiguration struct {
	ResourceGroup    string `json:"resourceGroup" mapstructure:"resourceGroup"`
	Registry         string `json:"registry" mapstructure:"registry"`
	RepositoryFilter string `json:"repositoryFilter" mapstructure:"repositoryFilter"`
	TagFilter        string `json:"tagFilter" mapstructure:"tagFilter"`
}

func (t *OnImagePushed) Name() string {
	return "azure.onContainerImagePushed"
}

func (t *OnImagePushed) Label() string {
	return "On Image Pushed"
}

func (t *OnImagePushed) Description() string {
	return "Listen to Azure Container Registry image push events"
}

func (t *OnImagePushed) Documentation() string {
	return `
The On Image Pushed trigger starts a workflow execution when a container image is pushed to an Azure Container Registry.

## Use Cases

- **CI/CD pipelines**: Trigger deployments when a new image version is pushed
- **Image scanning**: Kick off security scans when new images arrive
- **Notification workflows**: Notify teams when images are updated
- **Tag tracking**: React to specific image tags being published

## How It Works

This trigger listens to Azure Event Grid events from an ACR registry. When an image push succeeds,
the ` + "`Microsoft.ContainerRegistry.ImagePushed`" + ` event is delivered and the trigger fires with the full event payload.

## Configuration

- **Resource Group** (required): The resource group containing the ACR registry.
- **Registry** (required): The ACR registry to watch.
- **Repository Filter** (optional): A regex pattern to filter by repository name.
- **Tag Filter** (optional): A regex pattern to filter by image tag.

## Event Data

Each push event includes:

- **target.repository**: The repository name
- **target.tag**: The image tag
- **target.digest**: The image manifest digest
- **actor.name**: The user or service principal that pushed the image
- **request.host**: The registry hostname
`
}

func (t *OnImagePushed) Icon() string {
	return "azure"
}

func (t *OnImagePushed) Color() string {
	return "blue"
}

func (t *OnImagePushed) ExampleData() map[string]any {
	return map[string]any{
		"id":              "831e1650-001e-001b-66ab-eeb76e069631",
		"topic":           "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/my-rg/providers/Microsoft.ContainerRegistry/registries/myregistry",
		"subject":         "myregistry.azurecr.io/myrepository:v1.2.3",
		"eventType":       "Microsoft.ContainerRegistry.ImagePushed",
		"eventTime":       "2026-03-16T10:00:00Z",
		"dataVersion":     "1.0",
		"metadataVersion": "1",
		"data": map[string]any{
			"id":        "831e1650-001e-001b-66ab-eeb76e069631",
			"timestamp": "2026-03-16T10:00:00Z",
			"action":    "push",
			"target": map[string]any{
				"mediaType":  "application/vnd.docker.distribution.manifest.v2+json",
				"size":       1234567,
				"digest":     "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				"length":     1234567,
				"repository": "myrepository",
				"tag":        "v1.2.3",
				"url":        "https://myregistry.azurecr.io/v2/myrepository/manifests/sha256:abcdef1234567890",
			},
			"request": map[string]any{
				"id":        "6d6cef9a-a602-4a23-bc26-91bb68a2bf74",
				"addr":      "203.0.113.0:49926",
				"host":      "myregistry.azurecr.io",
				"method":    "PUT",
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

func (t *OnImagePushed) Configuration() []configuration.Field {
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
			Description: "The Azure Container Registry to watch for image push events",
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
		{
			Name:        "tagFilter",
			Label:       "Tag Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., v[0-9]+\\.[0-9]+\\.[0-9]+",
			Description: "Optional regex pattern to filter by image tag",
		},
	}
}

func (t *OnImagePushed) Setup(ctx core.TriggerContext) error {
	config := OnImagePushedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Registry == "" {
		return fmt.Errorf("registry is required")
	}

	// The registry field stores the ARM resource ID of the ACR instance.
	// Event Grid subscriptions for ACR must be scoped to the registry resource.
	scope := config.Registry

	err := ctx.Integration.RequestWebhook(AzureWebhookConfiguration{
		EventTypes: []string{EventTypeContainerImagePushed},
		Scope:      scope,
	})
	if err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	ctx.Logger.Info("Azure On Image Pushed trigger configured successfully")
	return nil
}

func (t *OnImagePushed) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := t.authenticateWebhook(ctx); err != nil {
		ctx.Logger.Warnf("Webhook authentication failed: %v", err)
		return http.StatusUnauthorized, nil, err
	}

	config := OnImagePushedConfiguration{}
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

		if event.EventType == EventTypeContainerImagePushed {
			if err := t.handleImagePushedEvent(ctx, event, rawEvents[i], config); err != nil {
				ctx.Logger.Errorf("Failed to process image pushed event: %v", err)
				continue
			}
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnImagePushed) handleSubscriptionValidation(ctx core.WebhookRequestContext, event EventGridEvent) (*core.WebhookResponseBody, error) {
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

func (t *OnImagePushed) handleImagePushedEvent(
	ctx core.WebhookRequestContext,
	event EventGridEvent,
	rawEvent map[string]any,
	config OnImagePushedConfiguration,
) error {
	var eventData ACREventData
	if err := mapstructure.Decode(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to parse ACR event data: %w", err)
	}

	repository := extractACRRepository(event.Subject)
	if eventData.Target != nil && eventData.Target.Repository != "" {
		repository = eventData.Target.Repository
	}

	tag := extractACRTag(event.Subject)
	if eventData.Target != nil && eventData.Target.Tag != "" {
		tag = eventData.Target.Tag
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

	if config.TagFilter != "" {
		matched, err := regexp.MatchString(config.TagFilter, tag)
		if err != nil {
			return fmt.Errorf("invalid tagFilter regex: %w", err)
		}
		if !matched {
			ctx.Logger.Debugf("Skipping image event for tag %s (filter: %s)", tag, config.TagFilter)
			return nil
		}
	}

	ctx.Logger.Infof("Image pushed: %s:%s", repository, tag)

	if err := ctx.Events.Emit("azure.image.pushed", rawEvent); err != nil {
		return fmt.Errorf("failed to emit event: %w", err)
	}

	ctx.Logger.Infof("Successfully emitted azure.image.pushed event for %s:%s", repository, tag)
	return nil
}

func (t *OnImagePushed) authenticateWebhook(ctx core.WebhookRequestContext) error {
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

func (t *OnImagePushed) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnImagePushed) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnImagePushed) Cleanup(ctx core.TriggerContext) error {
	ctx.Logger.Info("Cleaning up Azure On Image Pushed trigger")
	return nil
}
