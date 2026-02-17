package render

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test__payloadType(t *testing.T) {
	assert.Equal(t, "render.build.ended", payloadType("build_ended"))
	assert.Equal(t, "render.server.failed", payloadType("server_failed"))
	assert.Equal(t, "render.autoscaling.ended", payloadType("autoscaling_ended"))
	assert.Equal(t, "render.event", payloadType(""))
}

func buildSignedHeaders(secret string, body []byte) http.Header {
	return buildSignedHeadersWithTimestamp(secret, body, strconv.FormatInt(time.Now().Unix(), 10))
}

func buildSignedHeadersWithTimestamp(secret string, body []byte, webhookTimestamp string) http.Header {
	webhookID := "msg_2mN8M5S"

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(webhookID))
	h.Write([]byte("."))
	h.Write([]byte(webhookTimestamp))
	h.Write([]byte("."))
	h.Write(body)
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	headers := http.Header{}
	headers.Set("webhook-id", webhookID)
	headers.Set("webhook-timestamp", webhookTimestamp)
	headers.Set("webhook-signature", "v1,"+signature)

	return headers
}
