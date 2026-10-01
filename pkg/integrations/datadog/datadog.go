package datadog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ResourceTypeService = "service"

const (
	datadogWebhookOutcomeRejected  = "rejected"
	datadogWebhookOutcomeFailed    = "failed"
	datadogWebhookOutcomeIgnored   = "ignored"
	datadogWebhookOutcomeReceived  = "received"
	datadogWebhookOutcomeDelivered = "delivered"
	datadogEventTypeUnknown        = "unknown"
)

const installationInstructions = `
To configure Datadog to work with SuperPlane:

1. **Get API Key**: In Datadog, go to Organization Settings > API Keys and copy an API Key. The API key has no scopes.
2. **Get Application Key**: Go to Organization Settings > Application Keys and create an Application Key.
3. **Grant the minimum permissions**: If you restrict the application key, grant only these permissions:

- ` + "`create_webhooks`" + ` creates the SuperPlane webhook.
- ` + "`manage_integrations`" + ` updates and removes that webhook.
- ` + "`error_tracking_read`" + ` lists Error Tracking issues and reads issue details for an alert.

Grant these optional permissions to add the error sample and related logs to each task:

- ` + "`apm_read`" + ` reads the sample error span: trace, request, user, and environment.
- ` + "`logs_read_data`" + ` reads log samples and the logs that share the sample trace.
- ` + "`rum_apps_read`" + ` reads the sample for browser and mobile errors.

An unrestricted application key also works.
4. **Select Site**: Choose the Datadog site that matches your account (US1, US3, US5, EU, or AP1)
5. **Enter Credentials**: Provide your API Key, Application Key, and Site in the integration configuration
6. **Add the webhook to a monitor**: In an Error Tracking New Issue monitor, add ` + "`@webhook-superplane`" + ` to the notification message
`

func init() {
	registry.RegisterIntegration("datadog", &Datadog{})
}

type Datadog struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
	AppKey string `json:"appKey"`
	Site   string `json:"site"`
}

func (d *Datadog) Name() string {
	return "datadog"
}

func (d *Datadog) Label() string {
	return "Datadog"
}

func (d *Datadog) Icon() string {
	return "chart-bar"
}

func (d *Datadog) Description() string {
	return "React to Error Tracking alerts and create events in Datadog"
}

func (d *Datadog) Instructions() string {
	return installationInstructions
}

func (d *Datadog) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "site",
			Label:    "Datadog Site",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "datadoghq.com",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "US1 (datadoghq.com)", Value: "datadoghq.com"},
						{Label: "US3 (us3.datadoghq.com)", Value: "us3.datadoghq.com"},
						{Label: "US5 (us5.datadoghq.com)", Value: "us5.datadoghq.com"},
						{Label: "EU (datadoghq.eu)", Value: "datadoghq.eu"},
						{Label: "AP1 (ap1.datadoghq.com)", Value: "ap1.datadoghq.com"},
					},
				},
			},
		},
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Datadog API Key for authentication",
		},
		{
			Name:        "appKey",
			Label:       "Application Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "A restricted key needs create_webhooks, manage_integrations, and error_tracking_read. Add apm_read, logs_read_data, and rum_apps_read to include the error sample and related logs.",
		},
	}
}

func (d *Datadog) Actions() []core.Action {
	return []core.Action{
		&CreateEvent{},
	}
}

func (d *Datadog) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnErrorTrackingAlert{},
	}
}

func (d *Datadog) Cleanup(ctx core.IntegrationCleanupContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to create datadog client during cleanup: %v", err)
		}
		return nil
	}

	if err := deleteWebhook(client); err != nil && ctx.Logger != nil {
		ctx.Logger.Warnf("failed to delete datadog webhook during cleanup: %v", err)
	}

	return nil
}

func (d *Datadog) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.Site == "" {
		return fmt.Errorf("site is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	if config.AppKey == "" {
		return fmt.Errorf("appKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateCredentials()
	if err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	if err := reconcileWebhook(ctx, client); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (d *Datadog) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	if ctx.Request.Method != http.MethodPost {
		ctx.Response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := verifyWebhookRequest(ctx.Integration, ctx.Request); err != nil {
		logDatadogWebhookRejected(ctx, err)
		setWebhookReceipt(ctx.Request, models.DatadogWebhookOutcomeRejected, 0)
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	body, err := readWebhookBody(ctx.Request.Body)
	if err != nil {
		if errors.Is(err, errWebhookBodyTooLarge) {
			logDatadogWebhookFailed(ctx, datadogEventTypeUnknown, "", nil, err)
			setWebhookReceipt(ctx.Request, models.DatadogWebhookOutcomeFailed, 0)
			ctx.Response.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		logDatadogWebhookFailed(ctx, datadogEventTypeUnknown, "", nil, err)
		setWebhookReceipt(ctx.Request, models.DatadogWebhookOutcomeFailed, 0)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		logDatadogWebhookFailed(ctx, datadogEventTypeUnknown, "", body, err)
		setWebhookReceipt(ctx.Request, models.DatadogWebhookOutcomeFailed, 0)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	eventType, alertTransition := datadogEventKind(payload)
	if !shouldDispatchErrorTrackingAlert(payload) {
		logDatadogWebhookIgnored(ctx, eventType, alertTransition, body)
		setWebhookReceipt(ctx.Request, models.DatadogWebhookOutcomeIgnored, 0)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	stampWebhookReceipt(payload, ctx.Request)
	subscriptionCount, delivered, sendErr := d.dispatchWebhookMessage(ctx, payload)
	if delivered == 0 && sendErr != nil {
		logDatadogWebhookFailed(ctx, eventType, alertTransition, body, sendErr)
		setWebhookReceipt(ctx.Request, models.DatadogWebhookOutcomeFailed, subscriptionCount)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if sendErr != nil {
		logDatadogWebhookPartialFailure(ctx, eventType, alertTransition, body, sendErr)
		setWebhookReceipt(ctx.Request, models.DatadogWebhookOutcomeAccepted, subscriptionCount)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	logDatadogWebhookReceived(ctx, eventType, alertTransition, body, nil)
	outcome := models.DatadogWebhookOutcomeAccepted
	if subscriptionCount == 0 {
		outcome = models.DatadogWebhookOutcomeNoSubscription
	}
	setWebhookReceipt(ctx.Request, outcome, subscriptionCount)
	ctx.Response.WriteHeader(http.StatusOK)
}

func datadogEventKind(payload map[string]any) (string, string) {
	eventType, _ := payload["event_type"].(string)
	alertTransition, _ := payload["alert_transition"].(string)
	return eventType, alertTransition
}

func logDatadogWebhookRejected(ctx core.HTTPRequestContext, err error) {
	logging.LogDatadogWebhookWarn(
		"Datadog webhook rejected",
		datadogReceiptFields(ctx, datadogEventTypeUnknown, "", datadogWebhookOutcomeRejected, nil),
		err,
	)
}

func logDatadogWebhookFailed(ctx core.HTTPRequestContext, eventType, alertTransition string, body []byte, err error) {
	logging.LogDatadogWebhookError(
		"Datadog webhook failed",
		datadogReceiptFields(ctx, eventType, alertTransition, datadogWebhookOutcomeFailed, body),
		err,
	)
}

func logDatadogWebhookIgnored(ctx core.HTTPRequestContext, eventType, alertTransition string, body []byte) {
	logging.LogDatadogWebhookInfo(
		"Datadog webhook ignored",
		datadogReceiptFields(ctx, eventType, alertTransition, datadogWebhookOutcomeIgnored, body),
		nil,
	)
}

func logDatadogWebhookReceived(ctx core.HTTPRequestContext, eventType, alertTransition string, body []byte, err error) {
	logging.LogDatadogWebhookInfo(
		"Datadog webhook received",
		datadogReceiptFields(ctx, eventType, alertTransition, datadogWebhookOutcomeReceived, body),
		err,
	)
}

func logDatadogWebhookPartialFailure(ctx core.HTTPRequestContext, eventType, alertTransition string, body []byte, err error) {
	logging.LogDatadogWebhookError(
		"Datadog webhook received",
		datadogReceiptFields(ctx, eventType, alertTransition, datadogWebhookOutcomeReceived, body),
		err,
	)
}

func datadogReceiptFields(ctx core.HTTPRequestContext, eventType, alertTransition, outcome string, body []byte) log.Fields {
	integrationID := ""
	if ctx.Integration != nil {
		integrationID = ctx.Integration.ID().String()
	}

	return logging.WithWebhookPayload(log.Fields{
		"outcome":           outcome,
		"event_type":        eventType,
		"alert_transition":  alertTransition,
		"organization_id":   ctx.OrganizationID,
		"organization_name": "",
		"integration_id":    integrationID,
		"workspace_id":      "",
		"workspace_name":    "",
		"intake_id":         "",
		"intake_name":       "",
	}, body)
}

var errWebhookBodyTooLarge = errors.New("webhook body is too large")

func readWebhookBody(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	payload, err := io.ReadAll(io.LimitReader(body, int64(config.MaxWebhookPayloadSize)+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > config.MaxWebhookPayloadSize {
		return nil, errWebhookBodyTooLarge
	}
	return payload, nil
}

func shouldDispatchErrorTrackingAlert(payload map[string]any) bool {
	eventType, _ := payload["event_type"].(string)
	if !strings.EqualFold(eventType, ErrorTrackingAlertEventType) {
		return false
	}

	transition, _ := payload["alert_transition"].(string)
	return isDispatchableAlertTransition(transition)
}

func isDispatchableAlertTransition(transition string) bool {
	transition = strings.TrimSpace(transition)
	return strings.EqualFold(transition, AlertTransitionTriggered) ||
		strings.EqualFold(transition, AlertTransitionRetriggered)
}

func (d *Datadog) dispatchWebhookMessage(ctx core.HTTPRequestContext, payload map[string]any) (subscriptionCount int, delivered int, sendErr error) {
	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list datadog subscriptions: %w", err)
	}

	subscriptionCount = len(subscriptions)
	for _, subscription := range subscriptions {
		if err := subscription.SendMessage(payload); err != nil {
			sendErr = errors.Join(sendErr, err)
			continue
		}
		delivered++
	}

	// A retry repeats the whole webhook. Return an error only when no
	// subscription accepted the alert, so a later retry cannot duplicate a
	// delivery that already succeeded.
	if delivered == 0 && sendErr != nil {
		return subscriptionCount, delivered, sendErr
	}
	return subscriptionCount, delivered, sendErr
}

func (d *Datadog) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeService, ResourceTypeEnvironment:
	default:
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	if resourceType == ResourceTypeEnvironment {
		names, err := client.ListEnvironments(ctx.Parameters["service"])
		if err != nil {
			return nil, err
		}
		return environmentResources(names), nil
	}

	issues, err := client.SearchErrorTrackingIssues("*", maxErrorTrackingSearchLimit)
	if err != nil {
		return nil, err
	}

	return serviceResources(issues), nil
}

func serviceResources(issues []ErrorTrackingIssue) []core.IntegrationResource {
	seen := map[string]bool{}
	names := make([]string, 0, len(issues))
	for _, issue := range issues {
		name := strings.TrimSpace(issue.Service)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	slices.Sort(names)

	resources := make([]core.IntegrationResource, 0, len(names))
	for _, name := range names {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeService,
			ID:   name,
			Name: name,
		})
	}
	return resources
}

func (d *Datadog) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *Datadog) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
