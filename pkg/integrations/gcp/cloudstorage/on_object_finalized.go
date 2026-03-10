package cloudstorage

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
	storageServiceName  = "storage.googleapis.com"
	objectsCreateMethod = "storage.objects.create"
	EmittedEventType    = "gcp.cloudstorage.objectFinalized"

	SinkFilter = `protoPayload.serviceName="storage.googleapis.com" AND protoPayload.methodName="storage.objects.create"`
)

type OnObjectFinalized struct{}

type OnObjectFinalizedConfiguration struct {
	Bucket string `json:"bucket,omitempty" mapstructure:"bucket"`
}

type OnObjectFinalizedMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
	SinkID         string `json:"sinkId" mapstructure:"sinkId"`
}

func (t *OnObjectFinalized) Name() string {
	return "gcp.cloudstorage.onObjectFinalized"
}

func (t *OnObjectFinalized) Label() string {
	return "Cloud Storage • On Object Finalized"
}

func (t *OnObjectFinalized) Description() string {
	return "Trigger a workflow when an object is created or overwritten in a Cloud Storage bucket"
}

func (t *OnObjectFinalized) Documentation() string {
	return `The On Object Finalized trigger starts a workflow execution when a new object is created or an existing object is overwritten in a Cloud Storage bucket.

**Trigger behavior:** SuperPlane creates a Cloud Logging sink that captures Cloud Storage audit log events and routes them to a shared Pub/Sub topic. Events are pushed to SuperPlane and matched to this trigger automatically.

## Use Cases

- **Data pipeline automation**: Process files as soon as they land in a bucket
- **Notifications**: Alert teams when new artifacts are uploaded
- **Compliance and auditing**: Record all object writes for audit trails

## Setup

**Required GCP setup:**
1. Enable **Data Access audit logs** for Cloud Storage in your project (IAM → Audit Logs → Cloud Storage → check "Data Write").
2. Ensure the **Pub/Sub** API is enabled and the integration service account has ` + "`roles/logging.configWriter`" + ` and ` + "`roles/pubsub.admin`" + `.

SuperPlane automatically creates a Cloud Logging sink to capture object finalization events.

## Configuration

- **Bucket** (optional): Filter events to a specific bucket. Leave empty to receive events from all buckets in the project.

## Event Data

Each event includes the audit log entry with resourceName (e.g. projects/_/buckets/my-bucket/objects/path/to/file.txt), serviceName (storage.googleapis.com), methodName (storage.objects.create), and the full log entry data.`
}

func (t *OnObjectFinalized) Icon() string {
	return "gcp"
}

func (t *OnObjectFinalized) Color() string {
	return "gray"
}

func (t *OnObjectFinalized) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optionally filter events to a specific bucket. Leave empty for all buckets.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeBucket,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
	}
}

func (t *OnObjectFinalized) ExampleData() map[string]any {
	return map[string]any{
		"serviceName":  storageServiceName,
		"methodName":   objectsCreateMethod,
		"resourceName": "projects/_/buckets/my-bucket/objects/path/to/file.txt",
		"logName":      "projects/my-project/logs/cloudaudit.googleapis.com%2Fdata_access",
		"timestamp":    "2025-06-01T12:00:00Z",
		"data": map[string]any{
			"protoPayload": map[string]any{
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/my-bucket/objects/path/to/file.txt",
				"serviceName":  storageServiceName,
			},
		},
	}
}

func (t *OnObjectFinalized) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing")
	}

	var metadata OnObjectFinalizedMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.SubscriptionID != "" && metadata.SinkID != "" {
		return nil
	}

	subscriptionID, err := ctx.Integration.Subscribe(subscriptionPattern())
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	sinkID := "sp-sink-" + sanitizeSinkID(subscriptionID.String())

	if err := ctx.Metadata.Set(OnObjectFinalizedMetadata{
		SubscriptionID: subscriptionID.String(),
		SinkID:         sinkID,
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("provisionSink", map[string]any{
		"sinkId": sinkID,
	}, 2*time.Second)
}

func (t *OnObjectFinalized) Actions() []core.Action {
	return []core.Action{
		{Name: "provisionSink"},
	}
}

func (t *OnObjectFinalized) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "provisionSink" {
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return t.provisionSink(ctx)
}

func (t *OnObjectFinalized) provisionSink(ctx core.TriggerActionContext) (map[string]any, error) {
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

	writerIdentity, err := gcppubsub.CreateSink(reqCtx, client, projectID, sinkID, meta.PubSubTopic, SinkFilter)
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

func (t *OnObjectFinalized) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var event struct {
		ServiceName  string `mapstructure:"serviceName"`
		MethodName   string `mapstructure:"methodName"`
		ResourceName string `mapstructure:"resourceName"`
		Data         any    `mapstructure:"data"`
	}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode event: %w", err)
	}

	if event.ServiceName != storageServiceName {
		return nil
	}

	if strings.TrimSpace(event.MethodName) != objectsCreateMethod {
		return nil
	}

	var config OnObjectFinalizedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err == nil && config.Bucket != "" {
		if !resourceNameMatchesBucket(event.ResourceName, config.Bucket) {
			return nil
		}
	}

	return ctx.Events.Emit(EmittedEventType, ctx.Message)
}

func resourceNameMatchesBucket(resourceName, bucket string) bool {
	return strings.Contains(resourceName, "/buckets/"+bucket+"/")
}

func (t *OnObjectFinalized) Cleanup(ctx core.TriggerContext) error {
	var metadata OnObjectFinalizedMetadata
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

func (t *OnObjectFinalized) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func subscriptionPattern() map[string]any {
	return map[string]any{
		"serviceName": storageServiceName,
		"methodName":  objectsCreateMethod,
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
