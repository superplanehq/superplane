package sendgrid

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const EmailEventPayloadType = "sendgrid.email.event"

var sendGridEventTypes = []configuration.FieldOption{
	{Label: "Processed", Value: "processed"},
	{Label: "Delivered", Value: "delivered"},
	{Label: "Deferred", Value: "deferred"},
	{Label: "Bounce", Value: "bounce"},
	{Label: "Dropped", Value: "dropped"},
	{Label: "Open", Value: "open"},
	{Label: "Click", Value: "click"},
	{Label: "Spam Report", Value: "spamreport"},
	{Label: "Unsubscribe", Value: "unsubscribe"},
	{Label: "Group Unsubscribe", Value: "group_unsubscribe"},
	{Label: "Group Resubscribe", Value: "group_resubscribe"},
}

type OnEmailEvent struct{}

type OnEmailEventConfiguration struct {
	EventTypes     []string `json:"eventTypes" mapstructure:"eventTypes"`
	CategoryFilter string   `json:"categoryFilter" mapstructure:"categoryFilter"`
}

func (t *OnEmailEvent) Name() string {
	return "sendgrid.onEmailEvent"
}

func (t *OnEmailEvent) Label() string {
	return "On Email Event"
}

func (t *OnEmailEvent) Description() string {
	return "Listen to SendGrid email events (delivered, bounce, open, click)"
}

func (t *OnEmailEvent) Documentation() string {
	return `The On Email Event trigger emits events when SendGrid posts delivery or engagement events to your webhook.

## Use Cases

- **Bounce handling**: Stop sending to bounced addresses and notify your team
- **Delivery confirmations**: Trigger follow-ups when critical notifications are delivered
- **Engagement tracking**: Update CRM records when recipients open or click emails

## Configuration

- **Event Types**: Optional filter for specific SendGrid events (processed, delivered, bounce, open, click, etc.)
- **Category Filter**: Optional category filter (supports ` + "`*`" + ` wildcards)

## Webhook Verification

SuperPlane configures the SendGrid Event Webhook via API and enables Signed Event Webhook by default. The verification key is stored automatically. Verification uses:
- ` + "`X-Twilio-Email-Event-Webhook-Signature`" + ` header
- ` + "`X-Twilio-Email-Event-Webhook-Timestamp`" + ` header
- Raw request body (no transformations)

## Event Data

Each event includes fields such as ` + "`event`" + `, ` + "`email`" + `, ` + "`timestamp`" + `, ` + "`sg_event_id`" + `, ` + "`sg_message_id`" + `, ` + "`category`" + ` and event-specific properties like ` + "`reason`" + `, ` + "`response`" + `, ` + "`url`" + `, ` + "`bounce_classification`" + `.`
}

func (t *OnEmailEvent) Icon() string {
	return "mail"
}

func (t *OnEmailEvent) Color() string {
	return "gray"
}

func (t *OnEmailEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "eventTypes",
			Label:    "Event Types",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			Default: []string{
				"processed",
				"delivered",
				"deferred",
				"bounce",
				"dropped",
				"open",
				"click",
				"spamreport",
				"unsubscribe",
				"group_unsubscribe",
				"group_resubscribe",
			},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: sendGridEventTypes,
				},
			},
			Description: "Only emit events for these SendGrid event types (leave empty for all)",
		},
		{
			Name:        "categoryFilter",
			Label:       "Category Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "*",
			Description: "Optional category filter (supports * wildcards)",
		},
		{},
	}
}

func (t *OnEmailEvent) Setup(ctx core.TriggerContext) error {
	config := OnEmailEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventTypes:     config.EventTypes,
		CategoryFilter: config.CategoryFilter,
	})
}

func (t *OnEmailEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnEmailEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnEmailEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnEmailEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := verifySignedWebhook(ctx); err != nil {
		return http.StatusForbidden, err
	}

	events, err := parseWebhookEvents(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	emitted := 0
	for _, event := range events {
		eventType := extractString(event, "event")
		if eventType == "" {
			continue
		}

		if len(config.EventTypes) > 0 && !slices.Contains(config.EventTypes, eventType) {
			continue
		}

		if !matchesCategoryFilter(config.CategoryFilter, event["category"]) {
			continue
		}

		if err := ctx.Events.Emit(EmailEventPayloadType, event); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
		}
		emitted++
	}

	if emitted == 0 {
		return http.StatusOK, nil
	}

	return http.StatusOK, nil
}

func (t *OnEmailEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

type WebhookConfiguration struct {
	EventTypes     []string `json:"eventTypes"`
	CategoryFilter string   `json:"categoryFilter"`
}

func parseWebhookEvents(body []byte) ([]map[string]any, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty body")
	}

	var events []map[string]any
	if err := json.Unmarshal(body, &events); err == nil {
		return events, nil
	}

	var single map[string]any
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}

	return []map[string]any{single}, nil
}

func extractString(event map[string]any, key string) string {
	raw, ok := event[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func matchesCategoryFilter(filter string, categoryValue any) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	if filter == "*" {
		return true
	}

	categories := normalizeCategories(categoryValue)
	if len(categories) == 0 {
		return false
	}

	for _, filterValue := range splitFilters(filter) {
		isWildcard := strings.ContainsAny(filterValue, "*?")
		for _, category := range categories {
			if isWildcard {
				match, err := path.Match(strings.ToLower(filterValue), strings.ToLower(category))
				if err == nil && match {
					return true
				}
				continue
			}

			if strings.EqualFold(filterValue, category) {
				return true
			}
		}
	}

	return false
}

func splitFilters(filter string) []string {
	parts := strings.Split(filter, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeCategories(value any) []string {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{v}
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok || strings.TrimSpace(s) == "" {
				continue
			}
			result = append(result, s)
		}
		return result
	case []string:
		return v
	default:
		return nil
	}
}

func verifySignedWebhook(ctx core.WebhookRequestContext) error {
	verificationKey := ""
	if ctx.Webhook != nil {
		secret, err := ctx.Webhook.GetSecret()
		if err == nil && len(secret) > 0 {
			verificationKey = strings.TrimSpace(string(secret))
		}
	}

	if verificationKey == "" {
		return nil
	}

	signature := ctx.Headers.Get("X-Twilio-Email-Event-Webhook-Signature")
	timestamp := ctx.Headers.Get("X-Twilio-Email-Event-Webhook-Timestamp")
	if signature == "" || timestamp == "" {
		return fmt.Errorf("missing signature headers")
	}

	publicKey, err := parseSendGridPublicKey(verificationKey)
	if err != nil {
		return fmt.Errorf("invalid verification key: %w", err)
	}

	if err := verifySendGridSignature(publicKey, signature, timestamp, ctx.Body); err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	return nil
}

func parseSendGridPublicKey(value string) (*ecdsa.PublicKey, error) {
	pemValue := strings.TrimSpace(value)
	if !strings.Contains(pemValue, "BEGIN PUBLIC KEY") && !strings.Contains(pemValue, "BEGIN EC PUBLIC KEY") {
		pemValue = "-----BEGIN PUBLIC KEY-----\n" + pemValue + "\n-----END PUBLIC KEY-----"
	}

	block, _ := pem.Decode([]byte(pemValue))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	ecdsaKey, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ECDSA")
	}

	return ecdsaKey, nil
}

func verifySendGridSignature(publicKey *ecdsa.PublicKey, signature, timestamp string, payload []byte) error {
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	r, s, err := parseECDSASignature(sigBytes)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	hasher.Write([]byte(timestamp))
	hasher.Write(payload)
	digest := hasher.Sum(nil)

	if !ecdsa.Verify(publicKey, digest, r, s) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

func parseECDSASignature(signature []byte) (*big.Int, *big.Int, error) {
	var ecdsaSig struct {
		R *big.Int
		S *big.Int
	}
	if _, err := asn1.Unmarshal(signature, &ecdsaSig); err == nil && ecdsaSig.R != nil && ecdsaSig.S != nil {
		return ecdsaSig.R, ecdsaSig.S, nil
	}

	if len(signature)%2 != 0 {
		return nil, nil, fmt.Errorf("invalid signature length")
	}

	half := len(signature) / 2
	r := new(big.Int).SetBytes(signature[:half])
	s := new(big.Int).SetBytes(signature[half:])
	return r, s, nil
}
