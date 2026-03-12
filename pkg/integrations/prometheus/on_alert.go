package prometheus

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAlert struct{}

var errWebhookAuthConfig = errors.New("failed to read webhook auth configuration")

type OnAlertConfiguration struct {
	Statuses   []string `json:"statuses" mapstructure:"statuses"`
	AlertNames []string `json:"alertNames" mapstructure:"alertNames"`
}

type OnAlertMetadata struct {
	WebhookURL         string `json:"webhookUrl" mapstructure:"webhookUrl"`
	WebhookAuthEnabled bool   `json:"webhookAuthEnabled,omitempty" mapstructure:"webhookAuthEnabled"`
}

type AlertmanagerWebhookPayload struct {
	Version           string              `json:"version"`
	GroupKey          string              `json:"groupKey"`
	Status            string              `json:"status"`
	Receiver          string              `json:"receiver"`
	GroupLabels       map[string]string   `json:"groupLabels"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	ExternalURL       string              `json:"externalURL"`
	TruncatedAlerts   int                 `json:"truncatedAlerts"`
	Alerts            []AlertmanagerAlert `json:"alerts"`
}

type AlertmanagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

func (t *OnAlert) Name() string {
	return "prometheus.onAlert"
}

func (t *OnAlert) Label() string {
	return "On Alert"
}

func (t *OnAlert) Description() string {
	return "Listen to Alertmanager webhook alert events"
}

func (t *OnAlert) Documentation() string {
	return `The On Alert trigger starts a workflow execution when Alertmanager sends alerts to SuperPlane.

## What this trigger does

- Receives Alertmanager webhook payloads
- Optionally validates bearer auth when **Webhook Secret** is configured
- Emits one event per matching alert as ` + "`prometheus.alert`" + `
- Filters by selected statuses (` + "`firing`" + ` and/or ` + "`resolved`" + `)

## Configuration

- **Statuses**: Required list of alert statuses to emit
- **Alert Names**: Optional exact ` + "`alertname`" + ` filters

## Alertmanager setup (manual)

When the node is saved, SuperPlane generates a webhook URL shown in the trigger setup panel. Copy that URL into your Alertmanager receiver.

Receiver registration in upstream Alertmanager is config-based (not API-created by SuperPlane). Use the setup instructions shown in the workflow sidebar for the exact ` + "`alertmanager.yml`" + ` snippet.

After updating Alertmanager config, reload it (for example ` + "`POST /-/reload`" + ` when lifecycle reload is enabled).`
}

func (t *OnAlert) Icon() string {
	return "prometheus"
}

func (t *OnAlert) Color() string {
	return "gray"
}

func (t *OnAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "statuses",
			Label:    "Statuses",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{AlertStateFiring},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Firing", Value: AlertStateFiring},
						{Label: "Resolved", Value: AlertStateResolved},
					},
				},
			},
			Description: "Only emit alerts with these statuses",
		},
		{
			Name:     "alertNames",
			Label:    "Alert Names",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Alert Name",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
			Default:     []string{"MyAlert"},
			Description: "Optional exact alertname filters",
		},
	}
}

func (t *OnAlert) Setup(ctx core.TriggerContext) error {
	metadata := OnAlertMetadata{}
	if ctx.Metadata != nil && ctx.Metadata.Get() != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
			return fmt.Errorf("failed to decode metadata: %w", err)
		}
	}

	if _, err := parseAndValidateOnAlertConfiguration(ctx.Configuration); err != nil {
		return err
	}

	if err := ctx.Integration.RequestWebhook(struct{}{}); err != nil {
		return err
	}

	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook URL: %w", err)
	}

	metadata.WebhookURL = webhookURL
	webhookBearerToken, _ := optionalIntegrationConfig(ctx.Integration, "webhookBearerToken")
	metadata.WebhookAuthEnabled = webhookBearerToken != ""

	if ctx.Metadata == nil {
		return nil
	}

	return ctx.Metadata.Set(metadata)
}

func (t *OnAlert) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAlert) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config, statusCode, err := parseOnAlertWebhookConfiguration(ctx.Configuration)
	if err != nil {
		return statusCode, err
	}

	if statusCode, err = authorizeOnAlertWebhook(ctx); err != nil {
		return statusCode, err
	}

	payload, statusCode, err := parseAlertmanagerWebhookPayload(ctx.Body)
	if err != nil {
		return statusCode, err
	}

	if err := emitMatchingAlerts(ctx.Events, config, payload); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (t *OnAlert) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func validateOnAlertConfiguration(config OnAlertConfiguration) error {
	normalizedStatuses := normalizeStatuses(config.Statuses)
	if len(normalizedStatuses) == 0 {
		return fmt.Errorf("at least one status must be selected")
	}

	for _, status := range normalizedStatuses {
		if status != AlertStateFiring && status != AlertStateResolved {
			return fmt.Errorf("invalid status %q, expected firing or resolved", status)
		}
	}

	return nil
}

func normalizeStatuses(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || slices.Contains(normalized, value) {
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized
}

func filterEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func validateWebhookAuth(ctx core.WebhookRequestContext) error {
	if ctx.Integration == nil {
		return nil
	}

	webhookBearerToken, err := optionalIntegrationConfig(ctx.Integration, "webhookBearerToken")
	if err != nil {
		return fmt.Errorf("%w: %v", errWebhookAuthConfig, err)
	}

	if webhookBearerToken == "" {
		return nil
	}

	authorization := ctx.Headers.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		return fmt.Errorf("missing bearer authorization")
	}

	token := authorization[len("Bearer "):]
	if subtle.ConstantTimeCompare([]byte(token), []byte(webhookBearerToken)) != 1 {
		return fmt.Errorf("invalid bearer token")
	}

	return nil
}

func optionalIntegrationConfig(integration core.IntegrationContext, name string) (string, error) {
	if integration == nil {
		return "", nil
	}

	value, err := integration.GetConfig(name)
	if err != nil {
		if isMissingIntegrationConfigError(err, name) {
			return "", nil
		}
		return "", err
	}

	return string(value), nil
}

func isMissingIntegrationConfigError(err error, name string) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, strings.ToLower(name)) && strings.Contains(message, "not found")
}

func buildAlertPayloadFromAlertmanager(alert AlertmanagerAlert, payload AlertmanagerWebhookPayload) map[string]any {
	output := map[string]any{
		"status":            alert.Status,
		"labels":            alert.Labels,
		"annotations":       alert.Annotations,
		"startsAt":          alert.StartsAt,
		"endsAt":            alert.EndsAt,
		"generatorURL":      alert.GeneratorURL,
		"fingerprint":       alert.Fingerprint,
		"receiver":          payload.Receiver,
		"groupKey":          payload.GroupKey,
		"groupLabels":       payload.GroupLabels,
		"commonLabels":      payload.CommonLabels,
		"commonAnnotations": payload.CommonAnnotations,
		"externalURL":       payload.ExternalURL,
	}

	if output["status"] == "" {
		output["status"] = payload.Status
	}

	return output
}

func buildAlertPayloadFromPrometheusAlert(alert PrometheusAlert) map[string]any {
	return map[string]any{
		"status":      alert.State,
		"labels":      alert.Labels,
		"annotations": alert.Annotations,
		"startsAt":    alert.ActiveAt,
		"value":       alert.Value,
	}
}

func sanitizeOnAlertConfiguration(config OnAlertConfiguration) OnAlertConfiguration {
	for i := range config.Statuses {
		config.Statuses[i] = strings.ToLower(strings.TrimSpace(config.Statuses[i]))
	}

	for i := range config.AlertNames {
		config.AlertNames[i] = strings.TrimSpace(config.AlertNames[i])
	}

	return config
}

func parseOnAlertWebhookConfiguration(configuration any) (OnAlertConfiguration, int, error) {
	config, err := parseAndValidateOnAlertConfiguration(configuration)
	if err != nil {
		return OnAlertConfiguration{}, http.StatusInternalServerError, err
	}

	return config, http.StatusOK, nil
}

func authorizeOnAlertWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := validateWebhookAuth(ctx); err != nil {
		if errors.Is(err, errWebhookAuthConfig) {
			return http.StatusInternalServerError, err
		}
		return http.StatusForbidden, err
	}

	return http.StatusOK, nil
}

func parseAlertmanagerWebhookPayload(body []byte) (AlertmanagerWebhookPayload, int, error) {
	payload := AlertmanagerWebhookPayload{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return AlertmanagerWebhookPayload{}, http.StatusBadRequest, fmt.Errorf("failed to parse request body: %w", err)
	}

	return payload, http.StatusOK, nil
}

func emitMatchingAlerts(events core.EventContext, config OnAlertConfiguration, payload AlertmanagerWebhookPayload) error {
	filteredNames := filterEmpty(config.AlertNames)

	for _, alert := range payload.Alerts {
		alertStatus := alert.Status
		if alertStatus == "" {
			alertStatus = payload.Status
		}

		if !containsStatus(config.Statuses, alertStatus) {
			continue
		}

		alertName := alert.Labels["alertname"]
		if len(filteredNames) > 0 && !slices.Contains(filteredNames, alertName) {
			continue
		}

		if err := events.Emit(PrometheusAlertPayloadType, buildAlertPayloadFromAlertmanager(alert, payload)); err != nil {
			return fmt.Errorf("failed to emit alert event: %w", err)
		}
	}

	return nil
}

func parseAndValidateOnAlertConfiguration(configuration any) (OnAlertConfiguration, error) {
	config := OnAlertConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return OnAlertConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = sanitizeOnAlertConfiguration(config)
	if err := validateOnAlertConfiguration(config); err != nil {
		return OnAlertConfiguration{}, err
	}
	config.Statuses = normalizeStatuses(config.Statuses)

	return config, nil
}

func containsStatus(allowed []string, state string) bool {
	for _, value := range allowed {
		if strings.EqualFold(value, state) {
			return true
		}
	}

	return false
}
