package sendgrid

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnEmailEvent__HandleWebhook_Filters(t *testing.T) {
	trigger := &OnEmailEvent{}
	eventsCtx := &contexts.EventContext{}

	body := mustJSON(t, []map[string]any{
		{
			"event":    "delivered",
			"email":    "delivered@example.com",
			"category": []string{"order-confirmation"},
		},
		{
			"event":    "bounce",
			"email":    "bounced@example.com",
			"category": []string{"order-cancelled"},
		},
	})

	status, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:   body,
		Events: eventsCtx,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Configuration: map[string]any{
			"eventTypes": []string{"delivered"},
			"categoryFilter": []map[string]any{
				{
					"type":  configuration.PredicateTypeMatches,
					"value": "order-.*",
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	require.Len(t, eventsCtx.Payloads, 1)
	assert.Equal(t, EmailEventPayloadType, eventsCtx.Payloads[0].Type)
}

func Test__OnEmailEvent__HandleWebhook_Verification(t *testing.T) {
	trigger := &OnEmailEvent{}
	eventsCtx := &contexts.EventContext{}

	body := mustJSON(t, []map[string]any{
		{
			"event": "open",
			"email": "open@example.com",
		},
	})

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	publicKeyPEM := encodePublicKey(t, &privateKey.PublicKey)
	timestamp := "1700000000"
	signature := signPayload(t, privateKey, timestamp, body)

	status, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:   body,
		Events: eventsCtx,
		Headers: http.Header{
			"X-Twilio-Email-Event-Webhook-Signature": []string{signature},
			"X-Twilio-Email-Event-Webhook-Timestamp": []string{timestamp},
		},
		Webhook: &testNodeWebhookContext{secret: []byte(publicKeyPEM)},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	require.Len(t, eventsCtx.Payloads, 1)
}

func Test__OnEmailEvent__HandleWebhook_InvalidSignature(t *testing.T) {
	trigger := &OnEmailEvent{}
	eventsCtx := &contexts.EventContext{}

	body := mustJSON(t, []map[string]any{
		{
			"event": "open",
			"email": "open@example.com",
		},
	})

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	publicKeyPEM := encodePublicKey(t, &privateKey.PublicKey)
	timestamp := "1700000000"
	signature := signPayload(t, privateKey, timestamp, []byte(`[]`))

	status, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:   body,
		Events: eventsCtx,
		Headers: http.Header{
			"X-Twilio-Email-Event-Webhook-Signature": []string{signature},
			"X-Twilio-Email-Event-Webhook-Timestamp": []string{timestamp},
		},
		Webhook: &testNodeWebhookContext{secret: []byte(publicKeyPEM)},
	})

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, status)
	assert.Len(t, eventsCtx.Payloads, 0)
}

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return raw
}

func encodePublicKey(t *testing.T, key *ecdsa.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(key)
	require.NoError(t, err)
	block := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}
	return string(pem.EncodeToMemory(&block))
}

func signPayload(t *testing.T, key *ecdsa.PrivateKey, timestamp string, body []byte) string {
	t.Helper()
	hasher := sha256.New()
	hasher.Write([]byte(timestamp))
	hasher.Write(body)
	digest := hasher.Sum(nil)

	r, s, err := ecdsa.Sign(rand.Reader, key, digest)
	require.NoError(t, err)

	sigBytes, err := asn1.Marshal(struct {
		R, S *big.Int
	}{R: r, S: s})
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(sigBytes)
}

type testNodeWebhookContext struct {
	secret []byte
}

func (t *testNodeWebhookContext) Setup() (string, error) {
	return "", nil
}

func (t *testNodeWebhookContext) GetSecret() ([]byte, error) {
	return t.secret, nil
}

func (t *testNodeWebhookContext) SetSecret(secret []byte) error {
	t.secret = secret
	return nil
}

func (t *testNodeWebhookContext) ResetSecret() ([]byte, []byte, error) {
	return nil, nil, nil
}

func (t *testNodeWebhookContext) GetBaseURL() string {
	return ""
}
