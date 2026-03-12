package compute

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	gcppubsub "github.com/superplanehq/superplane/pkg/integrations/gcp/pubsub"
)

const (
	computeServiceName         = "compute.googleapis.com"
	instancesInsertMethod      = "v1.compute.instances.insert"
	instancesInsertMethodBeta  = "beta.compute.instances.insert"
	instancesInsertMethodShort = "compute.instances.insert"
)

var vmInsertMethodNames = []string{
	instancesInsertMethod,
	instancesInsertMethodBeta,
	instancesInsertMethodShort,
}

const (
	EmittedEventType = "gcp.compute.vmInstance"

	// SinkFilter is the Cloud Logging advanced log filter for VM instance
	// creation audit events. Used when creating the per-trigger logging sink.
	SinkFilter = `protoPayload.serviceName="compute.googleapis.com" AND (protoPayload.methodName="v1.compute.instances.insert" OR protoPayload.methodName="beta.compute.instances.insert" OR protoPayload.methodName="compute.instances.insert")`
)

type OnVMInstance struct{}

type OnVMInstanceConfiguration struct{}

type OnVMInstanceMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
	SinkID         string `json:"sinkId" mapstructure:"sinkId"`
}

func (t *OnVMInstance) Name() string {
	return "gcp.compute.onVMInstance"
}

func (t *OnVMInstance) Label() string {
	return "Compute • On VM Instance"
}

func (t *OnVMInstance) Description() string {
	return "Listen to GCP Compute Engine VM instance lifecycle events"
}

func (t *OnVMInstance) Documentation() string {
	return `The On VM Instance trigger starts a workflow execution when a Compute Engine VM instance lifecycle event occurs.

**Trigger behavior:** SuperPlane creates a Cloud Logging sink that captures Compute Engine audit log events and routes them to a shared Pub/Sub topic. Events are pushed to SuperPlane and matched to this trigger automatically.

## Use Cases

- **Post-provisioning automation**: Run configuration, monitoring, or security setup after a VM is created
- **Inventory and compliance**: Record new VMs or trigger audits
- **Notifications**: Notify teams or systems when new VMs appear in a project or zone

## Setup

**Required GCP setup:** Ensure the **Pub/Sub** API is enabled in your project and the integration's service account has ` + "`roles/logging.configWriter`" + ` and ` + "`roles/pubsub.admin`" + ` permissions.

SuperPlane automatically creates a Cloud Logging sink to capture VM instance events.

## Event Data

Each event includes the audit log entry with resourceName (e.g. projects/my-project/zones/us-central1-a/instances/my-vm), serviceName (compute.googleapis.com), methodName (v1.compute.instances.insert), and the full log entry data.`
}

func (t *OnVMInstance) Icon() string {
	return "gcp"
}

func (t *OnVMInstance) Color() string {
	return "gray"
}

func (t *OnVMInstance) Configuration() []configuration.Field {
	return nil
}

func (t *OnVMInstance) ExampleData() map[string]any {
	return map[string]any{
		"serviceName":  computeServiceName,
		"methodName":   instancesInsertMethod,
		"resourceName": "projects/my-project/zones/us-central1-a/instances/my-vm",
		"logName":      "projects/my-project/logs/cloudaudit.googleapis.com%2Factivity",
		"timestamp":    "2025-02-14T12:00:00Z",
		"data": map[string]any{
			"protoPayload": map[string]any{
				"methodName":   instancesInsertMethod,
				"resourceName": "projects/my-project/zones/us-central1-a/instances/my-vm",
				"serviceName":  computeServiceName,
			},
		},
	}
}

func (t *OnVMInstance) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing")
	}

	var metadata OnVMInstanceMetadata
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

	if err := ctx.Metadata.Set(OnVMInstanceMetadata{
		SubscriptionID: subscriptionID.String(),
		SinkID:         sinkID,
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("provisionSink", map[string]any{
		"sinkId": sinkID,
	}, 2*time.Second)
}

func (t *OnVMInstance) Actions() []core.Action {
	return []core.Action{
		{Name: "provisionSink"},
	}
}

func (t *OnVMInstance) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name != "provisionSink" {
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return t.provisionSink(ctx)
}

func (t *OnVMInstance) provisionSink(ctx core.TriggerActionContext) (map[string]any, error) {
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

func (t *OnVMInstance) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var event struct {
		ServiceName  string `mapstructure:"serviceName"`
		MethodName   string `mapstructure:"methodName"`
		ResourceName string `mapstructure:"resourceName"`
		Data         any    `mapstructure:"data"`
	}
	if err := mapstructure.Decode(ctx.Message, &event); err != nil {
		return fmt.Errorf("failed to decode event: %w", err)
	}

	if event.ServiceName != computeServiceName {
		return nil
	}

	methodName := strings.TrimSpace(event.MethodName)
	if !slices.Contains(vmInsertMethodNames, methodName) {
		return nil
	}

	return ctx.Events.Emit(EmittedEventType, ctx.Message)
}

func (t *OnVMInstance) Cleanup(ctx core.TriggerContext) error {
	var metadata OnVMInstanceMetadata
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

func (t *OnVMInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func subscriptionPattern() map[string]any {
	return map[string]any{
		"serviceName": computeServiceName,
		"methodName":  instancesInsertMethod,
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
