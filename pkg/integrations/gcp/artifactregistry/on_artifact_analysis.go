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
	containerAnalysisServiceName = "containeranalysis.googleapis.com"

	onArtifactAnalysisEmittedEventType = "gcp.artifactregistry.artifactAnalysis"

	OnArtifactAnalysisSinkFilter = `protoPayload.serviceName="containeranalysis.googleapis.com" AND ` +
		`(protoPayload.methodName="google.devtools.containeranalysis.v1.Grafeas.CreateOccurrence" OR ` +
		`protoPayload.methodName="google.devtools.containeranalysis.v1.Grafeas.BatchCreateOccurrences")`

	createOccurrenceMethod      = "google.devtools.containeranalysis.v1.Grafeas.CreateOccurrence"
	batchCreateOccurrenceMethod = "google.devtools.containeranalysis.v1.Grafeas.BatchCreateOccurrences"
)

type OnArtifactAnalysis struct{}

type OnArtifactAnalysisConfiguration struct{}

type OnArtifactAnalysisMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
	SinkID         string `json:"sinkId" mapstructure:"sinkId"`
}

func (t *OnArtifactAnalysis) Name() string {
	return "gcp.artifactregistry.onArtifactAnalysis"
}

func (t *OnArtifactAnalysis) Label() string {
	return "Artifact Registry • On Artifact Analysis"
}

func (t *OnArtifactAnalysis) Description() string {
	return "Trigger a workflow when a vulnerability scan completes for an artifact"
}

func (t *OnArtifactAnalysis) Documentation() string {
	return `The On Artifact Analysis trigger starts a workflow execution when a vulnerability occurrence is created by Container Analysis (Artifact Analysis).

**Trigger behavior:** SuperPlane creates a Cloud Logging sink that captures Container Analysis audit log events for vulnerability occurrence creation and routes them to a shared Pub/Sub topic. Events are pushed to SuperPlane and matched to this trigger automatically.

## Use Cases

- **Security gates**: Block deployments when critical vulnerabilities are found
- **Alerting**: Notify teams immediately when new vulnerabilities are detected
- **Compliance**: Audit vulnerability scanning results automatically

## Setup

**Required GCP setup:** Ensure the **Pub/Sub**, **Artifact Registry**, and **Container Analysis** APIs are enabled in your project and the integration's service account has ` + "`roles/logging.configWriter`" + ` and ` + "`roles/pubsub.admin`" + ` permissions.

SuperPlane automatically creates a Cloud Logging sink to capture artifact analysis events.

## Event Data

Each event includes the audit log entry with serviceName (` + "`containeranalysis.googleapis.com`" + `), methodName, resourceName, and the full log entry data including the vulnerability occurrence details.`
}

func (t *OnArtifactAnalysis) Icon() string  { return "gcp" }
func (t *OnArtifactAnalysis) Color() string { return "gray" }

func (t *OnArtifactAnalysis) Configuration() []configuration.Field {
	return nil
}

func (t *OnArtifactAnalysis) ExampleData() map[string]any {
	return map[string]any{
		"serviceName":  containerAnalysisServiceName,
		"methodName":   createOccurrenceMethod,
		"resourceName": "projects/my-project/occurrences/occurrence-id",
		"logName":      "projects/my-project/logs/cloudaudit.googleapis.com%2Factivity",
		"timestamp":    "2025-06-15T12:00:00Z",
		"data": map[string]any{
			"protoPayload": map[string]any{
				"methodName":   createOccurrenceMethod,
				"resourceName": "projects/my-project/occurrences/occurrence-id",
				"serviceName":  containerAnalysisServiceName,
			},
		},
	}
}

func (t *OnArtifactAnalysis) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing")
	}

	var metadata OnArtifactAnalysisMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.SubscriptionID != "" && metadata.SinkID != "" {
		return nil
	}

	subscriptionID, err := ctx.Integration.Subscribe(artifactAnalysisSubscriptionPattern())
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	sinkID := "sp-sink-" + sanitizeSinkID(subscriptionID.String())

	if err := ctx.Metadata.Set(OnArtifactAnalysisMetadata{
		SubscriptionID: subscriptionID.String(),
		SinkID:         sinkID,
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("provisionSink", map[string]any{
		"sinkId": sinkID,
	}, 2*time.Second)
}

func (t *OnArtifactAnalysis) Actions() []core.Action {
	return []core.Action{
		{Name: "provisionSink"},
	}
}

func (t *OnArtifactAnalysis) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "provisionSink" {
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
	return t.provisionSink(ctx)
}

func (t *OnArtifactAnalysis) provisionSink(ctx core.TriggerActionContext) (map[string]any, error) {
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

	writerIdentity, err := gcppubsub.CreateSink(reqCtx, client, projectID, sinkID, meta.PubSubTopic, OnArtifactAnalysisSinkFilter)
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

func (t *OnArtifactAnalysis) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var event struct {
		ServiceName  string `mapstructure:"serviceName"`
		MethodName   string `mapstructure:"methodName"`
		ResourceName string `mapstructure:"resourceName"`
	}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode event: %w", err)
	}

	if event.ServiceName != containerAnalysisServiceName {
		return nil
	}

	methodName := strings.TrimSpace(event.MethodName)
	if methodName != createOccurrenceMethod && methodName != batchCreateOccurrenceMethod {
		return nil
	}

	return ctx.Events.Emit(onArtifactAnalysisEmittedEventType, ctx.Message)
}

func (t *OnArtifactAnalysis) Cleanup(ctx core.TriggerContext) error {
	var metadata OnArtifactAnalysisMetadata
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

func (t *OnArtifactAnalysis) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func artifactAnalysisSubscriptionPattern() map[string]any {
	return map[string]any{
		"serviceName": containerAnalysisServiceName,
	}
}
