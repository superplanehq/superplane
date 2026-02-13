package sns

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type OnTopicMessageConfiguration struct {
	Region   string `json:"region" mapstructure:"region"`
	TopicArn string `json:"topicArn" mapstructure:"topicArn"`
}

type OnTopicMessageMetadata struct {
	Region   string `json:"region" mapstructure:"region"`
	TopicArn string `json:"topicArn" mapstructure:"topicArn"`
}

type OnTopicMessage struct{}

func (t *OnTopicMessage) Name() string {
	return "aws.sns.onTopicMessage"
}

func (t *OnTopicMessage) Label() string {
	return "SNS • On Topic Message"
}

func (t *OnTopicMessage) Description() string {
	return "Listen to AWS SNS topic notifications"
}

func (t *OnTopicMessage) Documentation() string {
	return `The On Topic Message trigger starts a workflow execution when a message is published to an AWS SNS topic.

## Use Cases

- **Event-driven automation**: React to messages published by external systems
- **Notification processing**: Handle SNS payloads in workflow steps
- **Routing and enrichment**: Trigger downstream workflows based on topic activity

## How it works

During setup, SuperPlane creates a webhook endpoint for this trigger and subscribes it to the selected SNS topic using HTTPS. SNS sends notification payloads to the webhook endpoint, which then emits workflow events.`
}

func (t *OnTopicMessage) Icon() string {
	return "aws"
}

func (t *OnTopicMessage) Color() string {
	return "gray"
}

func (t *OnTopicMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
	}
}

func (t *OnTopicMessage) Setup(ctx core.TriggerContext) error {
	var config OnTopicMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode trigger configuration: %w", err)
	}

	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode trigger metadata: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	topicArn, err := requireTopicArn(config.TopicArn)
	if err != nil {
		return fmt.Errorf("invalid topic ARN: %w", err)
	}

	if metadata.Region == region && metadata.TopicArn == topicArn {
		return nil
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	topic, err := client.GetTopic(topicArn)
	if err != nil {
		return fmt.Errorf("failed to get topic %q in region %q: %w", topicArn, region, err)
	}

	err = ctx.Metadata.Set(OnTopicMessageMetadata{
		Region:   region,
		TopicArn: topicArn,
	})

	if err != nil {
		return fmt.Errorf("failed to persist trigger metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		Region: region,
		Type:   common.WebhookTypeSNS,
		SNS: &common.SNSWebhookConfiguration{
			TopicArn: topic.TopicArn,
		},
	})
}

func (t *OnTopicMessage) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnTopicMessage) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

type SubscriptionMessage struct {
	Type              string                      `json:"Type"`
	MessageID         string                      `json:"MessageId"`
	TopicArn          string                      `json:"TopicArn"`
	Subject           string                      `json:"Subject"`
	Message           string                      `json:"Message"`
	Timestamp         string                      `json:"Timestamp"`
	SignatureVersion  string                      `json:"SignatureVersion"`
	Signature         string                      `json:"Signature"`
	SigningCertURL    string                      `json:"SigningCertURL"`
	UnsubscribeURL    string                      `json:"UnsubscribeURL"`
	SubscribeURL      string                      `json:"SubscribeURL"`
	Token             string                      `json:"Token"`
	MessageAttributes map[string]MessageAttribute `json:"MessageAttributes"`
}

type MessageAttribute struct {
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

func (t *OnTopicMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnTopicMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode trigger configuration: %w", err)
	}

	var message SubscriptionMessage
	if err := json.Unmarshal(ctx.Body, &message); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to decode SNS webhook payload: %w", err)
	}

	if err := t.verifyMessageSignature(ctx, message); err != nil {
		ctx.Logger.Errorf("failed to verify SNS signature: %v", err)
		return http.StatusBadRequest, fmt.Errorf("invalid SNS message signature: %w", err)
	}

	switch message.Type {
	case "SubscriptionConfirmation":
		return t.confirmSubscription(ctx, config, message)

	case "Notification":
		return t.emitTopicNotification(ctx, message, config)

	case "UnsubscribeConfirmation":
		return http.StatusOK, nil

	default:
		return http.StatusBadRequest, fmt.Errorf("unsupported SNS message type %q", message.Type)
	}
}

func (t *OnTopicMessage) confirmSubscription(ctx core.WebhookRequestContext, config OnTopicMessageConfiguration, message SubscriptionMessage) (int, error) {
	if strings.TrimSpace(message.TopicArn) != config.TopicArn {
		ctx.Logger.Infof("message topic ARN %s does not match configured topic ARN %s, ignoring", message.TopicArn, config.TopicArn)
		return http.StatusOK, nil
	}

	if message.SubscribeURL == "" {
		ctx.Logger.Errorf("missing SubscribeURL")
		return http.StatusBadRequest, fmt.Errorf("missing SubscribeURL")
	}

	subscribeURL, err := url.Parse(message.SubscribeURL)
	if err != nil {
		ctx.Logger.Errorf("invalid SubscribeURL: %v", err)
		return http.StatusBadRequest, fmt.Errorf("invalid SubscribeURL: %w", err)
	}

	if subscribeURL.Scheme != "https" {
		ctx.Logger.Errorf("SubscribeURL must use https")
		return http.StatusBadRequest, fmt.Errorf("SubscribeURL must use https")
	}

	host := strings.ToLower(subscribeURL.Hostname())
	if host == "" {
		ctx.Logger.Errorf("SubscribeURL host is required")
		return http.StatusBadRequest, fmt.Errorf("SubscribeURL host is required")
	}

	if !strings.HasSuffix(host, ".amazonaws.com") && !strings.HasSuffix(host, ".amazonaws.com.cn") {
		ctx.Logger.Errorf("SubscribeURL host must be an AWS SNS domain")
		return http.StatusBadRequest, fmt.Errorf("SubscribeURL host must be an AWS SNS domain")
	}

	req, err := http.NewRequest(http.MethodGet, subscribeURL.String(), nil)
	if err != nil {
		ctx.Logger.Errorf("failed to create request to confirm subscription: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := ctx.HTTP.Do(req)
	if err != nil {
		ctx.Logger.Errorf("failed to confirm SNS subscription: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to confirm SNS subscription: %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			ctx.Logger.Errorf("failed to read response body: %v", readErr)
			return http.StatusInternalServerError, fmt.Errorf(
				"SNS subscription confirmation failed with status %d and unreadable body: %v",
				response.StatusCode,
				readErr,
			)
		}

		ctx.Logger.Errorf("SNS subscription confirmation failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
		return http.StatusInternalServerError, fmt.Errorf(
			"SNS subscription confirmation failed with status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	ctx.Logger.Info("Subscription confirmation was successful")
	return http.StatusOK, nil
}

func (t *OnTopicMessage) emitTopicNotification(ctx core.WebhookRequestContext, message SubscriptionMessage, config OnTopicMessageConfiguration) (int, error) {
	topicArn := strings.TrimSpace(message.TopicArn)
	if topicArn == "" {
		ctx.Logger.Errorf("missing TopicArn in SNS notification payload")
		return http.StatusBadRequest, fmt.Errorf("missing TopicArn in SNS notification payload")
	}

	if topicArn != config.TopicArn {
		ctx.Logger.Infof("message topic ARN %s does not match configured topic ARN %s, ignoring", topicArn, config.TopicArn)
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("aws.sns.topic.message", message); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit topic message event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnTopicMessage) Cleanup(ctx core.TriggerContext) error {
	return nil
}

/*
 * Verifies that the message comes from AWS SNS.
 * See: https://docs.aws.amazon.com/sns/latest/dg/sns-verify-signature-of-message-verify-message-signature.html
 */
func (t *OnTopicMessage) verifyMessageSignature(ctx core.WebhookRequestContext, message SubscriptionMessage) error {
	signature, err := base64.StdEncoding.DecodeString(message.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	stringToSign, err := t.buildStringToSign(message)
	if err != nil {
		return fmt.Errorf("failed to build string to sign: %w", err)
	}

	//
	// TODO: it would be good to not fetch the certificate every time.
	//
	cert, err := t.fetchSigningCertificate(ctx, message.SigningCertURL)
	if err != nil {
		return fmt.Errorf("failed to fetch signing certificate: %w", err)
	}

	hash, digest, err := t.getHashAndDigest(message.SignatureVersion, stringToSign)
	if err != nil {
		return fmt.Errorf("failed to get hash and digest: %w", err)
	}

	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("unsupported signing certificate key type %T", cert.PublicKey)
	}

	if err := rsa.VerifyPKCS1v15(publicKey, hash, digest, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

type SignableField struct {
	name  string
	value string
}

func (t *OnTopicMessage) buildStringToSign(message SubscriptionMessage) (string, error) {
	signableFields, err := t.getSignableFields(message)
	if err != nil {
		return "", err
	}

	for _, field := range signableFields {
		if field.value == "" {
			return "", fmt.Errorf("missing %s for SNS signature verification", field.name)
		}
	}

	var builder strings.Builder
	for _, field := range signableFields {
		builder.WriteString(field.name)
		builder.WriteString("\n")
		builder.WriteString(field.value)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func (t *OnTopicMessage) getSignableFields(message SubscriptionMessage) ([]SignableField, error) {
	var fields []SignableField

	switch message.Type {
	case "Notification":
		fields = append(fields, SignableField{"Message", message.Message})
		fields = append(fields, SignableField{"MessageId", message.MessageID})
		if message.Subject != "" {
			fields = append(fields, SignableField{"Subject", message.Subject})
		}
		fields = append(fields, SignableField{"Timestamp", message.Timestamp})
		fields = append(fields, SignableField{"TopicArn", message.TopicArn})
		fields = append(fields, SignableField{"Type", message.Type})
		return fields, nil

	case "SubscriptionConfirmation", "UnsubscribeConfirmation":
		fields = append(fields, SignableField{"Message", message.Message})
		fields = append(fields, SignableField{"MessageId", message.MessageID})
		fields = append(fields, SignableField{"SubscribeURL", message.SubscribeURL})
		fields = append(fields, SignableField{"Timestamp", message.Timestamp})
		fields = append(fields, SignableField{"Token", message.Token})
		fields = append(fields, SignableField{"TopicArn", message.TopicArn})
		fields = append(fields, SignableField{"Type", message.Type})
		return fields, nil

	default:
		return nil, fmt.Errorf("unsupported SNS message type %q", message.Type)
	}
}

func (t *OnTopicMessage) fetchSigningCertificate(ctx core.WebhookRequestContext, signingCertURL string) (*x509.Certificate, error) {
	parsedURL, err := url.Parse(signingCertURL)
	if err != nil {
		return nil, fmt.Errorf("invalid SigningCertURL: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("SigningCertURL must use https")
	}

	host := strings.ToLower(parsedURL.Hostname())
	if host == "" {
		return nil, fmt.Errorf("SigningCertURL host is required")
	}

	if !strings.HasPrefix(host, "sns.") {
		return nil, fmt.Errorf("SigningCertURL host must start with sns")
	}

	if !strings.HasSuffix(host, ".amazonaws.com") && !strings.HasSuffix(host, ".amazonaws.com.cn") {
		return nil, fmt.Errorf("SigningCertURL host must be an AWS SNS domain")
	}

	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate request: %w", err)
	}

	response, err := ctx.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download signing certificate: %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to download signing certificate: status %d with unreadable body: %w", response.StatusCode, readErr)
		}
		return nil, fmt.Errorf("failed to download signing certificate: status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	certBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read signing certificate: %w", err)
	}

	var block *pem.Block
	rest := certBytes
	for {
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			break
		}
	}

	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("SigningCertURL did not return a certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signing certificate: %w", err)
	}

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, fmt.Errorf("signing certificate is not currently valid")
	}

	return cert, nil
}

func (t *OnTopicMessage) getHashAndDigest(signatureVersion, stringToSign string) (crypto.Hash, []byte, error) {
	switch signatureVersion {
	case "1":
		sum := sha1.Sum([]byte(stringToSign))
		return crypto.SHA1, sum[:], nil

	case "2":
		sum := sha256.Sum256([]byte(stringToSign))
		return crypto.SHA256, sum[:], nil

	default:
		return 0, nil, fmt.Errorf("unsupported SignatureVersion %q", signatureVersion)
	}
}
