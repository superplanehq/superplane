package onvmcreate

import (
	"encoding/base64"
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
	// CloudEvents type for Cloud Audit Logs (Eventarc).
	auditLogEventType = "google.cloud.audit.log.v1.written"
	// Compute Engine service name in audit logs.
	computeServiceName = "compute.googleapis.com"
	// Method names for VM instance creation
	instancesInsertMethod      = "v1.compute.instances.insert"
	instancesInsertMethodBeta  = "beta.compute.instances.insert"
	instancesInsertMethodShort = "compute.instances.insert"
)

var vmInsertMethodNames = []string{
	instancesInsertMethod,
	instancesInsertMethodBeta,
	instancesInsertMethodShort,
}

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

type pubSubPushEnvelope struct {
	Message struct {
		Data        string `json:"data"`
		MessageID   string `json:"messageId"`
		PublishTime string `json:"publishTime"`
	} `json:"message"`
	Subscription string `json:"subscription"`
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

const EmittedEventType = "gcp.compute.vmCreated"

type OnVMCreated struct{}

type OnVMCreatedConfiguration struct {
	ProjectID string `json:"projectId" mapstructure:"projectId"`
}

func (t *OnVMCreated) Name() string {
	return "gcp.onVMCreated"
}

func (t *OnVMCreated) Label() string {
	return "On VM Created"
}

func (t *OnVMCreated) Description() string {
	return "Emit when a new Compute Engine VM is created (provisioning succeeded)"
}

func (t *OnVMCreated) Documentation() string {
	return "The On VM Created trigger starts a workflow execution when a new Compute Engine VM is created and provisioning has succeeded.\n\n" +
		"## Use Cases\n\n" +
		"- **Post-provisioning automation**: Run configuration, monitoring, or security setup after a VM is created\n" +
		"- **Inventory and compliance**: Record new VMs or trigger audits\n" +
		"- **Notifications**: Notify teams or systems when new VMs appear in a project or zone\n\n" +
		"## Event Source\n\n" +
		"This trigger expects events from Google Cloud via **Eventarc** (Cloud Audit Logs) or a **Cloud Logging sink to Pub/Sub** with push to the trigger webhook URL.\n\n" +
		"1. **Eventarc (recommended)**\n   Create an Eventarc trigger with:\n   - **Event type**: Cloud Audit Log\n   - **Log type**: Admin Activity (VM create is an admin write)\n   - **Filters**: protoPayload.serviceName=\"compute.googleapis.com\", protoPayload.methodName=\"v1.compute.instances.insert\"\n   - **Destination**: HTTP destination (SuperPlane webhook URL for this trigger)\n\n" +
		"2. **Log sink + Pub/Sub**\n   Create a log sink that writes to a Pub/Sub topic (filter as above), then create a push subscription to the trigger webhook URL.\n\n" +
		"## Configuration\n\n" +
		"- **Project ID**: Optional. Only emit for VMs created in this project.\n\n" +
		"## Event Data\n\n" +
		"Each event includes the full CloudEvents audit payload, including resourceName (e.g. projects/my-project/zones/us-central1-a/instances/my-vm), serviceName (compute.googleapis.com), methodName (v1.compute.instances.insert), and data (decoded audit log entry)."
}

func (t *OnVMCreated) Icon() string {
	return "gcp"
}

func (t *OnVMCreated) Color() string {
	return "gray"
}

func (t *OnVMCreated) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectId",
			Label:       "Project ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Only trigger for VMs created in this project (leave empty for any project).",
		},
	}
}

func (t *OnVMCreated) ExampleData() map[string]any {
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

type OnVMCreatedMetadata struct {
	WebhookURL string `json:"webhookUrl" mapstructure:"webhookUrl"`
}

func resolvePayloadBytes(body []byte) ([]byte, error) {
	var envelope pubSubPushEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, nil
	}
	if envelope.Message.Data == "" {
		return body, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid Pub/Sub message.data base64: %w", err)
	}
	return decoded, nil
}

func (t *OnVMCreated) Setup(ctx core.TriggerContext) error {
	var metadata OnVMCreatedMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)

	if metadata.WebhookURL != "" {
		return nil
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	return ctx.Metadata.Set(OnVMCreatedMetadata{WebhookURL: webhookURL})
}

func (t *OnVMCreated) Actions() []core.Action {
	return nil
}

func (t *OnVMCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnVMCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnVMCreatedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	payloadBytes, err := resolvePayloadBytes(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, err
	}

	var payload EventPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid JSON body: %w", err)
	}

	var serviceName, methodName, resourceName string
	var eventData map[string]any
	var operation *auditLogOperation

	if payload.Type == auditLogEventType && payload.ServiceName != "" {
		serviceName, methodName, resourceName, eventData, operation = normalizedFromEnvelope(&payload)
	} else {
		var logEntry LogEntryDataPayload
		if err := json.Unmarshal(payloadBytes, &logEntry); err != nil || logEntry.ProtoPayload == nil {
			return http.StatusOK, nil
		}
		serviceName, methodName, resourceName, eventData, operation = normalizedFromLogEntry(payloadBytes, &logEntry)
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
			ctx.Logger.Infof("Skipping VM created event for resource %s (project filter: %s)", resourceName, projectID)
			return http.StatusOK, nil
		}
	}

	if err := ctx.Events.Emit(EmittedEventType, eventData); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}
	return http.StatusOK, nil
}

func (t *OnVMCreated) Cleanup(ctx core.TriggerContext) error {
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
