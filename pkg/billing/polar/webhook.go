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
	ID             string          `json:"id"`
	Status         string          `json:"status"`
	BillingReason  string          `json:"billing_reason"`
	RefundedAmount int64           `json:"refunded_amount"`
	NetAmount      int64           `json:"net_amount"`
	Customer       OrderCustomer   `json:"customer"`
	Product        OrderProduct    `json:"product"`
	ProductPrice   priceJSON       `json:"product_price"`
	Items          []orderItemJSON `json:"items"`
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

func VerifyAndParseOrderEvent(headers http.Header, body []byte, secret string) (*OrderWebhookEvent, error) {
	if _, err := decodeWebhookSecret(secret); err != nil {
		return nil, err
	}
	if err := verifySignature(headers, body, secret); err != nil {
		return nil, err
	}

	var event OrderWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnusableWebhookPayload, err)
	}
	switch event.Type {
	case orderPaidType, orderRefundedType:
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedWebhookEvent, event.Type)
	}
	if strings.TrimSpace(event.Data.ID) == "" {
		return nil, fmt.Errorf("%w: order id is required", ErrUnusableWebhookPayload)
	}
	return &event, nil
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
