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
	secret := base64.StdEncoding.EncodeToString([]byte("webhook-secret"))
	body := []byte(`{
		"type": "order.paid",
		"data": {
			"id": "order_1",
			"customer": {"id": "cust_1", "external_id": "11111111-1111-1111-1111-111111111111"},
			"product": {
				"id": "prod_1",
				"name": "Hosted credit 25",
				"metadata": {"superplane_credit_pack": "true"},
				"prices": [{"amount_type": "fixed", "price_amount": 2500}]
			}
		}
	}`)
	headers := signedHeaders("msg_1", body, secret)

	event, err := VerifyAndParseOrderPaid(headers, body, "whsec_"+secret)
	require.NoError(t, err)
	assert.Equal(t, "order_1", event.Data.ID)
	assert.True(t, event.Data.Product.IsCreditPack())
	assert.Equal(t, int64(2500), event.Data.Product.FaceValueCents())
}

func Test__VerifyAndParseOrderPaidRejectsBadSignature(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("webhook-secret"))
	body := []byte(`{"type":"order.paid","data":{"id":"order_1"}}`)
	headers := signedHeaders("msg_1", body, secret)
	headers.Set(webhookSignatureHeader, "v1,AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")

	_, err := VerifyAndParseOrderPaid(headers, body, secret)
	require.ErrorIs(t, err, ErrInvalidWebhookSignature)
}

func Test__VerifyAndParseOrderPaidIgnoresOtherEvents(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("webhook-secret"))
	body := []byte(`{"type":"order.updated","data":{"id":"order_1"}}`)
	headers := signedHeaders("msg_2", body, secret)

	_, err := VerifyAndParseOrderPaid(headers, body, secret)
	require.ErrorIs(t, err, ErrUnsupportedWebhookEvent)
}

func signedHeaders(msgID string, body []byte, secret string) http.Header {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	key, _ := decodeWebhookSecret(secret)
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
