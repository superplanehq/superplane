package artifactregistry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	gcppubsub "github.com/superplanehq/superplane/pkg/integrations/gcp/pubsub"
)

const (
	artifactRegistryServiceName = "artifactregistry.googleapis.com"

	onArtifactPushEmittedEventType = "gcp.artifactregistry.artifactPush"

	OnArtifactPushSinkFilter = `protoPayload.serviceName="artifactregistry.googleapis.com" AND ` +
		`protoPayload.methodName="google.devtools.artifactregistry.v1.ArtifactRegistry.CreateDockerImage"`

	createDockerImageMethod = "google.devtools.artifactregistry.v1.ArtifactRegistry.CreateDockerImage"
)

type OnArtifactPush struct{}

type OnArtifactPushConfiguration struct {
	Location   string `json:"location" mapstructure:"location"`
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnArtifactPushMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
	SinkID         string `json:"sinkId" mapstructure:"sinkId"`
}

func (t *OnArtifactPush) Name() string {
	return "gcp.artifactregistry.onArtifactPush"
}

func (t *OnArtifactPush) Label() string {
	return "Artifact Registry • On Artifact Push"
}

func (t *OnArtifactPush) Description() string {
	return "Trigger a workflow when an artifact is pushed to Artifact Registry"
}

func (t *OnArtifactPush) Documentation() string {
	return `The On Artifact Push trigger starts a workflow execution when a Docker image is pushed to an Artifact Registry repository.

**Trigger behavior:** SuperPlane creates a Cloud Logging sink that captures Artifact Registry audit log events for Docker image creation and routes them to a shared Pub/Sub topic. Events are pushed to SuperPlane and matched to this trigger automatically.

## Use Cases

- **CI/CD pipelines**: Trigger deployments when a new image is pushed
- **Security scanning**: Kick off vulnerability scans on newly pushed images
- **Notifications**: Alert teams when new artifacts are published

## Setup

**Required GCP setup:** Ensure the **Pub/Sub** and **Artifact Registry** APIs are enabled in your project and the integration's service account has ` + "`roles/logging.configWriter`" + ` and ` + "`roles/pubsub.admin`" + ` permissions.

SuperPlane automatically creates a Cloud Logging sink to capture artifact push events.

## Configuration

- **Location**: Optionally filter events to a specific GCP region.
- **Repository**: Optionally filter events to a specific Artifact Registry repository.

## Event Data

Each event includes the audit log entry with resourceName, serviceName (` + "`artifactregistry.googleapis.com`" + `), methodName, and the full log entry data.`
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
			Description: "Optionally filter to a specific region. Leave empty to receive events from all locations.",
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
			Description: "Optionally filter to a specific repository. Leave empty to receive events from all repositories.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeRepository,
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
					},
				},
			},
		},
	}
}

func (t *OnArtifactPush) ExampleData() map[string]any {
	return map[string]any{
		"serviceName":  artifactRegistryServiceName,
		"methodName":   createDockerImageMethod,
		"resourceName": "projects/my-project/locations/us-central1/repositories/my-repo/dockerImages/my-image@sha256:abc123",
		"logName":      "projects/my-project/logs/cloudaudit.googleapis.com%2Factivity",
		"timestamp":    "2025-06-15T12:00:00Z",
		"data": map[string]any{
			"protoPayload": map[string]any{
				"methodName":   createDockerImageMethod,
				"resourceName": "projects/my-project/locations/us-central1/repositories/my-repo/dockerImages/my-image@sha256:abc123",
				"serviceName":  artifactRegistryServiceName,
			},
		},
	}
}

func (t *OnArtifactPush) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing")
	}

	var metadata OnArtifactPushMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.SubscriptionID != "" && metadata.SinkID != "" {
		return nil
	}

	subscriptionID, err := ctx.Integration.Subscribe(artifactPushSubscriptionPattern())
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	sinkID := "sp-sink-" + sanitizeSinkID(subscriptionID.String())

	if err := ctx.Metadata.Set(OnArtifactPushMetadata{
		SubscriptionID: subscriptionID.String(),
		SinkID:         sinkID,
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("provisionSink", map[string]any{
		"sinkId": sinkID,
	}, 2*time.Second)
}

func (t *OnArtifactPush) Actions() []core.Action {
	return []core.Action{
		{Name: "provisionSink"},
	}
}

func (t *OnArtifactPush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "provisionSink" {
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
	return t.provisionSink(ctx)
}

func (t *OnArtifactPush) provisionSink(ctx core.TriggerActionContext) (map[string]any, error) {
	meta, err := integrationMetadata(ctx.Integration)
	if err != nil {
		return nil, err
	}

	if meta.PubSubTopic == "" {
		return nil, fmt.Errorf("integration Pub/Sub topic not configured; re-sync the GCP integration")
	}

	sinkID, _ := ctx.Parameters["sinkId"].(string)
	if sinkID == "" {
		return nil, fmt.Errorf("sinkId parameter is required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("create GCP client: %w", err)
	}

	projectID := client.ProjectID()
	reqCtx := context.Background()

	writerIdentity, err := gcppubsub.CreateSink(reqCtx, client, projectID, sinkID, meta.PubSubTopic, OnArtifactPushSinkFilter)
	if err != nil {
		if !gcpcommon.IsAlreadyExistsError(err) {
			return nil, fmt.Errorf("create logging sink: %w", err)
		}

		writerIdentity, err = gcppubsub.GetSink(reqCtx, client, projectID, sinkID)
		if err != nil {
			return nil, fmt.Errorf("get existing logging sink: %w", err)
		}
	}

	if err := gcppubsub.EnsureTopicPublisher(reqCtx, client, projectID, meta.PubSubTopic, writerIdentity); err != nil {
		return nil, fmt.Errorf("grant sink publisher permission on topic: %w", err)
	}

	return nil, nil
}

func (t *OnArtifactPush) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config, err := decodeOnArtifactPushConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	var event struct {
		ServiceName  string `mapstructure:"serviceName"`
		MethodName   string `mapstructure:"methodName"`
		ResourceName string `mapstructure:"resourceName"`
	}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode event: %w", err)
	}

	if event.ServiceName != artifactRegistryServiceName {
		return nil
	}

	if event.MethodName != createDockerImageMethod {
		return nil
	}

	if config.Location != "" && !strings.Contains(event.ResourceName, "/locations/"+config.Location+"/") {
		return nil
	}

	if config.Repository != "" && !strings.Contains(event.ResourceName, "/repositories/"+config.Repository+"/") {
		return nil
	}

	return ctx.Events.Emit(onArtifactPushEmittedEventType, ctx.Message)
}

func (t *OnArtifactPush) Cleanup(ctx core.TriggerContext) error {
	var metadata OnArtifactPushMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil || metadata.SinkID == "" {
		return nil
	}

	if ctx.Integration == nil {
		return nil
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("failed to create GCP client for sink cleanup: %v", err)
		return nil
	}

	if err := gcppubsub.DeleteSink(context.Background(), client, client.ProjectID(), metadata.SinkID); err != nil {
		if !gcpcommon.IsNotFoundError(err) {
			ctx.Logger.Warnf("failed to delete logging sink %s: %v", metadata.SinkID, err)
		}
	}

	return nil
}

func (t *OnArtifactPush) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return 200, nil
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

func artifactPushSubscriptionPattern() map[string]any {
	return map[string]any{
		"serviceName": artifactRegistryServiceName,
		"methodName":  createDockerImageMethod,
	}
}

func integrationMetadata(integration core.IntegrationContext) (*gcpcommon.Metadata, error) {
	var m gcpcommon.Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &m); err != nil {
		return nil, fmt.Errorf("failed to read integration metadata: %w", err)
	}
	if m.ProjectID == "" {
		return nil, fmt.Errorf("integration metadata does not contain a project ID; re-sync the GCP integration")
	}
	return &m, nil
}

func sanitizeSinkID(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	result := b.String()
	if len(result) > 80 {
		result = result[:80]
	}
	return result
}
