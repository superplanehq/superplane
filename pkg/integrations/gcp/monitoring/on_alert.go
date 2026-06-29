package monitoring

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// roleHintChannelEditor is the IAM role needed to create/delete the webhook
// notification channel this trigger provisions.
const roleHintChannelEditor = "roles/monitoring.notificationChannelEditor (or roles/monitoring.editor)"

// AlertPayloadType is the event type emitted for each matching incident.
const AlertPayloadType = "gcp.monitoring.alert"

const (
	incidentStateOpen   = "open"
	incidentStateClosed = "closed"
)

type OnAlert struct{}

type OnAlertConfiguration struct {
	States []string `json:"states" mapstructure:"states"`
}

type OnAlertMetadata struct {
	WebhookURL          string `json:"webhookUrl,omitempty" mapstructure:"webhookUrl"`
	NotificationChannel string `json:"notificationChannel,omitempty" mapstructure:"notificationChannel"`
}

func (t *OnAlert) Name() string {
	return "gcp.monitoring.onAlert"
}

func (t *OnAlert) Label() string {
	return "Monitoring • On Alert"
}

func (t *OnAlert) Description() string {
	return "Trigger a workflow when a Cloud Monitoring alerting policy opens or closes an incident"
}

func (t *OnAlert) Documentation() string {
	return `The On Alert trigger starts a workflow execution when a Cloud Monitoring alerting policy fires (opens) or resolves (closes) an incident.

## Trigger behavior

When this trigger is set up, SuperPlane automatically creates a **webhook notification channel** in Cloud Monitoring that points back at SuperPlane. Cloud Monitoring POSTs the incident to SuperPlane whenever a policy attached to that channel changes state.

To route a policy's incidents here, attach this trigger's notification channel to the policy via the **Create Alerting Policy** or **Update Alerting Policy** component's *Notification Channels* field. The channel's resource name is shown on the node after setup.

## Configuration

- **States**: Which incident states to emit on — ` + "`open`" + ` (fired) and/or ` + "`closed`" + ` (resolved). Defaults to ` + "`open`" + `.

## Event Data

Emits one ` + "`gcp.monitoring.alert`" + ` event per matching incident, including the incident id, state, policy and condition names, the affected resource and metric, the observed/threshold values, and the incident URL.

## Important Notes

- Requires the ` + "`roles/monitoring.notificationChannelEditor`" + ` (or ` + "`roles/monitoring.editor`" + `) IAM role so SuperPlane can create the webhook channel.
- Removing the trigger deletes the webhook notification channel it created.`
}

func (t *OnAlert) Icon() string {
	return "bell"
}

func (t *OnAlert) Color() string {
	return "blue"
}

func (t *OnAlert) ExampleData() map[string]any {
	return onAlertExampleData()
}

func (t *OnAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "states",
			Label:    "States",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{incidentStateOpen},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open (fired)", Value: incidentStateOpen},
						{Label: "Closed (resolved)", Value: incidentStateClosed},
					},
				},
			},
			Description: "Only emit incidents in these states",
		},
	}
}

func (t *OnAlert) Setup(ctx core.TriggerContext) error {
	if _, err := parseOnAlertConfiguration(ctx.Configuration); err != nil {
		return err
	}

	metadata := OnAlertMetadata{}
	if ctx.Metadata != nil && ctx.Metadata.Get() != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
			return fmt.Errorf("failed to decode metadata: %w", err)
		}
	}

	if ctx.Integration == nil {
		return fmt.Errorf("a connected GCP integration is required to set up the On Alert trigger")
	}
	if err := ctx.Integration.RequestWebhook(struct{}{}); err != nil {
		return err
	}
	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}
	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to set up webhook URL: %w", err)
	}
	metadata.WebhookURL = webhookURL

	// The node webhook secret is the channel's Basic-auth password, so Cloud
	// Monitoring signs every incident delivery (verified in HandleWebhook).
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return fmt.Errorf("failed to read webhook secret: %w", err)
	}
	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	if metadata.NotificationChannel == "" {
		// First setup for this node: create the signed webhook channel.
		channelName, err := createWebhookChannel(client, webhookURL, string(secret))
		if err != nil {
			return err
		}
		metadata.NotificationChannel = channelName
	} else {
		// The channel already exists: resync its URL and Basic-auth password so a
		// moved node webhook URL or a rotated webhook secret keeps signed
		// deliveries flowing, instead of Cloud Monitoring POSTing to a stale URL
		// or with an outdated password.
		if err := updateWebhookChannel(client, metadata.NotificationChannel, webhookURL, string(secret)); err != nil {
			return err
		}
	}

	if ctx.Metadata == nil {
		return nil
	}
	return ctx.Metadata.Set(metadata)
}

func (t *OnAlert) Cleanup(ctx core.TriggerContext) error {
	metadata := OnAlertMetadata{}
	if ctx.Metadata == nil || ctx.Metadata.Get() == nil {
		return nil
	}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.NotificationChannel == "" {
		return nil
	}

	if ctx.Integration == nil {
		// The integration is already gone, so we can't (and needn't) call Cloud
		// Monitoring to delete the channel. Don't block removing the node on a
		// missing integration — best-effort cleanup.
		return nil
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}
	_, err = client.DeleteURL(context.Background(), fmt.Sprintf("%s/%s", monitoringBaseURL, metadata.NotificationChannel))
	if err != nil {
		return fmt.Errorf("%s", apiErrorMessage("failed to delete notification channel", roleHintChannelEditor, err))
	}
	return nil
}

func (t *OnAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnAlert) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := authenticateWebhook(ctx); err != nil {
		return http.StatusUnauthorized, nil, err
	}

	config, err := parseOnAlertConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	var payload incidentWebhookPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse incident payload: %w", err)
	}

	// Cloud Monitoring sends a verification ping with no incident when the
	// channel is created; acknowledge it without emitting.
	if payload.Incident == nil || payload.Incident.IncidentID == "" {
		return http.StatusOK, nil, nil
	}

	state := strings.ToLower(strings.TrimSpace(payload.Incident.State))
	if !containsFold(config.States, state) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit(AlertPayloadType, buildAlertPayload(payload.Incident)); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to emit alert event: %w", err)
	}
	return http.StatusOK, nil, nil
}

// authenticateWebhook verifies the HTTP Basic credentials that Cloud Monitoring
// sends (configured on the webhook_basicauth channel) against the node's webhook
// secret, so only Cloud Monitoring can deliver incidents to this node.
func authenticateWebhook(ctx core.WebhookRequestContext) error {
	if ctx.Webhook == nil {
		return nil
	}
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		// Fail closed: a transient decrypt/lookup error must not be treated as
		// "no secret configured" and let an unauthenticated request through.
		return fmt.Errorf("failed to read webhook secret: %w", err)
	}
	if len(secret) == 0 {
		// No secret to verify against (e.g. a legacy channel) — don't block.
		return nil
	}

	const prefix = "Basic "
	authHeader := ctx.Headers.Get("Authorization")
	if !strings.HasPrefix(authHeader, prefix) {
		return fmt.Errorf("missing webhook credentials")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(authHeader[len(prefix):]))
	if err != nil {
		return fmt.Errorf("invalid webhook credentials encoding")
	}
	_, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return fmt.Errorf("invalid webhook credentials format")
	}
	if subtle.ConstantTimeCompare([]byte(password), secret) != 1 {
		return fmt.Errorf("invalid webhook credentials")
	}
	return nil
}

// webhookAuthUsername is the fixed Basic-auth username configured on the
// notification channel; only the password (the node webhook secret) is verified.
const webhookAuthUsername = "superplane"

// createWebhookChannel creates a webhook_basicauth notification channel pointing
// at the SuperPlane node webhook URL and returns its resource name. The node
// webhook secret is stored as the channel's Basic-auth password so Cloud
// Monitoring signs every incident delivery, which HandleWebhook verifies.
func createWebhookChannel(client Client, webhookURL, secret string) (string, error) {
	body := map[string]any{
		"type":        "webhook_basicauth",
		"displayName": "SuperPlane On Alert trigger",
		"description": "Auto-created by SuperPlane to deliver Cloud Monitoring incidents to a workflow.",
		"labels": map[string]any{
			"url":      webhookURL,
			"username": webhookAuthUsername,
			"password": secret,
		},
		"enabled": true,
	}
	respBody, err := client.PostURL(
		context.Background(),
		fmt.Sprintf("%s/projects/%s/notificationChannels", monitoringBaseURL, client.ProjectID()),
		body,
	)
	if err != nil {
		return "", fmt.Errorf("%s", apiErrorMessage("failed to create notification channel", roleHintChannelEditor, err))
	}
	var created struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("failed to parse notification channel response: %w", err)
	}
	if strings.TrimSpace(created.Name) == "" {
		return "", fmt.Errorf("notification channel response missing name")
	}
	return created.Name, nil
}

// updateWebhookChannel resyncs an existing webhook notification channel's URL and
// Basic-auth credentials, so a moved node webhook URL or a rotated webhook secret
// doesn't leave Cloud Monitoring posting incidents to a stale endpoint or with an
// outdated password.
func updateWebhookChannel(client Client, channelName, webhookURL, secret string) error {
	body := map[string]any{
		"labels": map[string]any{
			"url":      webhookURL,
			"username": webhookAuthUsername,
			"password": secret,
		},
	}
	url := fmt.Sprintf("%s/%s?updateMask=labels.url,labels.username,labels.password", monitoringBaseURL, channelName)
	if _, err := client.PatchURL(context.Background(), url, body); err != nil {
		return fmt.Errorf("%s", apiErrorMessage("failed to update notification channel", roleHintChannelEditor, err))
	}
	return nil
}

type incidentWebhookPayload struct {
	Version  string    `json:"version"`
	Incident *incident `json:"incident"`
}

type incident struct {
	IncidentID       string `json:"incident_id"`
	ScopingProjectID string `json:"scoping_project_id"`
	URL              string `json:"url"`
	State            string `json:"state"`
	StartedAt        int64  `json:"started_at"`
	// EndedAt is nullable: Cloud Monitoring sends "ended_at": null for incidents
	// that are still open, so decode it as a pointer and omit it from the emitted
	// payload when absent rather than reporting a misleading epoch (0) timestamp.
	EndedAt             *int64          `json:"ended_at"`
	Summary             string          `json:"summary"`
	ResourceName        string          `json:"resource_name"`
	ResourceDisplayName string          `json:"resource_display_name"`
	PolicyName          string          `json:"policy_name"`
	ConditionName       string          `json:"condition_name"`
	ObservedValue       string          `json:"observed_value"`
	ThresholdValue      string          `json:"threshold_value"`
	Metric              *incidentMetric `json:"metric"`
}

type incidentMetric struct {
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
}

func buildAlertPayload(i *incident) map[string]any {
	payload := map[string]any{
		"incidentId":          i.IncidentID,
		"state":               strings.ToLower(strings.TrimSpace(i.State)),
		"policyName":          i.PolicyName,
		"conditionName":       i.ConditionName,
		"summary":             i.Summary,
		"url":                 i.URL,
		"resourceName":        i.ResourceName,
		"resourceDisplayName": i.ResourceDisplayName,
		"observedValue":       i.ObservedValue,
		"thresholdValue":      i.ThresholdValue,
		"scopingProjectId":    i.ScopingProjectID,
		"startedAt":           i.StartedAt,
	}
	// Only present once the incident has resolved; open incidents have no end.
	if i.EndedAt != nil {
		payload["endedAt"] = *i.EndedAt
	}
	if i.Metric != nil {
		payload["metricType"] = i.Metric.Type
		payload["metricDisplayName"] = i.Metric.DisplayName
	}
	return payload
}

func parseOnAlertConfiguration(cfg any) (OnAlertConfiguration, error) {
	config := OnAlertConfiguration{}
	if err := mapstructure.Decode(cfg, &config); err != nil {
		return OnAlertConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	states := make([]string, 0, len(config.States))
	for _, s := range config.States {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || containsFold(states, s) {
			continue
		}
		if s != incidentStateOpen && s != incidentStateClosed {
			return OnAlertConfiguration{}, fmt.Errorf("invalid state %q, expected open or closed", s)
		}
		states = append(states, s)
	}
	if len(states) == 0 {
		// No states configured — fall back to the documented default ("open"),
		// matching the field schema's Default so a node saved without an explicit
		// selection still emits on fired incidents.
		states = []string{incidentStateOpen}
	}
	config.States = states
	return config, nil
}

func containsFold(values []string, target string) bool {
	for _, v := range values {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}
