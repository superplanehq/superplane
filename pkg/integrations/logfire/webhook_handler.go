package logfire

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

type LogfireWebhookHandler struct{}

type logfireWebhookConfiguration struct {
	EventType string `json:"eventType"`
	Resource  string `json:"resource"`
}

type LogfireWebhookMetadata struct {
	ManagedChannel       bool   `json:"managedChannel"`
	SupportsWebhookSetup bool   `json:"supportsWebhookSetup"`
	ChannelID            string `json:"channelId,omitempty"`
	ChannelName          string `json:"channelName,omitempty"`
	ChannelsPath         string `json:"channelsPath,omitempty"`
}

func (h *LogfireWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := logfireWebhookConfiguration{}
	if err := decodeAny(a, &configA); err != nil {
		return false, fmt.Errorf("failed to decode current webhook config: %w", err)
	}

	configB := logfireWebhookConfiguration{}
	if err := decodeAny(b, &configB); err != nil {
		return false, fmt.Errorf("failed to decode requested webhook config: %w", err)
	}

	return strings.EqualFold(strings.TrimSpace(configA.EventType), strings.TrimSpace(configB.EventType)) &&
		strings.EqualFold(strings.TrimSpace(configA.Resource), strings.TrimSpace(configB.Resource)), nil
}

func (h *LogfireWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *LogfireWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create Logfire client: %w", err)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook secret: %w", err)
	}
	if len(secret) == 0 {
		secret = []byte(uuid.NewString())
		if err := ctx.Webhook.SetSecret(secret); err != nil {
			return nil, fmt.Errorf("failed to set webhook secret: %w", err)
		}
	}

	channelName := fmt.Sprintf("superplane-webhook-%s", strings.ToLower(ctx.Webhook.GetID()))
	channel, channelsPath, err := client.UpsertAlertChannel(channelName, ctx.Webhook.GetURL(), string(secret))
	if err != nil {
		if isUnsupportedWebhookProvisioningError(err) || isPermissionDeniedWebhookProvisioningError(err) {
			setWebhookSetupSupport(ctx.Integration, false)
			return LogfireWebhookMetadata{
				ManagedChannel:       false,
				SupportsWebhookSetup: false,
			}, nil
		}

		return nil, fmt.Errorf("failed to provision Logfire alert channel: %w", err)
	}

	setWebhookSetupSupport(ctx.Integration, true)
	return LogfireWebhookMetadata{
		ManagedChannel:       true,
		SupportsWebhookSetup: true,
		ChannelID:            channel.ID,
		ChannelName:          channel.Label,
		ChannelsPath:         channelsPath,
	}, nil
}

func (h *LogfireWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := LogfireWebhookMetadata{}
	if err := decodeAny(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return nil
	}

	if !metadata.ManagedChannel || strings.TrimSpace(metadata.ChannelID) == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Logfire client: %w", err)
	}

	err = client.DeleteAlertChannel(metadata.ChannelID, metadata.ChannelsPath)
	if err != nil && !isNotFoundOrUnauthorized(err) && !isUnsupportedWebhookProvisioningError(err) {
		return fmt.Errorf("failed to delete Logfire alert channel: %w", err)
	}

	return nil
}

func isUnsupportedWebhookProvisioningError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusMethodNotAllowed
	}

	return strings.Contains(strings.ToLower(err.Error()), "not supported")
}

func isNotFoundOrUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusUnauthorized
	}

	return false
}

func isPermissionDeniedWebhookProvisioningError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		return false
	}

	body := strings.ToLower(strings.TrimSpace(apiErr.Body))
	return strings.Contains(body, "not enough permissions") || strings.Contains(body, "permission")
}

func decodeAny(value any, target any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func setWebhookSetupSupport(integration core.IntegrationContext, supported bool) {
	if integration == nil {
		return
	}

	metadata := Metadata{
		SupportsWebhookSetup: supported,
		SupportsQueryAPI:     true,
	}

	_ = decodeAny(integration.GetMetadata(), &metadata)
	metadata.SupportsWebhookSetup = supported
	if !metadata.SupportsQueryAPI {
		metadata.SupportsQueryAPI = true
	}

	integration.SetMetadata(metadata)
}
