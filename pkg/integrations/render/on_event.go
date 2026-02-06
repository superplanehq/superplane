package render

import (
	"crypto/hmac"
	"crypto/sha256"
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

type OnEvent struct{}

type OnEventConfiguration struct {
	EventTypes        []string                  `json:"eventTypes" mapstructure:"eventTypes"`
	ServiceIDFilter   []configuration.Predicate `json:"serviceIdFilter" mapstructure:"serviceIdFilter"`
	ServiceNameFilter []configuration.Predicate `json:"serviceNameFilter" mapstructure:"serviceNameFilter"`
}

var renderEventTypeOptions = []configuration.FieldOption{
	{Label: "Deploy Ended", Value: "deploy_ended"},
	{Label: "Deploy Started", Value: "deploy_started"},
	{Label: "Build Ended", Value: "build_ended"},
	{Label: "Build Started", Value: "build_started"},
	{Label: "Server Failed", Value: "server_failed"},
	{Label: "Server Available", Value: "server_available"},
	{Label: "Service Suspended", Value: "service_suspended"},
	{Label: "Service Resumed", Value: "service_resumed"},
	{Label: "Cron Job Run Ended", Value: "cron_job_run_ended"},
	{Label: "Job Run Ended", Value: "job_run_ended"},
	{Label: "Autoscaling Ended", Value: "autoscaling_ended"},
	{Label: "Deployment Failed", Value: "deployment_failed"},
	{Label: "Deployment Started", Value: "deployment_started"},
	{Label: "Deployment Succeeded", Value: "deployment_succeeded"},
	{Label: "Instance Deactivated", Value: "instance_deactivated"},
	{Label: "Instance Healthy", Value: "instance_healthy"},
	{Label: "Instance Unhealthy", Value: "instance_unhealthy"},
	{Label: "Service Deactivated", Value: "service_deactivated"},
	{Label: "Service Deploy Failed", Value: "service_deploy_failed"},
	{Label: "Service Deploy Started", Value: "service_deploy_started"},
	{Label: "Service Deploy Succeeded", Value: "service_deploy_succeeded"},
	{Label: "Service Live", Value: "service_live"},
	{Label: "Service Pre Deploy Failed", Value: "service_pre_deploy_failed"},
	{Label: "Service Restarted", Value: "service_restarted"},
	{Label: "Service Updated", Value: "service_updated"},
	{Label: "Service Update Failed", Value: "service_update_failed"},
	{Label: "Service Update Started", Value: "service_update_started"},
}

func (t *OnEvent) Name() string {
	return "render.onEvent"
}

func (t *OnEvent) Label() string {
	return "On Event"
}

func (t *OnEvent) Description() string {
	return "Listen to Render deploy and service webhook events"
}

func (t *OnEvent) Documentation() string {
	return `The On Event trigger emits an event when Render sends a webhook notification.

## Use Cases

- **Deploy notifications**: Notify Slack or PagerDuty when deploys succeed/fail or services become unavailable
- **Post-deploy automation**: Trigger smoke tests after successful deploy completion events
- **Operational follow-up**: Run cleanup or notifications for cron/job completion and service suspension events

## Configuration

- **Event Types**: Optional set of event types to listen for. Leave empty to accept all webhook events.
- **Service ID Filter**: Optional predicate filter for ` + "`data.serviceId`" + `.
- **Service Name Filter**: Optional predicate filter for ` + "`data.serviceName`" + `.

## Webhook Verification

Render webhooks are validated using the secret generated when SuperPlane creates the webhook via the Render API. Verification checks:
- ` + "`webhook-id`" + `
- ` + "`webhook-timestamp`" + `
- ` + "`webhook-signature`" + ` (` + "`v1,<base64-signature>`" + `)

## Event Data

The default output emits the webhook payload data, including fields like ` + "`data.id`" + `, ` + "`data.serviceId`" + `, ` + "`data.serviceName`" + `, and ` + "`data.status`" + ` (when present).`
}

func (t *OnEvent) Icon() string {
	return "server"
}

func (t *OnEvent) Color() string {
	return "gray"
}

func (t *OnEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "eventTypes",
			Label:       "Event Types",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     []string{},
			Description: "Optional Render event types to listen for (leave empty for all)",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: renderEventTypeOptions,
				},
			},
		},
		{
			Name:        "serviceIdFilter",
			Label:       "Service ID Filter",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Default:     []map[string]any{},
			Description: "Optional predicate filter for data.serviceId",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "serviceNameFilter",
			Label:       "Service Name Filter",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Default:     []map[string]any{},
			Description: "Optional predicate filter for data.serviceName",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (t *OnEvent) Setup(ctx core.TriggerContext) error {
	config := OnEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(struct{}{})
}

func (t *OnEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := verifyRenderWebhookSignature(ctx); err != nil {
		return http.StatusForbidden, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	eventType := readString(payload["type"])
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing event type")
	}

	if len(config.EventTypes) > 0 && !slices.Contains(config.EventTypes, eventType) {
		return http.StatusOK, nil
	}

	data := readMap(payload["data"])
	serviceID := readString(data["serviceId"])
	serviceName := readString(data["serviceName"])

	if len(config.ServiceIDFilter) > 0 {
		if serviceID == "" || !configuration.MatchesAnyPredicate(config.ServiceIDFilter, serviceID) {
			return http.StatusOK, nil
		}
	}

	if len(config.ServiceNameFilter) > 0 {
		if serviceName == "" || !configuration.MatchesAnyPredicate(config.ServiceNameFilter, serviceName) {
			return http.StatusOK, nil
		}
	}

	if err := ctx.Events.Emit(renderPayloadType(eventType), data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func verifyRenderWebhookSignature(ctx core.WebhookRequestContext) error {
	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return fmt.Errorf("error reading webhook secret")
	}

	if len(secret) == 0 {
		return fmt.Errorf("missing webhook secret")
	}

	webhookID := strings.TrimSpace(ctx.Headers.Get("webhook-id"))
	webhookTimestamp := strings.TrimSpace(ctx.Headers.Get("webhook-timestamp"))
	signatureHeader := strings.TrimSpace(ctx.Headers.Get("webhook-signature"))

	if webhookID == "" || webhookTimestamp == "" || signatureHeader == "" {
		return fmt.Errorf("missing signature headers")
	}

	signatures, err := parseRenderWebhookSignatures(signatureHeader)
	if err != nil {
		return err
	}

	signingKeys := renderSigningKeys(secret)
	payloadPrefix := webhookID + "." + webhookTimestamp + "."
	secretText := []byte(strings.TrimSpace(string(secret)))

	for _, key := range signingKeys {
		h := hmac.New(sha256.New, key)
		h.Write([]byte(payloadPrefix))
		h.Write(ctx.Body)
		if matchesAnyRenderSignature(signatures, h.Sum(nil)) {
			return nil
		}

		// Compatibility fallback for providers that document signature input as:
		// webhook-id.webhook-timestamp.body.webhook-secret
		h = hmac.New(sha256.New, key)
		h.Write([]byte(payloadPrefix))
		h.Write(ctx.Body)
		h.Write([]byte("."))
		h.Write(secretText)
		if matchesAnyRenderSignature(signatures, h.Sum(nil)) {
			return nil
		}
	}

	return fmt.Errorf("invalid signature")
}

func parseRenderWebhookSignatures(headerValue string) ([][]byte, error) {
	trimmed := strings.TrimSpace(headerValue)
	if trimmed == "" {
		return nil, fmt.Errorf("invalid signature header")
	}

	// Format follows Standard Webhooks header values and may include multiple signatures,
	// e.g. "v1,<sig>" or "v1,<sig1> v1,<sig2>".
	rawEntries := strings.Fields(trimmed)
	if len(rawEntries) == 0 {
		rawEntries = []string{trimmed}
	}

	signatures := make([][]byte, 0, len(rawEntries))
	seen := map[string]struct{}{}
	for _, entry := range rawEntries {
		parts := strings.Split(strings.TrimSpace(entry), ",")
		if len(parts) < 2 {
			continue
		}

		version := strings.TrimSpace(parts[0])
		if version != "v1" {
			continue
		}

		for _, encoded := range parts[1:] {
			signature := strings.TrimSpace(encoded)
			if signature == "" {
				continue
			}

			decoded, decodeErr := decodeRenderBase64(signature)
			if decodeErr != nil {
				continue
			}

			key := base64.StdEncoding.EncodeToString(decoded)
			if _, exists := seen[key]; exists {
				continue
			}

			seen[key] = struct{}{}
			signatures = append(signatures, decoded)
		}
	}

	if len(signatures) == 0 {
		return nil, fmt.Errorf("invalid signature")
	}

	return signatures, nil
}

func decodeRenderBase64(value string) ([]byte, error) {
	decoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, decoder := range decoders {
		decoded, err := decoder.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}

	return nil, fmt.Errorf("failed to decode base64 value")
}

func renderSigningKeys(secret []byte) [][]byte {
	trimmedSecret := strings.TrimSpace(string(secret))
	if trimmedSecret == "" {
		return [][]byte{secret}
	}

	keys := [][]byte{[]byte(trimmedSecret)}
	encodedSecret := trimmedSecret
	switch {
	case strings.HasPrefix(trimmedSecret, "whsec_"):
		encodedSecret = strings.TrimPrefix(trimmedSecret, "whsec_")
	case strings.HasPrefix(trimmedSecret, "whsec-"):
		encodedSecret = strings.TrimPrefix(trimmedSecret, "whsec-")
	default:
		return keys
	}

	decodedSecret, err := decodeRenderBase64(encodedSecret)
	if err != nil || len(decodedSecret) == 0 {
		return keys
	}

	key := base64.StdEncoding.EncodeToString(decodedSecret)
	if slices.ContainsFunc(keys, func(existing []byte) bool {
		return base64.StdEncoding.EncodeToString(existing) == key
	}) {
		return keys
	}

	return append(keys, decodedSecret)
}

func matchesAnyRenderSignature(signatures [][]byte, expected []byte) bool {
	return slices.ContainsFunc(signatures, func(signature []byte) bool {
		return hmac.Equal(signature, expected)
	})
}

func renderPayloadType(eventType string) string {
	trimmedEventType := strings.TrimSpace(eventType)
	if trimmedEventType == "" {
		return "render.event"
	}

	parts := strings.Split(trimmedEventType, "_")
	dotCaseParts := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			continue
		}

		dotCaseParts = append(dotCaseParts, strings.ToLower(trimmedPart))
	}

	if len(dotCaseParts) == 0 {
		return "render.event"
	}

	return "render." + strings.Join(dotCaseParts, ".")
}

func readString(value any) string {
	if value == nil {
		return ""
	}

	s, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(s)
}

func readMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return item
}
