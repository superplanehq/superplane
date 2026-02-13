package sns

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnTopicMessage__Setup(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("valid configuration -> requests webhook endpoint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetTopicAttributesResponse>
						  <GetTopicAttributesResult>
							<Attributes>
							  <entry><key>DisplayName</key><value>Orders Events</value></entry>
							</Attributes>
						  </GetTopicAttributesResult>
						</GetTopicAttributesResponse>
					`)),
				},
			},
		}

		metadataContext := &contexts.MetadataContext{}
		integration := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:        httpContext,
			Metadata:    metadataContext,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		require.Len(t, integration.WebhookRequests, 1)

		metadata, ok := metadataContext.Metadata.(OnTopicMessageMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", metadata.Region)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", metadata.TopicArn)

		webhookConfig, ok := integration.WebhookRequests[0].(common.WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", webhookConfig.Region)
		assert.Equal(t, common.WebhookTypeSNS, webhookConfig.Type)
		require.NotNil(t, webhookConfig.SNS)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", webhookConfig.SNS.TopicArn)
	})

	t.Run("existing matching metadata -> no subscribe call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		metadataContext := &contexts.MetadataContext{
			Metadata: OnTopicMessageMetadata{
				Region:   "us-east-1",
				TopicArn: "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
		}

		integration := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:        httpContext,
			Metadata:    metadataContext,
			Integration: integration,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 0)
		require.Len(t, integration.WebhookRequests, 0)
	})
}

func Test__OnTopicMessage__HandleWebhook(t *testing.T) {
	trigger := &OnTopicMessage{}

	t.Run("notification for configured topic -> emits event", func(t *testing.T) {
		privateKey, certPEM := createTestSigningCert(t)
		message := signTestMessage(t, trigger, SubscriptionMessage{
			Type:             "Notification",
			MessageID:        "msg-123",
			TopicArn:         "arn:aws:sns:us-east-1:123456789012:orders-events",
			Subject:          "order.created",
			Message:          "{\"orderId\":\"ord_123\"}",
			Timestamp:        "2026-01-10T10:00:00Z",
			SigningCertURL:   testSigningCertURL,
			SignatureVersion: "2",
		}, privateKey)

		body, err := json.Marshal(message)
		require.NoError(t, err)

		eventContext := &contexts.EventContext{}
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(certPEM)),
			}},
		}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			Events: eventContext,
			HTTP:   httpContext,
			Logger: log.NewEntry(log.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, eventContext.Payloads, 1)
		assert.Equal(t, "aws.sns.topic.message", eventContext.Payloads[0].Type)

		payload, ok := eventContext.Payloads[0].Data.(SubscriptionMessage)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", payload.TopicArn)
		assert.Equal(t, "order.created", payload.Subject)
		assert.Equal(t, "{\"orderId\":\"ord_123\"}", payload.Message)
		assert.Equal(t, "2026-01-10T10:00:00Z", payload.Timestamp)
	})

	t.Run("subscription confirmation for different topic -> ignored", func(t *testing.T) {
		privateKey, certPEM := createTestSigningCert(t)
		message := signTestMessage(t, trigger, SubscriptionMessage{
			Type:             "SubscriptionConfirmation",
			MessageID:        "msg-456",
			TopicArn:         "arn:aws:sns:us-east-1:123456789012:different-topic",
			Message:          "confirm",
			SubscribeURL:     "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription",
			Timestamp:        "2026-01-10T10:00:00Z",
			Token:            "token-123",
			SigningCertURL:   testSigningCertURL,
			SignatureVersion: "2",
		}, privateKey)

		body, err := json.Marshal(message)
		require.NoError(t, err)

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(certPEM)),
			}},
		}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			Events: &contexts.EventContext{},
			HTTP:   httpContext,
			Logger: log.NewEntry(log.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("confirmation for configured topic -> confirms subscription", func(t *testing.T) {
		privateKey, certPEM := createTestSigningCert(t)
		message := signTestMessage(t, trigger, SubscriptionMessage{
			Type:             "SubscriptionConfirmation",
			MessageID:        "msg-789",
			TopicArn:         "arn:aws:sns:us-east-1:123456789012:orders-events",
			Message:          "confirm",
			SubscribeURL:     "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription",
			Timestamp:        "2026-01-10T10:00:00Z",
			Token:            "token-456",
			SigningCertURL:   testSigningCertURL,
			SignatureVersion: "2",
		}, privateKey)

		body, err := json.Marshal(message)
		require.NoError(t, err)

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(certPEM)),
			}, {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(``)),
			}},
		}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			HTTP:   httpCtx,
			Events: &contexts.EventContext{},
			Logger: log.NewEntry(log.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, testSigningCertURL, httpCtx.Requests[0].URL.String())
		assert.Equal(t, "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription", httpCtx.Requests[1].URL.String())
	})

	t.Run("unsupported message type -> bad request", func(t *testing.T) {
		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"Type": "UnknownType",
				"TopicArn": "arn:aws:sns:us-east-1:123456789012:orders-events"
			}`),
			Configuration: map[string]any{
				"region":   "us-east-1",
				"topicArn": "arn:aws:sns:us-east-1:123456789012:orders-events",
			},
			Events: &contexts.EventContext{},
			Logger: log.NewEntry(log.New()),
		})

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, status)
	})
}

const testSigningCertURL = "https://sns.us-east-1.amazonaws.com/test.pem"

func createTestSigningCert(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	now := time.Now()
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	return privateKey, certPEM
}

func signTestMessage(t *testing.T, trigger *OnTopicMessage, message SubscriptionMessage, privateKey *rsa.PrivateKey) SubscriptionMessage {
	t.Helper()

	if message.SignatureVersion == "" {
		message.SignatureVersion = "2"
	}

	stringToSign, err := trigger.buildStringToSign(message)
	require.NoError(t, err)

	sum := sha256.Sum256([]byte(stringToSign))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, sum[:])
	require.NoError(t, err)

	message.Signature = base64.StdEncoding.EncodeToString(signature)
	return message
}
