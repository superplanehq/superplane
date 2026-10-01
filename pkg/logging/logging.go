package logging

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	// ComponentWebhookProductive is the Cloud Logging component for a
	// Productive task webhook failure. Filter:
	// jsonPayload.component="webhook.productive"
	ComponentWebhookProductive = "webhook.productive"
	// ComponentWebhookSentry is the Cloud Logging component for a hosted
	// Sentry webhook. Filter: jsonPayload.component="webhook.sentry"
	ComponentWebhookSentry = "webhook.sentry"

	// WebhookLogType is the Cloud Logging type for an integration webhook.
	// Filter: jsonPayload.type="webhook"
	WebhookLogType = "webhook"
	// DatadogIntegration is the Cloud Logging integration for a Datadog webhook.
	// Filter: jsonPayload.integration="datadog"
	DatadogIntegration = "datadog"

	productiveWebhookFailureMessage = "error handling webhook"
	webhookTypeUnknown              = "unknown"
	productiveTaskCreated           = "task.created"
	productiveTaskUpdated           = "task.updated"

	// webhookLogPayloadLimit keeps one webhook log under Cloud Logging's
	// 256 KB entry limit after the other fields are added.
	webhookLogPayloadLimit  = 128 * 1024
	webhookPayloadTruncated = "...(truncated)"
)

// WebhookNodeFields are the canvas and webhook ids added to a node logger
// when a webhook delivery runs.
type WebhookNodeFields struct {
	OrganizationID string
	CanvasID       string
	WebhookID      string
}

// standardLogWriter sends a JSON log line to the process logger output.
// The process formatter stays text.
type standardLogWriter struct{}

func (standardLogWriter) Write(p []byte) (int, error) {
	return log.StandardLogger().Out.Write(p)
}

func newJSONLineLogger() *log.Logger {
	logger := log.New()
	logger.SetFormatter(&log.JSONFormatter{})
	logger.SetOutput(standardLogWriter{})
	logger.SetLevel(log.StandardLogger().GetLevel())
	return logger
}

func newCloudLoggingLineLogger() *log.Logger {
	logger := newJSONLineLogger()
	logger.SetFormatter(NewCloudLoggingFormatter())
	return logger
}

func newProductiveWebhookLogger() *log.Logger {
	return newJSONLineLogger()
}

// productiveWebhookLogger writes one JSON object per Productive webhook failure.
var productiveWebhookLogger = newProductiveWebhookLogger()

// sentryWebhookLogger writes one JSON object per hosted Sentry webhook log.
// message and severity are the Cloud Logging summary fields. The other fields
// stay in the JSON payload.
var sentryWebhookLogger = newCloudLoggingLineLogger()

// datadogWebhookLogger writes one JSON object per Datadog webhook log.
var datadogWebhookLogger = newJSONLineLogger()

// SentryWebhookLogger returns the JSON logger used for hosted Sentry webhooks.
func SentryWebhookLogger() *log.Logger {
	return sentryWebhookLogger
}

// DatadogWebhookLogger returns the JSON logger used for Datadog webhooks.
func DatadogWebhookLogger() *log.Logger {
	return datadogWebhookLogger
}

// ProductiveWebhookLogger returns the JSON logger used for Productive webhook failures.
func ProductiveWebhookLogger() *log.Logger {
	return productiveWebhookLogger
}

func ForFactory(factory models.Factory) *log.Entry {
	return WithFactory(log.NewEntry(log.StandardLogger()), factory)
}

func WithFactory(logger *log.Entry, factory models.Factory) *log.Entry {
	return logger.WithFields(log.Fields{
		"factory_id": factory.ID,
	})
}

func ForWorkOrder(workOrder models.FactoryWorkOrder) *log.Entry {
	return WithWorkOrder(log.NewEntry(log.StandardLogger()), workOrder)
}

func WithWorkOrder(logger *log.Entry, workOrder models.FactoryWorkOrder) *log.Entry {
	return logger.WithFields(log.Fields{
		"order_id": workOrder.ID,
	})
}

func ForEvent(logger *log.Entry, event models.CanvasEvent) *log.Entry {
	return logger.WithFields(log.Fields{
		"event_id": event.ID,
		"node_id":  event.NodeID,
		"channel":  event.Channel,
	})
}

func ForExecution(execution *models.CanvasNodeExecution) *log.Entry {
	return WithExecution(log.NewEntry(log.StandardLogger()), execution)
}

func WithExecution(
	logger *log.Entry,
	execution *models.CanvasNodeExecution,
) *log.Entry {
	return logger.WithFields(log.Fields{
		"root_event": execution.RootEventID,
		"execution":  execution.ID,
	})
}

func ForNode(node models.CanvasNode) *log.Entry {
	return WithNode(log.NewEntry(log.StandardLogger()), node)
}

func WithNode(logger *log.Entry, node models.CanvasNode) *log.Entry {
	return logger.WithFields(log.Fields{
		"node_id": node.NodeID,
	})
}

func WithQueueItem(logger *log.Entry, queueItem models.CanvasNodeQueueItem) *log.Entry {
	return logger.WithFields(log.Fields{
		"queue_item_id": queueItem.ID,
		"root_event":    queueItem.RootEventID,
	})
}

func ForIntegration(integration models.Integration) *log.Entry {
	return WithIntegration(log.NewEntry(log.StandardLogger()), integration)
}

func WithIntegration(logger *log.Entry, integration models.Integration) *log.Entry {
	return logger.WithFields(log.Fields{
		"integration_name": integration.AppName,
		"integration_id":   integration.ID,
	})
}

func WithWebhook(logger *log.Entry, webhook models.Webhook) *log.Entry {
	return logger.WithFields(log.Fields{
		"webhook_id": webhook.ID,
	})
}

func ForRun(run models.CanvasRun) *log.Entry {
	return WithRun(log.NewEntry(log.StandardLogger()), run)
}

func WithRun(logger *log.Entry, run models.CanvasRun) *log.Entry {
	return logger.WithFields(log.Fields{
		"run_id":      run.ID,
		"workflow_id": run.WorkflowID,
	})
}

func WithCanvas(logger *log.Entry, canvas models.Canvas) *log.Entry {
	return logger.WithFields(log.Fields{
		"canvas_id": canvas.ID,
	})
}

// WithCanvasWorkspace adds organization_id and factory_id when the canvas
// belongs to a workspace. factory_id is the workspace UUID. A canvas outside
// a workspace is left unchanged.
func WithCanvasWorkspace(logger *log.Entry, canvas *models.Canvas) *log.Entry {
	if logger == nil || canvas == nil || canvas.FactoryID == nil || *canvas.FactoryID == uuid.Nil {
		return logger
	}

	return logger.WithFields(log.Fields{
		"organization_id": canvas.OrganizationID,
		"factory_id":      *canvas.FactoryID,
	})
}

// WithWebhookNode adds the SuperPlane organization, canvas, and webhook ids
// to the logger passed into a webhook handler.
func WithWebhookNode(logger *log.Entry, fields WebhookNodeFields) *log.Entry {
	logFields := log.Fields{}
	if fields.OrganizationID != "" {
		logFields["organization_id"] = fields.OrganizationID
	}
	if fields.CanvasID != "" {
		logFields["workflow_id"] = fields.CanvasID
	}
	if fields.WebhookID != "" {
		logFields["webhook_id"] = fields.WebhookID
	}
	if len(logFields) == 0 {
		return logger
	}
	return logger.WithFields(logFields)
}

// ProductiveWebhookType returns task.created or task.updated.
// Any other delivery is unknown.
func ProductiveWebhookType(event string) string {
	switch strings.TrimSpace(event) {
	case productiveTaskCreated, productiveTaskUpdated:
		return strings.TrimSpace(event)
	default:
		return webhookTypeUnknown
	}
}

func productiveWebhookFields(event string) log.Fields {
	return log.Fields{
		"component":   ComponentWebhookProductive,
		"webhookType": ProductiveWebhookType(event),
	}
}

// LogProductiveWebhookFailure writes one JSON failure line for a Productive
// task webhook. component and webhookType stay set when fields repeat them.
func LogProductiveWebhookFailure(event string, fields log.Fields, err error) {
	productiveWebhookEntry(event, fields, err).Error(productiveWebhookFailureMessage)
}

// LogProductiveWebhookWarning writes one JSON warning line for a Productive
// task webhook that succeeded with less data than expected.
func LogProductiveWebhookWarning(event string, message string, fields log.Fields, err error) {
	productiveWebhookEntry(event, fields, err).Warn(message)
}

// WithWebhookPayload adds the incoming webhook body to a log line.
// Valid JSON stays an object. A body over the limit is cut.
func WithWebhookPayload(fields log.Fields, body []byte) log.Fields {
	payload := webhookLogPayload(body)
	if payload == nil {
		return fields
	}
	if fields == nil {
		fields = log.Fields{}
	}
	fields["payload"] = payload
	return fields
}

func webhookLogPayload(body []byte) any {
	if len(body) == 0 {
		return nil
	}
	if len(body) > webhookLogPayloadLimit {
		return string(body[:webhookLogPayloadLimit]) + webhookPayloadTruncated
	}
	if json.Valid(body) {
		return json.RawMessage(body)
	}
	return string(body)
}

// LogSentryWebhookInfo writes one JSON info line for a hosted Sentry webhook.
func LogSentryWebhookInfo(message string, fields log.Fields) {
	sentryWebhookEntry(fields, nil).Info(message)
}

// LogSentryWebhookWarn writes one JSON warning line for a hosted Sentry webhook.
func LogSentryWebhookWarn(message string, fields log.Fields) {
	sentryWebhookEntry(fields, nil).Warn(message)
}

// LogSentryWebhookError writes one JSON error line for a hosted Sentry webhook.
func LogSentryWebhookError(message string, fields log.Fields, err error) {
	sentryWebhookEntry(fields, err).Error(message)
}

func sentryWebhookEntry(fields log.Fields, err error) *log.Entry {
	sentryWebhookLogger.SetLevel(log.StandardLogger().GetLevel())
	entry := log.NewEntry(sentryWebhookLogger)
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}
	entry = entry.WithField("component", ComponentWebhookSentry)
	if err != nil {
		entry = entry.WithError(err)
	}
	return entry
}

// LogDatadogWebhookInfo writes one JSON info line for a Datadog webhook.
// type and integration stay set when fields repeat them.
func LogDatadogWebhookInfo(message string, fields log.Fields, err error) {
	datadogWebhookEntry(fields, err).Info(message)
}

// LogDatadogWebhookWarn writes one JSON warning line for a Datadog webhook.
// type and integration stay set when fields repeat them.
func LogDatadogWebhookWarn(message string, fields log.Fields, err error) {
	datadogWebhookEntry(fields, err).Warn(message)
}

// LogDatadogWebhookError writes one JSON error line for a Datadog webhook.
// type and integration stay set when fields repeat them.
func LogDatadogWebhookError(message string, fields log.Fields, err error) {
	datadogWebhookEntry(fields, err).Error(message)
}

type datadogWebhookIdentityKey struct{}

// WithDatadogWebhookIdentity stores a resolver on the logger.
// The caller runs the resolver only when a delivery line needs the fields.
func WithDatadogWebhookIdentity(logger *log.Entry, resolve func() log.Fields) *log.Entry {
	if logger == nil {
		return nil
	}

	parent := logger.Context
	if parent == nil {
		parent = context.Background()
	}
	return logger.WithContext(context.WithValue(parent, datadogWebhookIdentityKey{}, resolve))
}

// DatadogWebhookIdentity returns fields from a resolver attached by
// WithDatadogWebhookIdentity. It returns nil when the logger has no resolver.
func DatadogWebhookIdentity(logger *log.Entry) log.Fields {
	if logger == nil || logger.Context == nil {
		return nil
	}

	resolve, ok := logger.Context.Value(datadogWebhookIdentityKey{}).(func() log.Fields)
	if !ok || resolve == nil {
		return nil
	}
	return resolve()
}

func datadogWebhookEntry(fields log.Fields, err error) *log.Entry {
	datadogWebhookLogger.SetLevel(log.StandardLogger().GetLevel())
	entry := log.NewEntry(datadogWebhookLogger)
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}
	entry = entry.WithFields(log.Fields{
		"type":        WebhookLogType,
		"integration": DatadogIntegration,
	})
	if err != nil {
		entry = entry.WithError(err)
	}
	return entry
}

func productiveWebhookEntry(event string, fields log.Fields, err error) *log.Entry {
	productiveWebhookLogger.SetLevel(log.StandardLogger().GetLevel())
	entry := log.NewEntry(productiveWebhookLogger)
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}
	entry = entry.WithFields(productiveWebhookFields(event))
	if err != nil {
		entry = entry.WithError(err)
	}
	return entry
}
