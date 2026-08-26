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
)

var (
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrUnsupportedWebhookEvent = errors.New("unsupported webhook event")
)

type OrderPaidEvent struct {
	Type string        `json:"type"`
	Data OrderPaidData `json:"data"`
}

type OrderPaidData struct {
	ID           string        `json:"id"`
	Customer     OrderCustomer `json:"customer"`
	Product      OrderProduct  `json:"product"`
	ProductPrice priceJSON     `json:"product_price"`
}

type OrderCustomer struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
}

type OrderProduct struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata"`
	Prices   []priceJSON    `json:"prices"`
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

func VerifyAndParseOrderPaid(headers http.Header, body []byte, secret string) (*OrderPaidEvent, error) {
	if err := verifySignature(headers, body, secret); err != nil {
		return nil, err
	}

	var event OrderPaidEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	if event.Type != orderPaidType {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedWebhookEvent, event.Type)
	}
	if strings.TrimSpace(event.Data.ID) == "" {
		return nil, fmt.Errorf("order id is required")
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
	trimmed = strings.TrimPrefix(trimmed, "whsec_")
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return []byte(trimmed), nil
	}
	return decoded, nil
}

func headerValue(headers http.Header, key string) string {
	return strings.TrimSpace(headers.Get(key))
}
