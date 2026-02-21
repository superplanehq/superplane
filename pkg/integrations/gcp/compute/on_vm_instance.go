package compute

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	auditLogEventType          = "google.cloud.audit.log.v1.written"
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

// InstanceCelFilter is the Eventarc Advanced enrollment filter for VM instance creation events.
const InstanceCelFilter = `message.type == "google.cloud.audit.log.v1.written" && message.serviceName == "compute.googleapis.com" && (message.methodName == "v1.compute.instances.insert" || message.methodName == "beta.compute.instances.insert" || message.methodName == "compute.instances.insert")`

type auditLogOperation struct {
	Last  bool `json:"last"`
	First bool `json:"first"`
}

type EventPayload struct {
	Type            string         `json:"type"`
	Source          string         `json:"source"`
	SpecVersion     string         `json:"specversion"`
	DataContentType string         `json:"datacontenttype"`
	ID              string         `json:"id"`
	Time            string         `json:"time"`
	ServiceName     string         `json:"serviceName"`
	MethodName      string         `json:"methodName"`
	ResourceName    string         `json:"resourceName"`
	Subject         string         `json:"subject"`
	Data            map[string]any `json:"data"`
}

type LogEntryDataPayload struct {
	ProtoPayload *struct {
		ServiceName  string `json:"serviceName"`
		MethodName   string `json:"methodName"`
		ResourceName string `json:"resourceName"`
	} `json:"protoPayload"`
	Operation        *auditLogOperation `json:"operation"`
	LogName          string             `json:"logName"`
	Timestamp        string             `json:"timestamp"`
	InsertID         string             `json:"insertId"`
	Resource         any                `json:"resource"`
	ReceiveTimestamp string             `json:"receiveTimestamp"`
}

const EmittedEventType = "gcp.compute.vmInstance"

type OnVMInstance struct{}

type OnVMInstanceConfiguration struct {
	ProjectID string `json:"projectId" mapstructure:"projectId"`
	Region    string `json:"region" mapstructure:"region"`
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

**Trigger behavior:** The trigger uses **Eventarc Advanced**: a shared message bus receives all Google Cloud audit log events, and an enrollment filters for VM instance events and delivers them to SuperPlane via an HTTPS pipeline with OIDC authentication.

## Use Cases

- **Post-provisioning automation**: Run configuration, monitoring, or security setup after a VM is created
- **Inventory and compliance**: Record new VMs or trigger audits
- **Notifications**: Notify teams or systems when new VMs appear in a project or zone

## Automatic setup

When you set **Project ID** (and optionally **Region**), SuperPlane automatically creates the Eventarc Advanced resources (message bus, Google API source, pipeline, and enrollment) needed to receive VM instance events. No manual setup is required.

**Required GCP setup:** Enable the **Eventarc** API in your project and grant the integration's service account ` + "`roles/eventarc.developer`" + ` and ` + "`roles/iam.serviceAccountTokenCreator`" + `.

**Local testing:** Use ngrok (` + "`ngrok http 8000`" + `) and set ` + "`BASE_URL`" + ` and ` + "`WEBHOOKS_BASE_URL`" + ` to the ngrok HTTPS URL so GCP can reach the webhook.

## Configuration

- **Project ID**: Required. The GCP project where Eventarc resources are created and where VM instance events are received.
- **Region**: Optional. Default: us-central1. The region where Eventarc resources are provisioned.

## Event Data

Each event includes the full CloudEvents audit payload, including resourceName (e.g. projects/my-project/zones/us-central1-a/instances/my-vm), serviceName (compute.googleapis.com), methodName (v1.compute.instances.insert), and data (audit log entry).`
}

func (t *OnVMInstance) Icon() string {
	return "gcp"
}

func (t *OnVMInstance) Color() string {
	return "gray"
}

func (t *OnVMInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectId",
			Label:       "Project ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "GCP project where Eventarc resources are created and where VM instance events are received.",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Region where Eventarc resources are provisioned (e.g. us-central1). Default: us-central1.",
			Default:     "us-central1",
		},
	}
}

func (t *OnVMInstance) ExampleData() map[string]any {
	return map[string]any{
		"type":         auditLogEventType,
		"serviceName":  computeServiceName,
		"methodName":   instancesInsertMethod,
		"resourceName": "projects/my-project/zones/us-central1-a/instances/my-vm",
		"source":       "//cloudaudit.googleapis.com/projects/my-project/logs/activity",
		"id":           "example-event-id",
		"time":         "2025-02-14T12:00:00Z",
		"data": map[string]any{
			"protoPayload": map[string]any{
				"methodName":   instancesInsertMethod,
				"resourceName": "projects/my-project/zones/us-central1-a/instances/my-vm",
				"serviceName":  computeServiceName,
			},
		},
	}
}

type OnVMInstanceMetadata struct {
	WebhookURL string `json:"webhookUrl" mapstructure:"webhookUrl"`
}

func (t *OnVMInstance) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing with Eventarc Advanced")
	}

	var config OnVMInstanceConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	projectID := strings.TrimSpace(config.ProjectID)
	if projectID == "" {
		return fmt.Errorf("project ID is required for automatic registration with Eventarc Advanced; set Project ID in the trigger configuration")
	}

	webhookConfig := map[string]any{
		"projectId": projectID,
		"region":    strings.TrimSpace(config.Region),
		"celFilter": InstanceCelFilter,
	}
	if err := ctx.Integration.RequestWebhook(webhookConfig); err != nil {
		return fmt.Errorf("failed to request webhook for On VM Instance: %w", err)
	}

	return ctx.Metadata.Set(OnVMInstanceMetadata{WebhookURL: "(registered automatically)"})
}

func (t *OnVMInstance) Actions() []core.Action {
	return nil
}

func (t *OnVMInstance) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnVMInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnVMInstanceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var payload EventPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid JSON body: %w", err)
	}

	var serviceName, methodName, resourceName string
	var eventData map[string]any
	var operation *auditLogOperation

	if payload.Type == auditLogEventType && payload.ServiceName != "" {
		serviceName, methodName, resourceName, eventData, operation = normalizedFromEnvelope(&payload)
	} else {
		var logEntry LogEntryDataPayload
		if err := json.Unmarshal(ctx.Body, &logEntry); err != nil || logEntry.ProtoPayload == nil {
			return http.StatusOK, nil
		}
		serviceName, methodName, resourceName, eventData, operation = normalizedFromLogEntry(ctx.Body, &logEntry)
	}

	if serviceName != computeServiceName {
		return http.StatusOK, nil
	}
	if !slices.Contains(vmInsertMethodNames, methodName) {
		return http.StatusOK, nil
	}
	if !isCompletionEvent(operation) {
		return http.StatusOK, nil
	}

	if config.ProjectID != "" {
		projectID := strings.TrimSpace(config.ProjectID)
		resourceProject := extractProjectFromResourceName(resourceName)
		if resourceProject == "" || resourceProject != projectID {
			ctx.Logger.Infof("Skipping VM instance event for resource %s (project filter: %s)", resourceName, projectID)
			return http.StatusOK, nil
		}
	}

	if err := ctx.Events.Emit(EmittedEventType, eventData); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}
	return http.StatusOK, nil
}

func (t *OnVMInstance) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func extractProjectFromResourceName(resourceName string) string {
	resourceName = strings.TrimSpace(resourceName)
	const prefix = "projects/"
	if !strings.HasPrefix(resourceName, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(resourceName, prefix)
	idx := strings.Index(rest, "/")
	if idx < 0 {
		return rest
	}
	return rest[:idx]
}

func isCompletionEvent(operation *auditLogOperation) bool {
	if operation == nil {
		return true
	}
	return operation.Last
}

func operationFromData(data map[string]any) *auditLogOperation {
	if data == nil {
		return nil
	}
	opMap, ok := data["operation"].(map[string]any)
	if !ok {
		return nil
	}
	last, _ := opMap["last"].(bool)
	return &auditLogOperation{Last: last}
}

func normalizedFromEnvelope(payload *EventPayload) (serviceName, methodName, resourceName string, eventData map[string]any, operation *auditLogOperation) {
	serviceName = payload.ServiceName
	methodName = strings.TrimSpace(payload.MethodName)
	resourceName = strings.TrimSpace(payload.ResourceName)
	eventData = map[string]any{
		"type":         payload.Type,
		"source":       payload.Source,
		"specversion":  payload.SpecVersion,
		"id":           payload.ID,
		"time":         payload.Time,
		"serviceName":  payload.ServiceName,
		"methodName":   methodName,
		"resourceName": resourceName,
		"subject":      payload.Subject,
		"data":         payload.Data,
	}
	operation = operationFromData(payload.Data)
	return serviceName, methodName, resourceName, eventData, operation
}

func normalizedFromLogEntry(body []byte, entry *LogEntryDataPayload) (serviceName, methodName, resourceName string, eventData map[string]any, operation *auditLogOperation) {
	proto := entry.ProtoPayload
	serviceName = proto.ServiceName
	methodName = strings.TrimSpace(proto.MethodName)
	resourceName = proto.ResourceName
	eventData = map[string]any{
		"type":         auditLogEventType,
		"serviceName":  serviceName,
		"methodName":   methodName,
		"resourceName": resourceName,
		"logName":      entry.LogName,
		"timestamp":    entry.Timestamp,
		"insertId":     entry.InsertID,
		"data":         nil,
	}
	var raw map[string]any
	if json.Unmarshal(body, &raw) == nil {
		eventData["data"] = raw
	}
	operation = entry.Operation
	return serviceName, methodName, resourceName, eventData, operation
}
