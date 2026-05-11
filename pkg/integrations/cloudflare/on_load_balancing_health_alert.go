package cloudflare

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

const LoadBalancingHealthAlertPayloadType = "cloudflare.loadBalancing.healthAlert"

var (
	loadBalancingHealthValues = []string{"Healthy", "Unhealthy"}
	loadBalancingEventSources = []string{"pool", "origin"}
)

type OnLoadBalancingHealthAlert struct{}

type OnLoadBalancingHealthAlertSpec struct {
	Pool        string   `json:"pool"`
	NewHealth   []string `json:"newHealth"`
	EventSource []string `json:"eventSource"`
}

func (t *OnLoadBalancingHealthAlert) Name() string {
	return "cloudflare.onLoadBalancingHealthAlert"
}

func (t *OnLoadBalancingHealthAlert) Label() string {
	return "On Load Balancing Health Alert"
}

func (t *OnLoadBalancingHealthAlert) Description() string {
	return "Trigger when a Cloudflare load balancing pool or origin health state changes"
}

func (t *OnLoadBalancingHealthAlert) Documentation() string {
	return `The On Load Balancing Health Alert trigger starts a workflow from Cloudflare Load Balancing health notifications.

## Use Cases

- **Pool unhealthy**: React when a load balancer pool becomes unhealthy
- **Origin unhealthy**: Notify or remediate when an endpoint/origin fails health checks
- **Failover awareness**: Detect health changes that cause Cloudflare to route traffic away from unhealthy pools

## Configuration

- **Pool**: Optional pool filter. Leave empty to listen across pools available to the account.
- **New Health**: Health states to listen for. Defaults to ` + "`Unhealthy`" + `.
- **Event Source**: Listen for pool events, origin events, or both.

## Webhook Setup

SuperPlane automatically creates a Cloudflare Alerting webhook destination and notification policy for ` + "`load_balancing_health_alert`" + `. Cloudflare signs requests with the generated webhook secret and SuperPlane verifies the ` + "`cf-webhook-auth`" + ` header before emitting an event.`
}

func (t *OnLoadBalancingHealthAlert) Icon() string {
	return "activity"
}

func (t *OnLoadBalancingHealthAlert) Color() string {
	return "orange"
}

func (t *OnLoadBalancingHealthAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pool",
			Label:       "Pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional load balancing pool to filter alerts by",
			Placeholder: "Select a pool",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pool",
				},
			},
		},
		{
			Name:        "newHealth",
			Label:       "New Health",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     []string{"Unhealthy"},
			Description: "Health states to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Healthy", Value: "Healthy"},
						{Label: "Unhealthy", Value: "Unhealthy"},
					},
				},
			},
		},
		{
			Name:        "eventSource",
			Label:       "Event Source",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     []string{"pool", "origin"},
			Description: "Cloudflare health event sources to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Pool", Value: "pool"},
						{Label: "Origin", Value: "origin"},
					},
				},
			},
		},
	}
}

func (t *OnLoadBalancingHealthAlert) Setup(ctx core.TriggerContext) error {
	spec := OnLoadBalancingHealthAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	normalized, err := normalizeHealthAlertSpec(spec)
	if err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(normalized)
}

func normalizeHealthAlertSpec(spec OnLoadBalancingHealthAlertSpec) (OnLoadBalancingHealthAlertSpec, error) {
	if len(spec.NewHealth) == 0 {
		spec.NewHealth = []string{"Unhealthy"}
	}
	if len(spec.EventSource) == 0 {
		spec.EventSource = []string{"pool", "origin"}
	}

	for i, value := range spec.NewHealth {
		normalized := normalizeOneOf(value, loadBalancingHealthValues)
		if normalized == "" {
			return spec, fmt.Errorf("newHealth must contain only %s", strings.Join(loadBalancingHealthValues, ", "))
		}
		spec.NewHealth[i] = normalized
	}

	for i, value := range spec.EventSource {
		normalized := normalizeOneOf(value, loadBalancingEventSources)
		if normalized == "" {
			return spec, fmt.Errorf("eventSource must contain only %s", strings.Join(loadBalancingEventSources, ", "))
		}
		spec.EventSource[i] = normalized
	}

	spec.Pool = strings.TrimSpace(spec.Pool)
	spec.NewHealth = compactStrings(spec.NewHealth)
	spec.EventSource = compactStrings(spec.EventSource)
	return spec, nil
}

func normalizeOneOf(value string, allowed []string) string {
	trimmed := strings.TrimSpace(value)
	for _, option := range allowed {
		if strings.EqualFold(trimmed, option) {
			return option
		}
	}
	return ""
}

func compactStrings(values []string) []string {
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func (t *OnLoadBalancingHealthAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnLoadBalancingHealthAlert) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnLoadBalancingHealthAlert) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnLoadBalancingHealthAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	secretBytes, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	provided := strings.TrimSpace(headerValue(ctx.Headers, "cf-webhook-auth"))
	if provided == "" {
		return http.StatusUnauthorized, nil, fmt.Errorf("missing cf-webhook-auth header")
	}

	if subtle.ConstantTimeCompare([]byte(provided), secretBytes) != 1 {
		return http.StatusForbidden, nil, fmt.Errorf("invalid cf-webhook-auth header")
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		payload = map[string]any{"raw": string(ctx.Body)}
	}

	if err := ctx.Events.Emit(LoadBalancingHealthAlertPayloadType, payload); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, nil, nil
}

func headerValue(headers http.Header, name string) string {
	if value := headers.Get(name); value != "" {
		return value
	}

	for key, values := range headers {
		if !strings.EqualFold(key, name) || len(values) == 0 {
			continue
		}
		return values[0]
	}

	return ""
}
