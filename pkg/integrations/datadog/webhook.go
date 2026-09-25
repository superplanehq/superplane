package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

const (
	IntegrationWebhookName = "superplane"
	WebhookSecretName      = "webhookToken"
	WebhookHeaderName      = "X-SuperPlane-Token"

	ErrorTrackingAlertEventType = "error_tracking_alert"
	AlertTransitionTriggered    = "Triggered"
)

type WebhookConfiguration struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	EncodeAs      string `json:"encode_as"`
	Payload       string `json:"payload"`
	CustomHeaders string `json:"custom_headers"`
}

func eventsURL(ctx core.SyncContext) string {
	baseURL := strings.TrimSuffix(ctx.BaseURL, "/")
	if ctx.WebhooksBaseURL != "" {
		baseURL = strings.TrimSuffix(ctx.WebhooksBaseURL, "/")
	}
	return fmt.Sprintf("%s/api/v1/integrations/%s/events", baseURL, ctx.Integration.ID().String())
}

func webhookToken(integration core.IntegrationContext) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == WebhookSecretName {
			return strings.TrimSpace(string(secret.Value)), nil
		}
	}

	return "", nil
}

func ensureWebhookToken(integration core.IntegrationContext) (string, error) {
	token, err := webhookToken(integration)
	if err != nil {
		return "", err
	}
	if token != "" {
		return token, nil
	}

	token, err = crypto.Base64String(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate webhook token: %w", err)
	}

	if err := integration.SetSecret(WebhookSecretName, []byte(token)); err != nil {
		return "", fmt.Errorf("failed to store webhook token: %w", err)
	}

	return token, nil
}

func buildWebhookConfiguration(webhookURL, token string) WebhookConfiguration {
	payload, _ := json.Marshal(map[string]string{
		"id":               "$ID",
		"event_type":       "$EVENT_TYPE",
		"title":            "$ALERT_TITLE",
		"body":             "$TEXT_ONLY_MSG",
		"alert_id":         "$ALERT_ID",
		"alert_transition": "$ALERT_TRANSITION",
		"tags":             "$TAGS",
		"link":             "$LINK",
	})
	customHeaders, _ := json.Marshal(map[string]string{
		WebhookHeaderName: token,
	})

	return WebhookConfiguration{
		Name:          IntegrationWebhookName,
		URL:           webhookURL,
		EncodeAs:      "json",
		Payload:       string(payload),
		CustomHeaders: string(customHeaders),
	}
}

func reconcileWebhook(ctx core.SyncContext, client *Client) error {
	token, err := ensureWebhookToken(ctx.Integration)
	if err != nil {
		return err
	}

	config := buildWebhookConfiguration(eventsURL(ctx), token)
	if err := client.UpsertWebhook(config); err != nil {
		return fmt.Errorf("failed to configure Datadog webhook: %w", err)
	}

	return nil
}

func deleteWebhook(client *Client) error {
	return client.DeleteWebhook(IntegrationWebhookName)
}

func verifyWebhookRequest(integration core.IntegrationContext, request *http.Request) error {
	expected, err := webhookToken(integration)
	if err != nil {
		return err
	}
	if expected == "" {
		return fmt.Errorf("missing webhook token")
	}

	received := strings.TrimSpace(request.Header.Get(WebhookHeaderName))
	if received == "" || received != expected {
		return fmt.Errorf("invalid webhook token")
	}

	return nil
}

func (c *Client) UpsertWebhook(config WebhookConfiguration) error {
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook configuration: %w", err)
	}

	updateURL := fmt.Sprintf("%s/api/v1/integration/webhooks/configuration/webhooks/%s", c.BaseURL, config.Name)
	_, err = c.execRequest(http.MethodPut, updateURL, bytes.NewReader(body))
	if err == nil {
		return nil
	}

	createURL := fmt.Sprintf("%s/api/v1/integration/webhooks/configuration/webhooks", c.BaseURL)
	_, err = c.execRequest(http.MethodPost, createURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteWebhook(name string) error {
	url := fmt.Sprintf("%s/api/v1/integration/webhooks/configuration/webhooks/%s", c.BaseURL, name)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}
