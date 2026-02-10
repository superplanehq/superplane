package prometheus

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAlert struct{}

type OnAlertConfiguration struct {
	Statuses   []string `json:"statuses" mapstructure:"statuses"`
	AlertNames []string `json:"alertNames" mapstructure:"alertNames"`
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
- Validates optional webhook authentication (` + "`none`" + `, ` + "`bearer`" + `, ` + "`basic`" + `)
- Emits one event per matching alert as ` + "`prometheus.alert`" + `
- Filters by selected statuses (` + "`firing`" + ` and/or ` + "`resolved`" + `)

## Configuration

- **Statuses**: Required list of alert statuses to emit
- **Alert Names**: Optional exact ` + "`alertname`" + ` filters

## Alertmanager setup (manual)

Receiver registration in upstream Alertmanager is config-based (not API-created by SuperPlane).

` + "```yaml" + `
receivers:
  - name: superplane
    webhook_configs:
      - url: https://<superplane-host>/api/v1/webhooks/<webhook-id>
        send_resolved: true
        # Optional bearer auth
        # http_config:
        #   authorization:
        #     type: Bearer
        #     credentials: <webhook-bearer-token>
        # Optional basic auth
        # http_config:
        #   basic_auth:
        #     username: <webhook-username>
        #     password: <webhook-password>

route:
  receiver: superplane
` + "```" + `

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
			Description: "Optional exact alertname filters",
		},
	}
}

func (t *OnAlert) Setup(ctx core.TriggerContext) error {
	config := OnAlertConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = sanitizeOnAlertConfigurationFromSetup(config)

	if err := validateOnAlertConfiguration(config); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(struct{}{})
}

func (t *OnAlert) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAlert) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnAlertConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateOnAlertConfiguration(config); err != nil {
		return http.StatusInternalServerError, err
	}

	if err := validateWebhookAuth(ctx); err != nil {
		return http.StatusForbidden, err
	}

	payload := AlertmanagerWebhookPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse request body: %w", err)
	}

	filteredNames := filterEmpty(config.AlertNames)

	emitted := 0
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

		if err := ctx.Events.Emit(PrometheusAlertPayloadType, buildAlertPayloadFromAlertmanager(alert, payload)); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to emit alert event: %w", err)
		}
		emitted++
	}

	if emitted == 0 {
		return http.StatusOK, nil
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
	authConfig, err := getWebhookRuntimeConfiguration(ctx.Webhook)
	if err != nil {
		return err
	}

	authorization := ctx.Headers.Get("Authorization")
	switch authConfig.AuthType {
	case AuthTypeNone:
		return nil
	case AuthTypeBearer:
		if !strings.HasPrefix(authorization, "Bearer ") {
			return fmt.Errorf("missing bearer authorization")
		}

		token := authorization[len("Bearer "):]
		if subtle.ConstantTimeCompare([]byte(token), []byte(authConfig.BearerToken)) != 1 {
			return fmt.Errorf("invalid bearer token")
		}
		return nil
	case AuthTypeBasic:
		username, password, err := decodeBasicAuthHeader(authorization)
		if err != nil {
			return err
		}

		if subtle.ConstantTimeCompare([]byte(username), []byte(authConfig.Username)) != 1 {
			return fmt.Errorf("invalid basic auth credentials")
		}
		if subtle.ConstantTimeCompare([]byte(password), []byte(authConfig.Password)) != 1 {
			return fmt.Errorf("invalid basic auth credentials")
		}
		return nil
	default:
		return fmt.Errorf("unsupported webhook authType %q", authConfig.AuthType)
	}
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

func sanitizeOnAlertConfigurationFromSetup(config OnAlertConfiguration) OnAlertConfiguration {
	for i := range config.Statuses {
		config.Statuses[i] = strings.ToLower(strings.TrimSpace(config.Statuses[i]))
	}

	for i := range config.AlertNames {
		config.AlertNames[i] = strings.TrimSpace(config.AlertNames[i])
	}

	return config
}

func containsStatus(allowed []string, state string) bool {
	for _, value := range allowed {
		if strings.EqualFold(value, state) {
			return true
		}
	}

	return false
}
