package artifactregistry

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ArtifactPushEmittedEventType = "gcp.artifactregistry.artifact.push"
	ArtifactPushSubscriptionType = "artifactregistry.push"

	artifactPushActionInsert = "INSERT"
)

type OnArtifactPush struct{}

type OnArtifactPushConfiguration struct {
	Location   string `json:"location" mapstructure:"location"`
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnArtifactPushMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

func (t *OnArtifactPush) Name() string {
	return "gcp.artifactregistry.onArtifactPush"
}

func (t *OnArtifactPush) Label() string {
	return "Artifact Registry • On Artifact Push"
}

func (t *OnArtifactPush) Description() string {
	return "Trigger a workflow when an artifact is pushed to GCP Artifact Registry"
}

func (t *OnArtifactPush) Documentation() string {
	return `The On Artifact Push trigger starts a workflow execution when a Docker image or other container artifact is pushed to Artifact Registry.

**Trigger behavior:** SuperPlane subscribes to the ` + "`gcr`" + ` Pub/Sub topic that Artifact Registry automatically publishes to for container image push events.

## Use Cases

- **Post-push automation**: Trigger vulnerability scans, deployments, or notifications when a new image is pushed
- **Release workflows**: Promote artifacts through environments when a new tag is published
- **Security automation**: Kick off container analysis on every new push

## Setup

**Required GCP setup:** Ensure the **Artifact Registry API** and **Pub/Sub API** are enabled in your project. The service account must have ` + "`roles/pubsub.admin`" + ` so SuperPlane can create the push subscription.

## Configuration

- **Location**: Optional filter by Artifact Registry location. Leave empty to receive events for all locations.
- **Repository**: Optional filter by repository name. Leave empty to receive events for all repositories.

## Event Data

Each event contains:
- ` + "`action`" + `: Always ` + "`INSERT`" + ` for pushes
- ` + "`digest`" + `: Full image digest URI (e.g. ` + "`us-central1-docker.pkg.dev/project/repo/image@sha256:abc`" + `)
- ` + "`tag`" + `: Full image tag URI (e.g. ` + "`us-central1-docker.pkg.dev/project/repo/image:latest`" + `)`
}

func (t *OnArtifactPush) Icon() string  { return "gcp" }
func (t *OnArtifactPush) Color() string { return "gray" }

func (t *OnArtifactPush) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional filter by Artifact Registry location. Leave empty to receive events for all locations.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeLocation,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional filter by repository name. Leave empty to receive events for all repositories.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRepository,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
					},
				},
			},
		},
	}
}

func (t *OnArtifactPush) Setup(ctx core.TriggerContext) error {
	if _, err := decodeOnArtifactPushConfiguration(ctx.Configuration); err != nil {
		return err
	}

	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing")
	}

	if err := scheduleArtifactRegistrySetupIfNeeded(ctx.Integration); err != nil {
		return err
	}

	subscriptionID, err := ctx.Integration.Subscribe(map[string]any{"type": ArtifactPushSubscriptionType})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnArtifactPushMetadata{
		SubscriptionID: subscriptionID.String(),
	})
}

func (t *OnArtifactPush) Actions() []core.Action {
	return nil
}

func (t *OnArtifactPush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnArtifactPush) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config, err := decodeOnArtifactPushConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	var pushEvent struct {
		Action string `mapstructure:"action"`
		Digest string `mapstructure:"digest"`
		Tag    string `mapstructure:"tag"`
	}
	if err := mapstructure.Decode(ctx.Message, &pushEvent); err != nil {
		return fmt.Errorf("failed to decode push event: %w", err)
	}

	if strings.ToUpper(pushEvent.Action) != artifactPushActionInsert {
		ctx.Logger.Infof("gcp artifact registry: action %q is not a push, skipping", pushEvent.Action)
		return nil
	}

	if config.Location != "" || config.Repository != "" {
		imageRef := pushEvent.Digest
		if imageRef == "" {
			imageRef = pushEvent.Tag
		}

		location, repository, _, _, err := parseArtifactResourceURL(imageRef)
		if err != nil {
			ctx.Logger.Infof("gcp artifact registry: cannot parse image reference %q: %v", imageRef, err)
			return nil
		}

		if config.Location != "" && !strings.EqualFold(location, config.Location) {
			ctx.Logger.Infof("gcp artifact registry: image %q does not match location filter %q, skipping", imageRef, config.Location)
			return nil
		}

		if config.Repository != "" && repository != config.Repository {
			ctx.Logger.Infof("gcp artifact registry: repository %q does not match filter %q, skipping", repository, config.Repository)
			return nil
		}
	}

	event := map[string]any{
		"action": pushEvent.Action,
	}
	if pushEvent.Digest != "" {
		event["digest"] = "https://" + pushEvent.Digest
	}
	if pushEvent.Tag != "" {
		event["tag"] = "https://" + pushEvent.Tag
	}
	return ctx.Events.Emit(ArtifactPushEmittedEventType, event)
}

func (t *OnArtifactPush) Cleanup(_ core.TriggerContext) error { return nil }

func (t *OnArtifactPush) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func decodeOnArtifactPushConfiguration(raw any) (OnArtifactPushConfiguration, error) {
	var config OnArtifactPushConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return OnArtifactPushConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Location = strings.TrimSpace(config.Location)
	config.Repository = strings.TrimSpace(config.Repository)
	return config, nil
}
