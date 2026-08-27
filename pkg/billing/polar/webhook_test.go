package polar

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__VerifyAndParseOrderPaid(t *testing.T) {
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("webhook-secret"))
	body := orderPaidBody("order_1")
	headers := signedHeaders("msg_1", body, secret)

	event, err := VerifyAndParseOrderPaid(headers, body, secret)
	require.NoError(t, err)
	assert.Equal(t, "order_1", event.Data.ID)
	assert.True(t, event.Data.Product.IsCreditPack())
	assert.Equal(t, int64(2500), event.Data.Product.FaceValueCents())
}

func Test__VerifyAndParseOrderPaidUsesLiteralSecretBytes(t *testing.T) {
	secret := "whsec_not-valid-base64!!"
	body := orderPaidBody("order_literal")
	headers := signedHeaders("msg_literal", body, secret)

	event, err := VerifyAndParseOrderPaid(headers, body, secret)
	require.NoError(t, err)
	assert.Equal(t, "order_literal", event.Data.ID)
}

func Test__VerifyAndParseOrderPaidRejectsStandardWebhooksKeyDerivation(t *testing.T) {
	raw := []byte("webhook-secret")
	secret := "whsec_" + base64.StdEncoding.EncodeToString(raw)
	body := orderPaidBody("order_encoded")
	headers := signedHeadersWithKey("msg_encoded", body, raw)

	_, err := VerifyAndParseOrderPaid(headers, body, secret)
	require.ErrorIs(t, err, ErrInvalidWebhookSignature)

	headers = signedHeaders("msg_encoded", body, secret)
	_, err = VerifyAndParseOrderPaid(headers, body, secret)
	require.NoError(t, err)
}

func Test__VerifyAndParseOrderPaidRejectsEmptySecret(t *testing.T) {
	body := []byte(`{"type":"order.paid","data":{"id":"order_1"}}`)
	headers := signedHeaders("msg_empty", body, "")

	_, err := VerifyAndParseOrderPaid(headers, body, "")
	require.ErrorIs(t, err, ErrWebhookSecretMissing)

	_, err = VerifyAndParseOrderPaid(headers, body, "whsec_")
	require.ErrorIs(t, err, ErrWebhookSecretMissing)

	_, err = VerifyAndParseOrderPaid(headers, body, "polar_whs_")
	require.ErrorIs(t, err, ErrWebhookSecretMissing)
}

func Test__VerifyAndParseOrderPaidRejectsBadSignature(t *testing.T) {
	secret := "whsec_test-secret"
	body := []byte(`{"type":"order.paid","data":{"id":"order_1"}}`)
	headers := signedHeaders("msg_1", body, secret)
	headers.Set(webhookSignatureHeader, "v1,AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")

	_, err := VerifyAndParseOrderPaid(headers, body, secret)
	require.ErrorIs(t, err, ErrInvalidWebhookSignature)
}

func Test__VerifyAndParseOrderEventRefunded(t *testing.T) {
	secret := "whsec_test-secret"
	body := []byte(`{"type":"order.refunded","data":{"id":"order_refund","status":"refunded","refunded_amount":2500}}`)
	headers := signedHeaders("msg_refund", body, secret)

	event, err := VerifyAndParseOrderEvent(headers, body, secret)
	require.NoError(t, err)
	assert.Equal(t, orderRefundedType, event.Type)
	assert.Equal(t, "order_refund", event.Data.ID)
}

func Test__VerifyAndParseOrderPaidIgnoresOtherEvents(t *testing.T) {
	secret := "whsec_test-secret"
	body := []byte(`{"type":"order.updated","data":{"id":"order_1"}}`)
	headers := signedHeaders("msg_2", body, secret)

	_, err := VerifyAndParseOrderPaid(headers, body, secret)
	require.ErrorIs(t, err, ErrUnsupportedWebhookEvent)
}

func orderPaidBody(orderID string) []byte {
	return []byte(fmt.Sprintf(`{
		"type": "order.paid",
		"data": {
			"id": %q,
			"customer": {"id": "cust_1", "external_id": "11111111-1111-1111-1111-111111111111"},
			"product": {
				"id": "prod_1",
				"name": "Hosted credit 25",
				"metadata": {"superplane_credit_pack": "true"},
				"prices": [{"amount_type": "fixed", "price_amount": 2500}]
			}
		}
	}`, orderID))
}

func signedHeaders(msgID string, body []byte, secret string) http.Header {
	key, _ := decodeWebhookSecret(secret)
	return signedHeadersWithKey(msgID, body, key)
}

func signedHeadersWithKey(msgID string, body []byte, key []byte) http.Header {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msgID))
	mac.Write([]byte("."))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	signature := fmt.Sprintf("v1,%s", base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	headers := http.Header{}
	headers.Set(webhookIDHeader, msgID)
	headers.Set(webhookTimestampHeader, timestamp)
	headers.Set(webhookSignatureHeader, signature)
	return headers
}
