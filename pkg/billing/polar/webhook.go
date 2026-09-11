package polar

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	webhookIDHeader        = "Webhook-Id"
	webhookTimestampHeader = "Webhook-Timestamp"
	webhookSignatureHeader = "Webhook-Signature"
	signatureTolerance     = 5 * time.Minute
	orderPaidType          = "order.paid"
	orderRefundedType      = "order.refunded"
	billingReasonPurchase  = "purchase"

	subscriptionCreatedType    = "subscription.created"
	subscriptionUpdatedType    = "subscription.updated"
	subscriptionActiveType     = "subscription.active"
	subscriptionCanceledType   = "subscription.canceled"
	subscriptionUncanceledType = "subscription.uncanceled"
	subscriptionRevokedType    = "subscription.revoked"
)

var (
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrUnsupportedWebhookEvent = errors.New("unsupported webhook event")
	ErrWebhookSecretMissing    = errors.New("webhook secret is not configured")
	ErrUnusableWebhookPayload  = errors.New("polar webhook payload cannot be applied")
)

type OrderWebhookEvent struct {
	Type string    `json:"type"`
	Data OrderData `json:"data"`
}

// OrderPaidEvent is the signed order.paid payload. Kept as an alias for apply callers.
type OrderPaidEvent = OrderWebhookEvent

type OrderData struct {
	ID                 string          `json:"id"`
	Status             string          `json:"status"`
	BillingReason      string          `json:"billing_reason"`
	RefundedAmount     int64           `json:"refunded_amount"`
	NetAmount          int64           `json:"net_amount"`
	ProductID          string          `json:"product_id"`
	ExternalCustomerID string          `json:"external_customer_id"`
	Customer           OrderCustomer   `json:"customer"`
	Product            OrderProduct    `json:"product"`
	ProductPrice       priceJSON       `json:"product_price"`
	Items              []orderItemJSON `json:"items"`
}

func (d OrderData) organizationExternalID() string {
	if id := strings.TrimSpace(d.Customer.ExternalID); id != "" {
		return id
	}
	return strings.TrimSpace(d.ExternalCustomerID)
}

func (d OrderData) productID() string {
	if id := strings.TrimSpace(d.Product.ID); id != "" {
		return id
	}
	return strings.TrimSpace(d.ProductID)
}

type OrderPaidData = OrderData

type OrderCustomer struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
}

type OrderProduct struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	IsRecurring bool           `json:"is_recurring"`
	Metadata    map[string]any `json:"metadata"`
	Prices      []priceJSON    `json:"prices"`
}

func (p OrderProduct) FaceValueCents() int64 {
	return productJSON{Prices: p.Prices}.faceValueCents()
}

func (p OrderProduct) IsCreditPack() bool {
	return isCreditPack(p.Metadata)
}

func WebhookSecret() string {
	return strings.TrimSpace(os.Getenv("POLAR_WEBHOOK_SECRET"))
}

func VerifyAndParseOrderPaid(headers http.Header, body []byte, secret string) (*OrderWebhookEvent, error) {
	event, err := VerifyAndParseOrderEvent(headers, body, secret)
	if err != nil {
		return nil, err
	}
	if event.Type != orderPaidType {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedWebhookEvent, event.Type)
	}
	return event, nil
}

type SubscriptionWebhookEvent struct {
	Type string           `json:"type"`
	Data SubscriptionData `json:"data"`
}

type SubscriptionData struct {
	ID                 string        `json:"id"`
	Status             string        `json:"status"`
	CancelAtPeriodEnd  bool          `json:"cancel_at_period_end"`
	CurrentPeriodStart polarTime     `json:"current_period_start"`
	CurrentPeriodEnd   polarTime     `json:"current_period_end"`
	CustomerID         string        `json:"customer_id"`
	ExternalCustomerID string        `json:"external_customer_id"`
	Customer           OrderCustomer `json:"customer"`
}

func (d SubscriptionData) organizationExternalID() string {
	if id := strings.TrimSpace(d.Customer.ExternalID); id != "" {
		return id
	}
	return strings.TrimSpace(d.ExternalCustomerID)
}

type polarTime struct {
	time.Time
}

func (p *polarTime) UnmarshalJSON(raw []byte) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		if strings.TrimSpace(s) == "" {
			return nil
		}
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339Nano, s)
		}
		if err != nil {
			return err
		}
		p.Time = parsed
		return nil
	}
	var unix int64
	if err := json.Unmarshal(raw, &unix); err != nil {
		return err
	}
	if unix <= 0 {
		return nil
	}
	p.Time = time.Unix(unix, 0).UTC()
	return nil
}

type ParsedWebhook struct {
	Type         string
	Order        *OrderWebhookEvent
	Subscription *SubscriptionWebhookEvent
}

func isSubscriptionEventType(eventType string) bool {
	switch eventType {
	case subscriptionCreatedType, subscriptionUpdatedType, subscriptionActiveType,
		subscriptionCanceledType, subscriptionUncanceledType, subscriptionRevokedType:
		return true
	default:
		return false
	}
}

func VerifyAndParseWebhook(headers http.Header, body []byte, secret string) (*ParsedWebhook, error) {
	if _, err := decodeWebhookSecret(secret); err != nil {
		return nil, err
	}
	if err := verifySignature(headers, body, secret); err != nil {
		return nil, err
	}

	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnusableWebhookPayload, err)
	}

	switch {
	case envelope.Type == orderPaidType || envelope.Type == orderRefundedType:
		event, err := parseOrderEvent(body)
		if err != nil {
			return nil, err
		}
		return &ParsedWebhook{Type: event.Type, Order: event}, nil
	case isSubscriptionEventType(envelope.Type):
		event, err := parseSubscriptionEvent(body)
		if err != nil {
			return nil, err
		}
		return &ParsedWebhook{Type: event.Type, Subscription: event}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedWebhookEvent, envelope.Type)
	}
}

func parseOrderEvent(body []byte) (*OrderWebhookEvent, error) {
	var event OrderWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnusableWebhookPayload, err)
	}
	if strings.TrimSpace(event.Data.ID) == "" {
		return nil, fmt.Errorf("%w: order id is required", ErrUnusableWebhookPayload)
	}
	return &event, nil
}

func parseSubscriptionEvent(body []byte) (*SubscriptionWebhookEvent, error) {
	var event SubscriptionWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnusableWebhookPayload, err)
	}
	if strings.TrimSpace(event.Data.ID) == "" {
		return nil, fmt.Errorf("%w: subscription id is required", ErrUnusableWebhookPayload)
	}
	return &event, nil
}

func VerifyAndParseOrderEvent(headers http.Header, body []byte, secret string) (*OrderWebhookEvent, error) {
	parsed, err := VerifyAndParseWebhook(headers, body, secret)
	if err != nil {
		return nil, err
	}
	if parsed.Order == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedWebhookEvent, parsed.Type)
	}
	return parsed.Order, nil
}

func verifySignature(headers http.Header, body []byte, secret string) error {
	msgID := headerValue(headers, webhookIDHeader)
	timestamp := headerValue(headers, webhookTimestampHeader)
	signatures := headerValue(headers, webhookSignatureHeader)
	if msgID == "" || timestamp == "" || signatures == "" {
		return ErrInvalidWebhookSignature
	}

	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrInvalidWebhookSignature
	}
	issuedAt := time.Unix(unix, 0)
	if time.Since(issuedAt) > signatureTolerance || time.Until(issuedAt) > signatureTolerance {
		return ErrInvalidWebhookSignature
	}

	key, err := decodeWebhookSecret(secret)
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msgID))
	mac.Write([]byte("."))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := mac.Sum(nil)

	for _, part := range strings.Split(signatures, " ") {
		version, encoded, ok := strings.Cut(part, ",")
		if !ok || version != "v1" {
			continue
		}
		actual, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		if hmac.Equal(expected, actual) {
			return nil
		}
	}
	return ErrInvalidWebhookSignature
}

func decodeWebhookSecret(secret string) ([]byte, error) {
	trimmed := strings.TrimSpace(secret)
	if webhookSecretMaterial(trimmed) == "" {
		return nil, ErrWebhookSecretMissing
	}
	return []byte(trimmed), nil
}

func webhookSecretMaterial(secret string) string {
	switch {
	case strings.HasPrefix(secret, "whsec_"):
		return strings.TrimPrefix(secret, "whsec_")
	case strings.HasPrefix(secret, "polar_whs_"):
		return strings.TrimPrefix(secret, "polar_whs_")
	default:
		return secret
	}
}

func headerValue(headers http.Header, key string) string {
	return strings.TrimSpace(headers.Get(key))
}
