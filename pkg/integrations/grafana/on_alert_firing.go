package grafana

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAlertFiring struct{}

type OnAlertFiringConfig struct {
	// SharedSecret is kept for backward compatibility with older trigger
	// configurations. New triggers use an auto-generated webhook secret
	// managed in webhook storage instead of user-provided configuration.
	SharedSecret      string                    `json:"sharedSecret,omitempty"`
	AlertNames        []configuration.Predicate `json:"alertNames,omitempty" mapstructure:"alertNames"`
	WebhookBindingKey string                    `json:"webhookBindingKey,omitempty" mapstructure:"webhookBindingKey"`
}

func (t *OnAlertFiring) Name() string {
	return "grafana.onAlertFiring"
}

func (t *OnAlertFiring) Label() string {
	return "On Alert Firing"
}

func (t *OnAlertFiring) Description() string {
	return "Trigger when a Grafana alert rule is firing"
}

func (t *OnAlertFiring) Documentation() string {
	return `The On Alert Firing trigger starts a workflow when Grafana Unified Alerting sends a firing alert webhook.

## Setup

1. SuperPlane automatically creates or updates a Grafana Webhook contact point and notification policy route for this trigger when provisioning succeeds.
2. SuperPlane manages webhook bearer authentication automatically.
3. If Grafana provisioning is unavailable or rejected, manual setup may still be required.

## Configuration

- **Alert Names**: Optional exact alert name filters

## Event Data

The trigger emits the full Grafana webhook payload, including:
- status (firing/resolved)
- alerts array with labels and annotations
- groupLabels, commonLabels, commonAnnotations
- externalURL and other alerting metadata
`
}

func (t *OnAlertFiring) Icon() string {
	return "alert-triangle"
}

func (t *OnAlertFiring) Color() string {
	return "gray"
}

func (t *OnAlertFiring) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alertNames",
			Label:       "Alert Names",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Only fire for alerts whose name equals or matches one of these predicates. Leave empty to accept all alerts.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (t *OnAlertFiring) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("missing integration context")
	}

	config := OnAlertFiringConfig{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	config = sanitizeOnAlertFiringConfig(config)
	if err := validateOnAlertFiringConfig(config); err != nil {
		return err
	}

	bindingKey := getWebhookBindingKey(ctx)
	if bindingKey == "" {
		bindingKey = uuid.NewString()
	}

	requestConfig := OnAlertFiringConfig{
		SharedSecret:      strings.TrimSpace(config.SharedSecret),
		WebhookBindingKey: bindingKey,
		AlertNames:        config.AlertNames,
	}

	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	if err := ctx.Integration.RequestWebhook(requestConfig); err != nil {
		return err
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return err
	}

	if ctx.Metadata != nil {
		if err := setWebhookMetadata(ctx, webhookURL, bindingKey); err != nil && ctx.Logger != nil {
			ctx.Logger.Warnf("grafana onAlertFiring: failed to store webhook url metadata: %v", err)
		}
	}

	return nil
}

func (t *OnAlertFiring) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAlertFiring) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("action %s not supported", ctx.Name)
}

func (t *OnAlertFiring) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	sharedSecret, err := resolveWebhookSharedSecret(ctx)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if sharedSecret != "" {
		authHeader := strings.TrimSpace(ctx.Headers.Get("Authorization"))
		if authHeader == "" {
			return http.StatusUnauthorized, nil, fmt.Errorf("missing Authorization header")
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return http.StatusUnauthorized, nil, fmt.Errorf("invalid Authorization header")
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if subtle.ConstantTimeCompare([]byte(token), []byte(sharedSecret)) != 1 {
			return http.StatusUnauthorized, nil, fmt.Errorf("invalid Authorization token")
		}
	}

	if len(ctx.Body) == 0 {
		return http.StatusBadRequest, nil, fmt.Errorf("empty body")
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	config, err := parseAndValidateOnAlertFiringConfig(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if !isFiringAlert(payload) {
		return http.StatusOK, nil, nil
	}
	if !matchesOnAlertFiringFilters(config, payload) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("grafana.alert.firing", payload); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnAlertFiring) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func isFiringAlert(payload map[string]any) bool {
	return strings.EqualFold(extractString(payload["status"]), "firing")
}

func extractString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func resolveWebhookSharedSecret(ctx core.WebhookRequestContext) (string, error) {
	if ctx.Webhook != nil {
		secret, err := ctx.Webhook.GetSecret()
		if err != nil {
			return "", fmt.Errorf("error getting webhook secret: %w", err)
		}

		normalizedSecret := strings.TrimSpace(string(secret))
		if normalizedSecret != "" {
			return normalizedSecret, nil
		}
	}

	// Backward compatibility for older records where sharedSecret lived in configuration.
	config := OnAlertFiringConfig{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return "", fmt.Errorf("error decoding configuration: %v", err)
	}

	return strings.TrimSpace(config.SharedSecret), nil
}

func getWebhookBindingKey(ctx core.TriggerContext) string {
	if ctx.Metadata == nil {
		return ""
	}

	existing := ctx.Metadata.Get()
	existingMap, ok := existing.(map[string]any)
	if !ok {
		return ""
	}

	return strings.TrimSpace(extractString(existingMap["webhookBindingKey"]))
}

func setWebhookMetadata(ctx core.TriggerContext, webhookURL, bindingKey string) error {
	metadata := map[string]any{}
	if existing := ctx.Metadata.Get(); existing != nil {
		if existingMap, ok := existing.(map[string]any); ok {
			for key, value := range existingMap {
				metadata[key] = value
			}
		}
	}

	metadata["webhookUrl"] = webhookURL
	metadata["webhookBindingKey"] = bindingKey
	return ctx.Metadata.Set(metadata)
}

func parseAndValidateOnAlertFiringConfig(configuration any) (OnAlertFiringConfig, error) {
	config := OnAlertFiringConfig{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return OnAlertFiringConfig{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	config = sanitizeOnAlertFiringConfig(config)
	if err := validateOnAlertFiringConfig(config); err != nil {
		return OnAlertFiringConfig{}, err
	}

	return config, nil
}

func sanitizeOnAlertFiringConfig(config OnAlertFiringConfig) OnAlertFiringConfig {
	return config
}

func validateOnAlertFiringConfig(config OnAlertFiringConfig) error {
	for _, p := range config.AlertNames {
		if p.Type == configuration.PredicateTypeMatches {
			if _, err := regexp.Compile(p.Value); err != nil {
				return fmt.Errorf("alertNames matches predicate: invalid regex: %w", err)
			}
		}
	}
	if combined, ok := combinedPositiveAlertNameRegex(config.AlertNames); ok {
		if _, err := regexp.Compile(combined); err != nil {
			return fmt.Errorf("alertNames: combined positive regex is invalid: %w", err)
		}
	}
	return nil
}

func matchesOnAlertFiringFilters(config OnAlertFiringConfig, payload map[string]any) bool {
	if len(config.AlertNames) == 0 {
		return true
	}

	names := extractAlertLabelNames(payload)
	if len(names) == 0 {
		// Grafana notification policies match on the alertname label only; without it we cannot
		// align with the provisioned route, so do not emit when filters are configured.
		return false
	}

	for _, name := range names {
		if alertLabelNameMatchesPredicates(name, config.AlertNames) {
			return true
		}
	}

	return false
}

// extractAlertLabelNames returns distinct alertname label values (not the human title), matching
// Grafana object_matchers on alertname.
func extractAlertLabelNames(payload map[string]any) []string {
	names := []string{}

	if commonLabels, ok := payload["commonLabels"].(map[string]any); ok {
		if alertName := extractString(commonLabels["alertname"]); alertName != "" {
			names = append(names, alertName)
		}
	}

	if alerts, ok := payload["alerts"].([]any); ok {
		for _, alert := range alerts {
			alertMap, ok := alert.(map[string]any)
			if !ok {
				continue
			}

			labels, ok := alertMap["labels"].(map[string]any)
			if !ok {
				continue
			}

			if alertName := extractString(labels["alertname"]); alertName != "" {
				names = append(names, alertName)
			}
		}
	}

	return uniqueNonEmpty(names)
}

// alertLabelNameMatchesPredicates mirrors Grafana notification policy semantics: all object_matchers
// are ANDed — positive equals/matches are one =~ (OR inside the regex); each notEquals is a separate !=.
func alertLabelNameMatchesPredicates(name string, predicates []configuration.Predicate) bool {
	if len(predicates) == 0 {
		return true
	}

	combined, hasPositive := combinedPositiveAlertNameRegex(predicates)
	if hasPositive {
		re, err := regexp.Compile(combined)
		if err != nil || !re.MatchString(name) {
			return false
		}
	}

	for _, p := range predicates {
		if p.Type == configuration.PredicateTypeNotEquals && p.Value == name {
			return false
		}
	}

	return true
}

func uniqueNonEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || slices.Contains(filtered, value) {
			continue
		}
		filtered = append(filtered, value)
	}

	return filtered
}
