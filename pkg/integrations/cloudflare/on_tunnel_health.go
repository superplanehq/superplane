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

const TunnelHealthEventPayloadType = "cloudflare.tunnel.healthEvent"

// tunnelHealthPolicyStatuses are the only values accepted by Cloudflare's
// tunnel_health_event notification policy new_status filter (see API / Terraform provider).
var tunnelHealthPolicyStatuses = []string{
	"TUNNEL_STATUS_TYPE_HEALTHY",
	"TUNNEL_STATUS_TYPE_DEGRADED",
	"TUNNEL_STATUS_TYPE_DOWN",
}

type OnTunnelHealth struct{}

type OnTunnelHealthSpec struct {
	Tunnel    string   `json:"tunnel"`
	NewStatus []string `json:"newStatus"`
}

func (t *OnTunnelHealth) Name() string {
	return "cloudflare.onTunnelHealth"
}

func (t *OnTunnelHealth) Label() string {
	return "On Tunnel Health"
}

func (t *OnTunnelHealth) Description() string {
	return "Trigger when a Cloudflare Tunnel health notification fires for degradation or recovery"
}

func (t *OnTunnelHealth) Documentation() string {
	return `The On Tunnel Health trigger starts a workflow from Cloudflare ` + "`tunnel_health_event`" + ` notifications (Cloudflare Tunnel / cloudflared).

## Use Cases

- **Degraded or down**: React when tunnel connectivity drops below healthy thresholds
- **Recovery**: Resume normal processing when the tunnel returns to a healthy state

## Configuration

- **Tunnel**: Optional. SuperPlane adds this tunnel to the Cloudflare notification policy **and** requires the webhook payload's ` + "`tunnel_id`" + ` to match (case-insensitive), so a mis-scoped or forged request is less likely to start your workflow. Leave **Tunnel** empty to accept any tunnel on the account (policy and webhook checks both allow any tunnel id present in the payload).

> **Note**: Cloudflare's **Send test** notification may include a sample ` + "`tunnel_id`" + ` that does not match your real tunnel. With a tunnel selected, that test delivery will not emit until you clear the tunnel filter or use a payload that matches your tunnel id.

## Webhook Setup

SuperPlane provisions a Cloudflare Alerting webhook destination and a notification policy for ` + "`tunnel_health_event`" + `. Cloudflare signs requests with the generated webhook secret and SuperPlane verifies the ` + "`cf-webhook-auth`" + ` header before emitting an event.

## Workflow execution details

The trigger **Payload** tab (and expressions such as ` + "`$.trigger.data`" + `) use a merged view of the webhook JSON: fields inside ` + "`data`" + ` are combined with top-level fields (for example ` + "`alert_type`" + `, ` + "`account_id`" + `) so everything is available in one object.`
}

func (t *OnTunnelHealth) Icon() string {
	return "activity"
}

func (t *OnTunnelHealth) Color() string {
	return "orange"
}

func (t *OnTunnelHealth) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tunnel",
			Label:       "Tunnel",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional. Scopes the Cloudflare policy and requires matching tunnel_id in the webhook payload (case-insensitive).",
			Placeholder: "Select a tunnel",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "tunnel",
				},
			},
		},
		{
			Name:        "newStatus",
			Label:       "New Status",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     []string{"TUNNEL_STATUS_TYPE_HEALTHY", "TUNNEL_STATUS_TYPE_DEGRADED", "TUNNEL_STATUS_TYPE_DOWN"},
			Description: "Tunnel status values to listen for (Cloudflare policy filter)",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Healthy", Value: "TUNNEL_STATUS_TYPE_HEALTHY"},
						{Label: "Degraded", Value: "TUNNEL_STATUS_TYPE_DEGRADED"},
						{Label: "Down", Value: "TUNNEL_STATUS_TYPE_DOWN"},
					},
				},
			},
		},
	}
}

func (t *OnTunnelHealth) Setup(ctx core.TriggerContext) error {
	spec := OnTunnelHealthSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	normalized, err := normalizeTunnelHealthSpec(spec)
	if err != nil {
		return err
	}

	if err := resolveTunnelHealthTunnelMetadata(ctx, normalized.Tunnel); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(normalized)
}

func resolveTunnelHealthTunnelMetadata(ctx core.TriggerContext, tunnelID string) error {
	tunnelID = strings.TrimSpace(tunnelID)
	if tunnelID == "" || ctx.Metadata == nil {
		return nil
	}

	meta := TunnelNodeMetadata{TunnelName: tunnelID}
	accountID := accountIDFromIntegration(ctx.Integration)
	if strings.Contains(tunnelID, "{{") || strings.Contains(accountID, "{{") || accountID == "" {
		return ctx.Metadata.Set(meta)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	tunnel, err := client.GetCFDTunnel(accountID, tunnelID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel: %w", err)
	}

	meta.TunnelName = tunnel.Name
	if strings.TrimSpace(meta.TunnelName) == "" {
		meta.TunnelName = tunnelID
	}
	return ctx.Metadata.Set(meta)
}

func normalizeTunnelHealthSpec(spec OnTunnelHealthSpec) (OnTunnelHealthSpec, error) {
	if len(spec.NewStatus) == 0 {
		spec.NewStatus = []string{
			"TUNNEL_STATUS_TYPE_HEALTHY",
			"TUNNEL_STATUS_TYPE_DEGRADED",
			"TUNNEL_STATUS_TYPE_DOWN",
		}
	}

	for i, value := range spec.NewStatus {
		normalized, err := normalizeTunnelHealthPolicyStatus(value)
		if err != nil {
			return spec, err
		}
		spec.NewStatus[i] = normalized
	}

	spec.Tunnel = strings.TrimSpace(spec.Tunnel)
	spec.NewStatus = compactStrings(spec.NewStatus)
	return spec, nil
}

// normalizeTunnelHealthPolicyStatus maps UI / legacy human-readable values to Cloudflare's
// tunnel_health_event new_status filter enums.
func normalizeTunnelHealthPolicyStatus(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("newStatus value is empty")
	}

	if slices.Contains(tunnelHealthPolicyStatuses, trimmed) {
		return trimmed, nil
	}

	legacy := map[string]string{
		"healthy":  "TUNNEL_STATUS_TYPE_HEALTHY",
		"degraded": "TUNNEL_STATUS_TYPE_DEGRADED",
		"down":     "TUNNEL_STATUS_TYPE_DOWN",
	}
	if mapped, ok := legacy[strings.ToLower(trimmed)]; ok {
		return mapped, nil
	}

	if strings.EqualFold(trimmed, "inactive") {
		return "", fmt.Errorf("newStatus Inactive is not supported for Cloudflare tunnel_health_event notification filters")
	}

	return "", fmt.Errorf("newStatus must be one of %s (or legacy Healthy, Degraded, Down)", strings.Join(tunnelHealthPolicyStatuses, ", "))
}

func (t *OnTunnelHealth) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnTunnelHealth) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnTunnelHealth) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnTunnelHealth) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
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

	triggerSpec := OnTunnelHealthSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &triggerSpec); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode trigger configuration: %w", err)
	}

	normalizedSpec, err := normalizeTunnelHealthSpec(triggerSpec)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if !tunnelHealthPayloadMatchesSpec(normalizedSpec, payload) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit(TunnelHealthEventPayloadType, tunnelHealthEventPayloadForEmit(payload)); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, nil, nil
}

// tunnelHealthFlattenPayload merges top-level webhook fields with the nested "data" object
// (inner keys win on collision). Cloudflare tunnel health notifications often put tunnel fields
// under "data" while alert_type and account_id sit at the top level.
func tunnelHealthFlattenPayload(payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload)+16)
	for k, v := range payload {
		if k == "data" {
			continue
		}
		out[k] = v
	}
	if inner, ok := payload["data"].(map[string]any); ok && inner != nil {
		for k, v := range inner {
			out[k] = v
		}
	}
	return out
}

func tunnelHealthEventPayloadForEmit(payload map[string]any) map[string]any {
	return tunnelHealthFlattenPayload(payload)
}

func tunnelHealthPayloadMatchesSpec(spec OnTunnelHealthSpec, payload map[string]any) bool {
	data := tunnelHealthFlattenPayload(payload)

	if spec.Tunnel != "" {
		tunnelID := strings.TrimSpace(tunnelHealthStringField(data, "tunnel_id", "tunnelId"))
		if tunnelID == "" || !strings.EqualFold(tunnelID, spec.Tunnel) {
			return false
		}
	}

	rawStatus := tunnelHealthNormalizeNewStatusRaw(tunnelHealthStringField(data, "new_status", "newStatus", "status"))
	normalized := tunnelHealthPayloadStatusForMatch(rawStatus)
	if normalized == "" || !slices.Contains(spec.NewStatus, normalized) {
		return false
	}

	return true
}

func tunnelHealthNormalizeNewStatusRaw(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// e.g. "TUNNEL_STATUS_TYPE_DEGRADED (status change)"
	if i := strings.Index(raw, " "); i > 0 {
		prefix := raw[:i]
		if strings.HasPrefix(prefix, "TUNNEL_STATUS_TYPE_") {
			return prefix
		}
	}
	return raw
}

// tunnelHealthPayloadStatusForMatch maps webhook body status (enum or human-readable) to the same
// policy enum strings stored in OnTunnelHealthSpec.NewStatus.
func tunnelHealthPayloadStatusForMatch(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if slices.Contains(tunnelHealthPolicyStatuses, raw) {
		return raw
	}
	legacy := map[string]string{
		"healthy":  "TUNNEL_STATUS_TYPE_HEALTHY",
		"degraded": "TUNNEL_STATUS_TYPE_DEGRADED",
		"down":     "TUNNEL_STATUS_TYPE_DOWN",
	}
	if mapped, ok := legacy[strings.ToLower(raw)]; ok {
		return mapped
	}
	return ""
}

func tunnelHealthStringField(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return healthAlertFieldString(value)
		}
	}
	return ""
}
