package sendgrid

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type SendGridWebhookHandler struct{}

func (s *SendGridWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

func (s *SendGridWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	settings := EventWebhookSettings{
		Enabled:          true,
		URL:              ctx.Webhook.GetURL(),
		Processed:        true,
		Delivered:        true,
		Deferred:         true,
		Bounce:           true,
		Dropped:          true,
		Open:             true,
		Click:            true,
		SpamReport:       true,
		Unsubscribe:      true,
		GroupUnsubscribe: true,
		GroupResubscribe: true,
	}

	if err := client.UpdateEventWebhookSettings(settings); err != nil {
		return nil, fmt.Errorf("failed to update SendGrid webhook settings: %w", err)
	}

	publicKey, err := client.EnableEventWebhookSignature()
	if err != nil {
		return nil, fmt.Errorf("failed to enable SendGrid signed webhook: %w", err)
	}
	verificationKey := strings.TrimSpace(publicKey)

	if verificationKey != "" {
		if err := ctx.Integration.SetSecret(webhookVerificationKeySecret, []byte(verificationKey)); err != nil {
			return nil, fmt.Errorf("failed to store integration verification key: %w", err)
		}

		if err := ctx.Webhook.SetSecret([]byte(verificationKey)); err != nil {
			return nil, fmt.Errorf("failed to store webhook verification key: %w", err)
		}
	}

	return nil, nil
}

func (s *SendGridWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	settings, err := client.GetEventWebhookSettings()
	if err != nil {
		return fmt.Errorf("failed to fetch SendGrid webhook settings: %w", err)
	}

	webhookURL := strings.TrimSpace(settings.URL)
	if webhookURL == "" {
		webhookURL = ctx.Webhook.GetURL()
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil || strings.ToLower(parsedURL.Scheme) != "https" {
		return nil
	}

	settings.Enabled = false
	settings.URL = webhookURL

	if err := client.UpdateEventWebhookSettings(*settings); err != nil {
		return fmt.Errorf("failed to disable SendGrid webhook: %w", err)
	}

	return nil
}
